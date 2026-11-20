package game

import (
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/openkakutou/game/native/internal/mode"
)

// modesFolder is where the game folder holds its modes (roadmap decision 031).
const modesFolder = "modes"

// DiscoverModes reads every .json or .jsonc file of the game folder's modes
// folder, in file-name order ignoring case. A file that is not a valid mode is
// left out with a notice carrying the mode error, and so is a second file with
// the id of an earlier one. A missing folder is not an error. When no mode has
// the id of a built-in mode (Arcade, Training, Versus, Watch), that built-in mode is added after
// the others, with a notice, so the game always has it.
func DiscoverModes(fsys fs.FS) (modes []mode.Mode, notices []string) {
	dir, err := ResolvePath(fsys, ".", modesFolder)
	if err == nil {
		modes, notices = readModes(fsys, dir)
	}
	builtIns, err := mode.BuiltIn()
	if err != nil {
		panic(fmt.Sprintf("a built-in mode is invalid: %v", err)) // guarded by mode's TestBuiltIn_ListsArcadeVersusThenWatchParsed
	}
	for _, b := range builtIns {
		if slices.ContainsFunc(modes, func(m mode.Mode) bool { return m.ID == b.ID }) {
			continue
		}
		modes = append(modes, b)
		notices = append(notices, fmt.Sprintf("modes/%s.jsonc not found or invalid, using the built-in %s mode", b.ID, b.Name))
	}
	return modes, notices
}

// readModes parses the mode files of dir, in file-name order.
func readModes(fsys fs.FS, dir string) (modes []mode.Mode, notices []string) {
	files, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, []string{fmt.Sprintf("reading %s: %v", dir, err)}
	}
	var names []string
	for _, f := range files {
		ext := strings.ToLower(path.Ext(f.Name()))
		if f.Type().IsRegular() && (ext == ".json" || ext == ".jsonc") {
			names = append(names, f.Name())
		}
	}
	sort.Slice(names, func(i, j int) bool {
		a, b := strings.ToLower(names[i]), strings.ToLower(names[j])
		if a != b {
			return a < b
		}
		return names[i] < names[j]
	})
	seen := map[string]string{} // id -> file
	for _, name := range names {
		file := path.Join(dir, name)
		src, err := fs.ReadFile(fsys, file)
		if err != nil {
			notices = append(notices, fmt.Sprintf("%v, mode left out", err))
			continue
		}
		m, err := mode.Parse(file, src)
		if err != nil {
			notices = append(notices, fmt.Sprintf("%v, mode left out", err))
			continue
		}
		if first, dup := seen[m.ID]; dup {
			notices = append(notices, fmt.Sprintf("%s: id %q is already used by %s, mode left out", file, m.ID, first))
			continue
		}
		seen[m.ID] = file
		modes = append(modes, m)
	}
	return modes, notices
}
