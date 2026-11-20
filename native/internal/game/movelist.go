package game

import (
	"strings"
	"unicode"

	"github.com/openkakutou/character/cmd"
)

// The move list screen, opened from the pause menu: it shows the move list of
// one side's character, content as Ikemen GO shows it
// (external/script/menu.lua f_commandlistParse and f_commandlistRender, commit
// 54d4148a728153c2097a0fb899a06812b2d3e193).

// MoveAlign is how a cell of a move list line sits in the list's width.
type MoveAlign int

// Cell alignments: the number of tabs before the cell's text, as
// menu.lua:803-808 reads it (none: left, one: center, more: right).
const (
	MoveLeft MoveAlign = iota
	MoveCenter
	MoveRight
)

// MoveRun is a piece of a cell: some text, or a button icon. Icon is the
// screenpack's button id (format.md, moveList.json buttons: up, down, forward,
// back, a, b, c, x, y, z, start); the layout draws Text instead when the pack
// has no such icon.
type MoveRun struct {
	Text string
	Icon string
}

// MoveCell is the part of a line between two tab groups.
type MoveCell struct {
	Align MoveAlign
	Runs  []MoveRun
}

// MoveLine is one line of the list; a blank line has no cell.
type MoveLine []MoveCell

// noMoveList is the line shown for a character with neither a move list nor
// commands.
var noMoveList = []MoveLine{{{Align: MoveLeft, Runs: []MoveRun{{Text: "No move list"}}}}}

// moveListState is the move list screen's own state.
type moveListState struct {
	open  bool
	side  int
	lines [2][]MoveLine
	top   [2]int // the first line shown, per side
}

// MoveListOpen is whether the move list is shown over the paused match.
func (f *Flow) MoveListOpen() bool { return f.moveList.open }

// MoveListTitle is the name of the character whose list is shown.
func (f *Flow) MoveListTitle() string {
	e := f.lastChars[f.moveList.side]
	if e.Display != "" {
		return e.Display
	}
	return e.Name
}

// MoveListLines are the lines of the list shown.
func (f *Flow) MoveListLines() []MoveLine { return f.moveList.lines[f.moveList.side] }

// MoveListTop is the index of the first line shown.
func (f *Flow) MoveListTop() int { return f.moveList.top[f.moveList.side] }

func (f *Flow) openMoveList() {
	f.moveList = moveListState{open: true, side: f.pauser}
	for i, fighter := range f.Fighters {
		f.moveList.lines[i] = moveLinesOf(fighter)
	}
}

func (f *Flow) navMoveList(n Nav) {
	m := &f.moveList
	switch n {
	case NavUp:
		m.top[m.side] = max(0, m.top[m.side]-1)
	case NavDown:
		last := max(0, len(m.lines[m.side])-f.MoveListWindow)
		m.top[m.side] = min(last, m.top[m.side]+1)
	case NavLeft, NavRight:
		m.side = 1 - m.side
	case NavCancel, NavPause:
		m.open = false
	}
}

// moveLinesOf is a fighter's list: its Ikemen GO move list (index 0, the first
// entry that could be read, as character's Movelist documents), else its
// commands, else a "No move list" line. An empty file counts as none, as
// menu.lua:778 `movelist ~= ”` does.
func moveLinesOf(fighter *Fighter) []MoveLine {
	for _, m := range fighter.Char.Movelists {
		if m.Index != 0 || m.Error != "" {
			continue
		}
		if m.Text != "" {
			return parseMovelist(m.Text)
		}
		break
	}
	if lines := commandLines(fighter.Cmd); len(lines) > 0 {
		return lines
	}
	return noMoveList
}

// parseMovelist splits a move list text into lines as menu.lua:792-814 does:
// trailing whitespace is dropped, the blank line the text's last newline
// leaves is dropped, and each line is cut into cells at its tabs.
func parseMovelist(text string) []MoveLine {
	rows := strings.Split(text, "\n")
	if rows[len(rows)-1] == "" {
		rows = rows[:len(rows)-1]
	}
	lines := make([]MoveLine, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, parseRow(strings.TrimRightFunc(row, unicode.IsSpace)))
	}
	return lines
}

// parseRow cuts a line (without trailing whitespace) into cells: each is some
// tabs and the text up to the next tab (menu.lua:797).
func parseRow(row string) MoveLine {
	var line MoveLine
	for row != "" {
		body := strings.TrimLeft(row, "\t")
		tabs := len(row) - len(body)
		end := strings.IndexByte(body, '\t')
		if end < 0 {
			end = len(body)
		}
		align := MoveLeft
		if tabs == 1 {
			align = MoveCenter
		}
		if tabs > 1 {
			align = MoveRight
		}
		line = append(line, MoveCell{Align: align, Runs: parseRuns(body[:end])})
		row = body[end:]
	}
	return line
}

// parseRuns cuts a cell's text into text and glyph runs. A glyph is a token
// of Ikemen GO's [Glyphs] table, or a <token> tag (menu.lua:736-747): a tag
// that is a color (<#rrggbb>, <#>, </>) is dropped, color is the pack's; any
// other tag is a glyph, drawn as the token's text when the table lacks it
// (menu.lua:898, 921).
// shortcut: the table's tokens are found left to right, longest first, instead
// of by menu.lua:780-791's key-by-key replacement: same result unless a token
// is written inside a tag; upgrade if a movelist needs it.
func parseRuns(s string) []MoveRun {
	var runs []MoveRun
	var text strings.Builder
	flush := func() {
		if text.Len() > 0 {
			runs = append(runs, MoveRun{Text: text.String()})
			text.Reset()
		}
	}
	for i := 0; i < len(s); {
		if s[i] == '<' {
			if tag, size, ok := tagAt(s[i:]); ok {
				flush()
				if !isColorTag(tag) {
					runs = append(runs, glyphRun(tag))
				}
				i += size
				continue
			}
		}
		if key, ok := glyphKeyAt(s[i:]); ok {
			flush()
			runs = append(runs, glyphRun(key))
			i += len(key)
			continue
		}
		text.WriteByte(s[i])
		i++
	}
	flush()
	return runs
}

var tagEntities = strings.NewReplacer("&lt;", "<", "&gt;", ">", "&amp;", "&")

// tagAt reads a <token> at the start of s (menu.lua:736 `<([^<>%s]+)>`): the
// token with its entities unescaped (menu.lua:740), and the tag's length.
func tagAt(s string) (token string, size int, ok bool) {
	end := strings.IndexByte(s, '>')
	if end < 2 {
		return "", 0, false
	}
	inner := s[1:end]
	if strings.ContainsFunc(inner, func(r rune) bool { return r == '<' || unicode.IsSpace(r) }) {
		return "", 0, false
	}
	return tagEntities.Replace(inner), end + 1, true
}

// isColorTag is whether a tag sets or resets the color (menu.lua:741-747).
func isColorTag(token string) bool {
	if token == "/" || token == "#" {
		return true
	}
	rest, isHex := strings.CutPrefix(token, "#")
	return isHex && strings.IndexFunc(rest, func(r rune) bool {
		return !(r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
	}) < 0
}

// glyphKeyAt is the longest token of the [Glyphs] table s starts with.
func glyphKeyAt(s string) (string, bool) {
	if s[0] != '^' && s[0] != '_' && s[0] != '~' {
		return "", false
	}
	for _, key := range glyphKeys {
		if strings.HasPrefix(s, key) {
			return key, true
		}
	}
	return "", false
}

// glyphRun is the run for a glyph token: the button icon it stands for, with
// its plain text as the fallback; a token with no icon is its plain text.
// shortcut: the plain text is the token without its leading ^ or _ marker
// (QCF, DF, ...), the table having no text form of its own besides a comment;
// upgrade if the screenpack format gets icons for motions.
func glyphRun(token string) MoveRun {
	text := token
	if strings.HasPrefix(token, "^") || strings.HasPrefix(token, "_") {
		text = token[1:]
	}
	return MoveRun{Text: text, Icon: glyphIcons[token]}
}

// commandLines lists the commands of a .cmd that take at least two inputs, in
// file order, each name once: the name, then its inputs (commandRuns).
func commandLines(commands cmd.CommandFile) []MoveLine {
	var lines []MoveLine
	seen := map[string]bool{}
	for _, c := range commands.Commands {
		inputs := strings.Split(c.Input, ",")
		if c.Name == "" || seen[c.Name] || len(inputs) < 2 {
			continue
		}
		seen[c.Name] = true
		var runs []MoveRun
		for i, input := range inputs {
			if i > 0 {
				runs = append(runs, MoveRun{Text: " "})
			}
			runs = append(runs, commandRuns(strings.TrimSpace(input))...)
		}
		lines = append(lines, MoveLine{
			{Align: MoveLeft, Runs: []MoveRun{{Text: c.Name}}},
			{Align: MoveRight, Runs: runs},
		})
	}
	return lines
}

// commandIcons are the .cmd inputs that have a button icon.
var commandIcons = map[string]string{
	"U": "up", "D": "down", "F": "forward", "B": "back",
	"a": "a", "b": "b", "c": "c", "x": "x", "y": "y", "z": "z", "s": "start",
}

// commandRuns is one .cmd input: the modifiers before it (~ release, / hold,
// $ and > as text, with ~'s time), then its buttons and directions, each an
// icon when it has one, joined by "+". A diagonal (DF...) has no icon: text.
func commandRuns(input string) []MoveRun {
	core := strings.TrimLeft(input, "~/$>0123456789")
	var runs []MoveRun
	if prefix := input[:len(input)-len(core)]; prefix != "" {
		runs = append(runs, MoveRun{Text: prefix})
	}
	for i, part := range strings.Split(core, "+") {
		if i > 0 {
			runs = append(runs, MoveRun{Text: "+"})
		}
		runs = append(runs, MoveRun{Text: part, Icon: commandIcons[part]})
	}
	return runs
}
