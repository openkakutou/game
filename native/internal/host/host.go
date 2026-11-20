// Package host holds what is about the running process and its file system:
// where the game folder is, how it is opened, and how a path typed on the
// command line is read. The core (package game) sees only the fs.FS made here.
package host

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/openkakutou/game/native/internal/game"
)

// ContentRoot is the folder the content lives in: the one holding the
// program, or on macOS the one holding the .app bundle when exePath is
// <X>.app/Contents/MacOS/<program>.
func ContentRoot(exePath string) string {
	dir := filepath.Dir(exePath)
	contentsDir := filepath.Dir(dir)
	app := filepath.Dir(contentsDir)
	if filepath.Base(dir) == "MacOS" && filepath.Base(contentsDir) == "Contents" &&
		strings.EqualFold(filepath.Ext(app), ".app") {
		return filepath.Dir(app)
	}
	return dir
}

// ConfigWriter is the writer of the settings files (game.WriteFunc): it
// stores name, a slash-separated name relative to the game folder root,
// creating its folder.
func ConfigWriter(root string) game.WriteFunc {
	return func(name string, data []byte) error {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		return os.WriteFile(p, data, 0o644)
	}
}

// absDir is the reserved top-level directory under which Open serves the
// whole disk, read-only: "@abs/<volume>/<rest>", where <volume> is the
// drive letter on Windows ("C") and "-" on other systems (rest is then the
// path below "/"). A game folder entry cannot collide with it: "@" is not
// used in the content layout, and a real entry of that name is shadowed.
const absDir = "@abs"

// noVolume is the <volume> of a path with no drive letter.
const noVolume = "-"

// hostFS is the content file system: names relative to the game folder, plus
// the absolute paths of the machine under absDir.
type hostFS struct{ root fs.FS }

// Open is the content file system of the game folder root, in which an
// absolute path of the machine is also reachable, as absName gives it.
func Open(root string) fs.FS {
	return hostFS{root: os.DirFS(root)}
}

// absName is the name under which hostFS serves the absolute OS path p; false
// for a path it cannot serve (a UNC share).
// shortcut: UNC paths (\\host\share) are not served, upgrade if one is used.
func absName(p string) (string, bool) {
	volume := filepath.VolumeName(p)
	rest := strings.TrimPrefix(filepath.ToSlash(p[len(volume):]), "/")
	name := noVolume
	if volume != "" {
		if len(volume) != 2 || volume[1] != ':' {
			return "", false
		}
		name = volume[:1]
	}
	return absDir + "/" + name + "/" + rest, true
}

// split reads name as an absolute-path name: the file system of its volume
// and the name inside it. ok is false for any other name.
func (h hostFS) split(name string) (volumeFS fs.FS, rest string, ok bool) {
	parts := strings.SplitN(name, "/", 3)
	if len(parts) < 2 || parts[0] != absDir {
		return nil, "", false
	}
	if len(parts) == 2 {
		parts = append(parts, ".")
	}
	volumeRoot := "/"
	if parts[1] != noVolume {
		volumeRoot = parts[1] + `:\`
	}
	return os.DirFS(volumeRoot), parts[2], true
}

func (h hostFS) Open(name string) (fs.File, error) {
	if volumeFS, rest, ok := h.split(name); ok {
		return volumeFS.Open(rest)
	}
	return h.root.Open(name)
}

func (h hostFS) Stat(name string) (fs.FileInfo, error) {
	if volumeFS, rest, ok := h.split(name); ok {
		return fs.Stat(volumeFS, rest)
	}
	if name == absDir {
		return dirInfo{name: name[strings.LastIndex(name, "/")+1:]}, nil
	}
	return fs.Stat(h.root, name)
}

func (h hostFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if volumeFS, rest, ok := h.split(name); ok {
		return fs.ReadDir(volumeFS, rest)
	}
	if name == absDir {
		return h.volumes(), nil
	}
	entries, err := fs.ReadDir(h.root, name)
	if err != nil || name != "." {
		return entries, err
	}
	return append(entries, fs.FileInfoToDirEntry(dirInfo{name: absDir})), nil
}

// volumes lists the volumes absDir holds.
func (h hostFS) volumes() []fs.DirEntry {
	if filepath.Separator == '/' {
		return []fs.DirEntry{fs.FileInfoToDirEntry(dirInfo{name: noVolume})}
	}
	var out []fs.DirEntry
	for letter := 'A'; letter <= 'Z'; letter++ {
		if _, err := os.Stat(string(letter) + `:\`); err == nil {
			out = append(out, fs.FileInfoToDirEntry(dirInfo{name: string(letter)}))
		}
	}
	return out
}

// dirInfo is the FileInfo of a directory made up by hostFS.
type dirInfo struct{ name string }

func (d dirInfo) Name() string       { return d.name }
func (d dirInfo) Size() int64        { return 0 }
func (d dirInfo) Mode() fs.FileMode  { return fs.ModeDir | 0o555 }
func (d dirInfo) ModTime() time.Time { return time.Time{} }
func (d dirInfo) IsDir() bool        { return true }
func (d dirInfo) Sys() any           { return nil }

// resolveExistingPrefix resolves the symlinks of the longest existing prefix
// of p and appends the rest as typed: a declaration may name a file only
// through the core's case-insensitive or no-.def lookup (macOS: /var ->
// /private/var), so EvalSymlinks on the whole path fails.
func resolveExistingPrefix(p string) string {
	rest := ""
	current := p
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			return filepath.Join(resolved, rest)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return p
		}
		rest = filepath.Join(filepath.Base(current), rest)
		current = parent
	}
}

// pathFlags take a character or stage declaration, which may be a path.
var pathFlags = []string{"-p1", "-p2", "-s"}

// MapPaths rewrites the absolute paths given in cl as names of the file
// system Open makes: a file name inside the game folder root when the path
// is inside it, else its absDir name. Other values are left as typed.
func MapPaths(cl game.CommandLine, root string) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return
	}
	if resolved, err := filepath.EvalSymlinks(absRoot); err == nil {
		absRoot = resolved
	}
	for _, flag := range pathFlags {
		value := filepath.FromSlash(cl.Flags[flag])
		if !filepath.IsAbs(value) {
			continue
		}
		// Symlinks are resolved here because the core refuses to walk
		// through one, and the path was used as it is before.
		value = resolveExistingPrefix(value)
		rel, err := filepath.Rel(absRoot, value)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			cl.Flags[flag] = filepath.ToSlash(rel)
			continue
		}
		if name, ok := absName(value); ok {
			cl.Flags[flag] = name
		}
	}
}
