package game

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

// ResolvePath finds rel under base, a folder name of fsys ("." for its
// root), with the cross-platform rules every
// content reader shares: "\" is a separator, each component is matched
// case-insensitively when no exact match exists (so MUGEN/Ikemen content
// works as-is on a case-sensitive file system), absolute paths and paths
// leaving base are rejected, and symbolic links are never followed. It
// returns the name in fsys, with the spelling stored there, or an error when
// a component is missing.
func ResolvePath(fsys fs.FS, base, rel string) (string, error) {
	clean, err := cleanRel(rel)
	if err != nil {
		return "", err
	}
	current := base
	for _, part := range strings.Split(clean, "/") {
		if part == "." {
			continue
		}
		next, err := matchEntry(fsys, current, part)
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
	if strings.HasPrefix(slashed, "/") || hasDriveLetter(slashed) {
		return "", fmt.Errorf("path %q must be relative", rel)
	}
	clean := path.Clean(slashed)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("path %q leaves its folder", rel)
	}
	return clean, nil
}

// hasDriveLetter is true for a Windows volume prefix such as "C:".
func hasDriveLetter(p string) bool {
	if len(p) < 2 || p[1] != ':' {
		return false
	}
	c := p[0] | 0x20
	return c >= 'a' && c <= 'z'
}

// matchEntry returns the entry of dir called name: the exact spelling when
// present, else the one equal ignoring case. The result carries the stored
// spelling, so it is the same on every file system. A symbolic link is an
// error; a missing entry wraps fs.ErrNotExist.
func matchEntry(fsys fs.FS, dir, name string) (string, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return "", err
	}
	var match fs.DirEntry
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
	candidate := path.Join(dir, match.Name())
	if match.Type()&fs.ModeSymlink != 0 {
		return "", fmt.Errorf("%s is a symbolic link", candidate)
	}
	return candidate, nil
}

// joinResolved resolves rel against each base (folder names of fsys) in turn and returns the first
// hit. When nothing is found it returns rel joined to the first base, so the
// caller's own open error still reports the path. When rel is rejected
// (symbolic link, absolute, leaving its folder) it returns "", which no
// caller can open.
func joinResolved(fsys fs.FS, bases []string, rel string) string {
	clean, err := cleanRel(rel)
	if err != nil {
		return ""
	}
	for _, base := range bases {
		p, err := ResolvePath(fsys, base, rel)
		if err == nil {
			return p
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return ""
		}
	}
	return path.Join(bases[0], clean)
}

// ContentFolders returns the names of the chars/ and stages/ folders of
// fsys, or an error naming each one that is missing or not a folder. label is
// where fsys is, as the player knows it (the game folder), for the messages.
func ContentFolders(fsys fs.FS, label string) (chars, stages string, err error) {
	var errs []error
	found := map[string]string{}
	for _, name := range []string{"chars", "stages"} {
		p, rerr := ResolvePath(fsys, ".", name)
		if rerr != nil {
			errs = append(errs, fmt.Errorf("the %s folder is missing: %s", name, filepath.Join(label, name)))
			continue
		}
		st, serr := fs.Stat(fsys, p)
		if serr != nil || !st.IsDir() {
			errs = append(errs, fmt.Errorf("the %s folder is missing (not a folder): %s", name, filepath.Join(label, p)))
			continue
		}
		found[name] = p
	}
	return found["chars"], found["stages"], errors.Join(errs...)
}

// CommonFolder is the common/ folder of fsys, where state files a character
// does not carry (common1.cns) are searched, the counterpart of Ikemen GO's
// data/ (roadmap decision 031). It may not exist: the state loader then
// reports the missing files.
func CommonFolder(fsys fs.FS) string {
	return joinResolved(fsys, []string{"."}, "common")
}
