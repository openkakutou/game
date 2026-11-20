package game

import (
	"bytes"
	"errors"
	"image/color"
	"io/fs"
	"sort"
	"strings"

	"github.com/openkakutou/sff"
)

// defInfo is what the select screens read of a .def: its [Info] names and
// author, and its [Files] sprite file.
type defInfo struct {
	name, displayName, author, sprite string
}

// display is the name to show: displayname when the .def sets one, else its
// name, else fallback (the folder or file name). MUGEN's .def has displayname
// for the select screens and name for the character's identity; real
// characters such as Kung Fu Man set both (testdata/reference/chars/kfm), and
// roadmap decision 034 takes the screens' name from [Info] name/displayname.
// The folder fallback is OpenKakutou's, for a .def that names nothing.
func (i defInfo) display(fallback string) string {
	if i.displayName != "" {
		return i.displayName
	}
	if i.name != "" {
		return i.name
	}
	return fallback
}

// readDefInfo reads the [Info] and [Files] sections of a .def: section and
// key names are case-insensitive, a value is quoted or ends at a ";" comment,
// and the first occurrence of a key wins.
func readDefInfo(src []byte) defInfo {
	var info defInfo
	section := ""
	for _, raw := range strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(strings.TrimPrefix(raw, "\uFEFF"))
		if strings.HasPrefix(line, "[") {
			end := strings.Index(line, "]")
			if end > 0 {
				section = strings.ToLower(strings.TrimSpace(line[1:end]))
			}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = defValue(value)
		switch section + "." + strings.ToLower(strings.TrimSpace(key)) {
		case "info.name":
			setFirst(&info.name, value)
		case "info.displayname":
			setFirst(&info.displayName, value)
		case "info.author":
			setFirst(&info.author, value)
		case "files.sprite":
			setFirst(&info.sprite, value)
		}
	}
	return info
}

func setFirst(dst *string, value string) {
	if *dst == "" {
		*dst = value
	}
}

// defValue is the value of a .def line: what is inside the first pair of
// double quotes, else the text up to a ";" comment.
func defValue(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, `"`) {
		if end := strings.Index(raw[1:], `"`); end >= 0 {
			return raw[1 : 1+end]
		}
	}
	text, _, _ := strings.Cut(raw, ";")
	return strings.TrimSpace(text)
}

// SortEntries orders a roster by the screenpack's sort rule: by "folder"
// (the entry's folder or file name), "displayName" or "author", ignoring
// case, ties by exact name so the order is the same on every file system.
func SortEntries(es []Entry, by string, descending bool) {
	key := func(e Entry) string {
		switch by {
		case "displayName":
			return strings.ToLower(e.Display)
		case "author":
			return strings.ToLower(e.Author)
		}
		return strings.ToLower(e.Name)
	}
	sort.SliceStable(es, func(i, j int) bool {
		a, b := key(es[i]), key(es[j])
		if a == b {
			a, b = es[i].Name, es[j].Name
		}
		if descending {
			return a > b
		}
		return a < b
	})
}

// gridMove is the cell reached from cursor by an arrow in a grid of n cells,
// columns to a row (0 or more than n: one row). Left and right stay in the
// cursor's row and wrap inside it; up and down stay in its column and wrap
// over the rows that have a cell in that column (the last row may be short).
func gridMove(cursor int, nav Nav, columns, n int) int {
	if n == 0 {
		return cursor
	}
	if columns <= 0 || columns > n {
		columns = n
	}
	row, col := cursor/columns, cursor%columns
	switch nav {
	case NavLeft, NavRight:
		start := row * columns
		length := min(columns, n-start)
		step := 1
		if nav == NavLeft {
			step = -1
		}
		return start + wrap(col, step, length)
	case NavUp, NavDown:
		rows := (n + columns - 1) / columns
		step := 1
		if nav == NavUp {
			step = -1
		}
		for range rows {
			row = wrap(row, step, rows)
			if row*columns+col < n {
				return row*columns + col
			}
		}
	}
	return cursor
}

// The portraits of a character are sprites of its own sprite file, in the
// MUGEN / Ikemen GO convention: group 9000, image 0 is the small one (the
// select grid) and image 1 the large one. Kung Fu Man's own sprite file,
// testdata/reference/chars/kfm/kfm.sff, holds both (25x25 and 120x140).
const (
	portraitGroup      = 9000
	portraitImageSmall = 0
	portraitImageLarge = 1
)

// LoadPortrait decodes a character's small or large portrait from its sprite
// file (a name in fsys), as the match renders the character's sprites: the
// sprite's own palette. It is an error, never a panic, when the character has
// no sprite file or no such portrait.
func LoadPortrait(fsys fs.FS, spriteFile string, large bool) (px []color.RGBA, w, h int, err error) {
	if spriteFile == "" {
		return nil, 0, 0, errors.New("the character has no sprite file")
	}
	data, err := fs.ReadFile(fsys, spriteFile)
	if err != nil {
		return nil, 0, 0, err
	}
	image := portraitImageSmall
	if large {
		image = portraitImageLarge
	}
	return sff.ResolveSpritePixels(bytes.NewReader(data), portraitGroup, image, nil)
}

// SideCursor is one side's cursor on the character select.
type SideCursor struct {
	// Shown is false for a side that does not pick, or has yet to.
	Shown  bool
	Cursor int
}

// SelectCursors are the sides' cursors: the side picking is on the cursor, a
// side that already picked stays on its pick, a side still to pick (or not
// listed by the step) has none.
func (f *Flow) SelectCursors() [2]SideCursor {
	var out [2]SideCursor
	for pos, name := range f.Mode.Flow[f.Step].Sides {
		side := sideIndex(name)
		switch {
		case pos < f.pickPos:
			out[side] = SideCursor{Shown: true, Cursor: f.Picks[side]}
		case pos == f.pickPos:
			out[side] = SideCursor{Shown: true, Cursor: f.Cursor}
		}
	}
	return out
}
