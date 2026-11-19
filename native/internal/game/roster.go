// Package game holds the native game logic that needs no
// window, GL context or SDL: roster discovery, match assembly, the match
// session wrapping the engine, and the screen-flow state machine. Keeping it
// cgo-free means it is unit-testable anywhere; the gfx package only draws
// what this package describes.
package game

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// Entry is one selectable character or stage found in the content.
type Entry struct {
	Name string
	// DefPath is the name of its .def in the content file system.
	DefPath string
}

// DiscoverCharacters lists the characters under root, a folder of fsys: every sub-directory
// "<name>" that contains "<name>/<name>.def" (the MUGEN folder convention).
// A missing root is an error; a root with no characters returns an empty
// list. Entries are sorted by name so the roster order is stable.
func DiscoverCharacters(fsys fs.FS, root string) ([]Entry, error) {
	dirs, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, fmt.Errorf("reading characters folder: %w", err)
	}
	var out []Entry
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		def, err := ResolvePath(fsys, root, d.Name()+"/"+d.Name()+".def")
		if err != nil {
			continue
		}
		if st, err := fs.Stat(fsys, def); err == nil && !st.IsDir() {
			out = append(out, Entry{Name: d.Name(), DefPath: def})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// DiscoverStages lists the stages under root, a folder of fsys: every "*.def" file directly in
// root that declares a [BGdef] section (this tells a stage apart from any
// other .def). Sorted by name.
func DiscoverStages(fsys fs.FS, root string) ([]Entry, error) {
	files, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, fmt.Errorf("reading stages folder: %w", err)
	}
	var out []Entry
	for _, f := range files {
		if !f.Type().IsRegular() || !strings.EqualFold(path.Ext(f.Name()), ".def") {
			continue
		}
		defPath := path.Join(root, f.Name())
		data, err := fs.ReadFile(fsys, defPath)
		if err != nil || !strings.Contains(strings.ToLower(string(data)), "[bgdef]") {
			continue
		}
		out = append(out, Entry{Name: strings.TrimSuffix(f.Name(), path.Ext(f.Name())), DefPath: defPath})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
