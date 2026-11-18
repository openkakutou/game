package game

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ResolvePath finds rel under base with the cross-platform rules every
// content reader shares: "\" is a separator, each component is matched
// case-insensitively when no exact match exists (so MUGEN/Ikemen content
// works as-is on a case-sensitive file system), absolute paths and paths
// leaving base are rejected, and symbolic links are never followed. It
// returns the on-disk path, or an error when a component is missing.
func ResolvePath(base, rel string) (string, error) {
	clean, err := cleanRel(rel)
	if err != nil {
		return "", err
	}
	current := base
	for _, part := range strings.Split(clean, "/") {
		if part == "." {
			continue
		}
		next, err := matchEntry(current, part)
		if err != nil {
			return "", err
		}
		current = next
	}
	return current, nil
}

// cleanRel turns rel into a clean slash path, resolving ".." textually as
// Windows does, before anything touches the disk: a ".." hidden behind a
// missing folder must not escape. Absolute paths and paths leaving their
// folder are errors.
func cleanRel(rel string) (string, error) {
	slashed := strings.ReplaceAll(rel, `\`, "/")
	if strings.HasPrefix(slashed, "/") || filepath.VolumeName(slashed) != "" {
		return "", fmt.Errorf("path %q must be relative", rel)
	}
	clean := path.Clean(slashed)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("path %q leaves its folder", rel)
	}
	return clean, nil
}

// matchEntry returns the entry of dir called name: the exact spelling when
// present, else the one equal ignoring case. The result carries the on-disk
// spelling, so it is the same on every file system. A symbolic link is an
// error; a missing entry wraps fs.ErrNotExist.
func matchEntry(dir, name string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var match os.DirEntry
	for _, e := range entries {
		if e.Name() == name {
			match = e
			break
		}
		if match == nil && strings.EqualFold(e.Name(), name) {
			match = e
		}
	}
	if match == nil {
		return "", fmt.Errorf("%s: no file %q: %w", dir, name, fs.ErrNotExist)
	}
	candidate := filepath.Join(dir, match.Name())
	if match.Type()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("%s is a symbolic link", candidate)
	}
	return candidate, nil
}

// joinResolved resolves rel against each base in turn and returns the first
// hit. When nothing is found it returns rel joined to the first base, so the
// caller's own open error still reports the path. When rel is rejected
// (symbolic link, absolute, leaving its folder) it returns "", which no
// caller can open.
func joinResolved(bases []string, rel string) string {
	clean, err := cleanRel(rel)
	if err != nil {
		return ""
	}
	for _, base := range bases {
		p, err := ResolvePath(base, rel)
		if err == nil {
			return p
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return ""
		}
	}
	return filepath.Join(bases[0], filepath.FromSlash(clean))
}

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

// ContentFolders returns the chars/ and stages/ folders of root, or an error
// naming the full path of each one that is missing or not a folder.
func ContentFolders(root string) (chars, stages string, err error) {
	var errs []error
	found := map[string]string{}
	for _, name := range []string{"chars", "stages"} {
		p, rerr := ResolvePath(root, name)
		if rerr != nil {
			errs = append(errs, fmt.Errorf("the %s folder is missing: %s", name, filepath.Join(root, name)))
			continue
		}
		st, serr := os.Stat(p)
		if serr != nil || !st.IsDir() {
			errs = append(errs, fmt.Errorf("the %s folder is missing (not a folder): %s", name, p))
			continue
		}
		found[name] = p
	}
	return found["chars"], found["stages"], errors.Join(errs...)
}

// CommonFolder is the common/ folder of root, where state files a character
// does not carry (common1.cns) are searched, the counterpart of Ikemen GO's
// data/ (roadmap decision 031). It may not exist: the state loader then
// reports the missing files.
func CommonFolder(root string) string {
	return joinResolved([]string{root}, "common")
}
