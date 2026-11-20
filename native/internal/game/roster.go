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
	// Name is the folder name of a character, the file name of a stage.
	Name string
	// DefPath is the name of its .def in the content file system.
	DefPath string
	// Display is the name shown on the select screens: the .def [Info]
	// displayname, else its name, else Name (see readDefInfo).
	Display string
	// Author is the .def [Info] author, "" when it has none.
	Author string
	// Sprite is the name of the character's sprite file in the content file
	// system, "" when its .def names none.
	Sprite string
}

// DiscoverCharacters lists the characters under root, a folder of fsys: every sub-directory
// "<name>" that contains "<name>/<name>.def" (the MUGEN folder convention).
// A missing root is an error; a root with no characters returns an empty
// list. Entries are sorted by name (see sortByName).
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
			out = append(out, characterEntry(fsys, d.Name(), def))
		}
	}
	sortByName(out)
	return out, nil
}

// DiscoverStages lists the stages under root, a folder of fsys: every "*.def" file directly in
// root that declares a [BGdef] section (this tells a stage apart from any
// other .def). Sorted by name (see sortByName).
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
		name := strings.TrimSuffix(f.Name(), path.Ext(f.Name()))
		info := readDefInfo(data)
		out = append(out, Entry{Name: name, DefPath: defPath, Display: info.display(name), Author: info.author})
	}
	sortByName(out)
	return out, nil
}

// sortByName orders entries by folder or file name ignoring case (the
// default select-screen order, roadmap decision 029), exact name as tie-break
// so the order stays stable on case-sensitive file systems.
func sortByName(es []Entry) {
	sort.Slice(es, func(i, j int) bool {
		a, b := strings.ToLower(es[i].Name), strings.ToLower(es[j].Name)
		if a != b {
			return a < b
		}
		return es[i].Name < es[j].Name
	})
}

// characterEntry builds the entry of the character folder name, whose .def is def.
func characterEntry(fsys fs.FS, name, def string) Entry {
	e := Entry{Name: name, DefPath: def, Display: name}
	data, err := fs.ReadFile(fsys, def)
	if err != nil {
		return e
	}
	info := readDefInfo(data)
	e.Display, e.Author = info.display(name), info.author
	if info.sprite != "" {
		e.Sprite = joinResolved(fsys, []string{path.Dir(def)}, info.sprite)
	}
	return e
}
