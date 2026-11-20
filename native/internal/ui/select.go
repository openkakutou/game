package ui

import (
	"github.com/openkakutou/game/native/internal/game"
	"github.com/openkakutou/screenpack"
)

// previewPlaceholder fills the stage preview.
// shortcut: the format has no stage preview and MUGEN / Ikemen GO have no
// such convention, and drawing the stage's own background would load and lay
// out every stage (.def, sprite file, layers) on the select screen; so the
// preview rect is a plain panel. Upgrade by drawing the hovered stage's
// layers into the preview rect, with the same code as the match.
const previewPlaceholder = "#1d1c2a"

// SideSelect is one side's cursor on the character select.
type SideSelect struct {
	// Shown is false for a side that does not pick (yet): no cursor, no portrait.
	Shown bool
	// Cursor is the cell the side hovers, or has picked.
	Cursor int
}

// SelectState is what the game shows on the character select.
type SelectState struct {
	Title  string
	Roster []game.Entry
	// Cells are the roster indexes the grid shows, in order: a mode can hide
	// characters. A cell's portrait is its roster index's.
	Cells []int
	Sides [2]SideSelect
	// Focus is the cell the grid keeps in view when it scrolls: the cell of
	// the side that is picking.
	Focus     int
	Portraits PortraitSizes
	ElapsedMs int
}

// CharacterSelect adds the character select to dst: background, title, the
// grid of cells with their small portraits and the sides' cursors, then each
// shown side's large portrait and name, foreground.
// The random cell, when the grid has one, follows the last character.
// shortcut: the palette part is not drawn; the game has no palette choice yet.
func (l *Layout) CharacterSelect(dst *DrawList, view View, s SelectState) {
	c := l.pack.resolved.Screens.CharacterSelect
	screen, from := view.rect(), len(dst.Ops)
	l.elements(dst, view, l.pack.scope("characterSelect", ""), c.Background, screen, s.ElapsedMs)
	if c.Title != nil {
		l.text(dst, view, l.pack.scope("characterSelect", "title"), &c.Title.Layout, c.Title.Style, s.Title, screen, 1, "")
	}
	if c.Grid != nil {
		l.grid(dst, view, c, s, screen)
	}
	for side, shown := range s.Sides {
		if !shown.Shown || shown.Cursor >= len(s.Cells) {
			continue
		}
		if c.Portrait != nil {
			lay, mirror := c.Portrait.P1, false
			if side == 1 {
				lay, mirror = c.Portrait.P2, c.Portrait.MirrorP2 == nil || *c.Portrait.MirrorP2
			}
			l.portrait(dst, view, lay, s.Portraits, s.Cells[shown.Cursor], mirror, screen)
		}
		if c.Name != nil {
			part := c.Name.P1
			if side == 1 {
				part = c.Name.P2
			}
			if part != nil {
				l.text(dst, view, l.pack.scope("characterSelect", "name"), &part.Layout, part.Style, s.Roster[s.Cells[shown.Cursor]].Display, screen, 1, "")
			}
		}
	}
	l.elements(dst, view, l.pack.scope("characterSelect", ""), c.Foreground, screen, s.ElapsedMs)
	fadeIn(dst, from, &c.Screen, s.ElapsedMs)
}

// grid draws the grid's slots: a cell and its small portrait for each
// character, an empty cell for the slots left. The rows shown are the
// window that keeps the focused cell in view, centered on it when it can be.
func (l *Layout) grid(dst *DrawList, view View, c *screenpack.CharacterSelect, s SelectState, screen rect) {
	g := c.Grid
	if g.Columns < 1 || g.Rows < 1 || g.CellSize == nil {
		return
	}
	unit := view.unit()
	cell := rect{w: float32(g.CellSize[0]) * unit, h: float32(g.CellSize[1]) * unit}
	var gap [2]float32
	if g.Spacing != nil {
		gap = [2]float32{float32(g.Spacing[0]) * unit, float32(g.Spacing[1]) * unit}
	}
	box := place(screen, &g.Layout,
		float32(g.Columns)*cell.w+float32(g.Columns-1)*gap[0],
		float32(g.Rows)*cell.h+float32(g.Rows-1)*gap[1], unit)
	count := len(s.Cells)
	random := g.Random != nil && count > 0
	if random {
		count++ // the random cell follows the characters (Flow.SelectCells)
	}
	totalRows := (count + g.Columns - 1) / g.Columns
	first := max(0, min(s.Focus/g.Columns-g.Rows/2, totalRows-g.Rows))
	sc := l.pack.scope("characterSelect", "grid")
	slot := func(index int) (rect, bool) {
		row := index/g.Columns - first
		if row < 0 || row >= g.Rows {
			return rect{}, false
		}
		return rect{box.x + float32(index%g.Columns)*(cell.w+gap[0]), box.y + float32(row)*(cell.h+gap[1]), cell.w, cell.h}, true
	}
	for row := range g.Rows {
		for col := range g.Columns {
			index := (first+row)*g.Columns + col
			at, _ := slot(index)
			if random && index == len(s.Cells) {
				if g.Cell != nil {
					l.element(dst, view, sc, g.Cell, at, s.ElapsedMs)
				}
				l.element(dst, view, sc, g.Random, at, s.ElapsedMs)
				continue
			}
			if index >= len(s.Cells) {
				if g.EmptyCell != nil {
					l.element(dst, view, sc, g.EmptyCell, at, s.ElapsedMs)
				}
				continue
			}
			if g.Cell != nil {
				l.element(dst, view, sc, g.Cell, at, s.ElapsedMs)
			}
			l.smallPortrait(dst, s.Portraits, s.Cells[index], at)
		}
	}
	l.cursors(dst, view, c, s, slot)
}

// smallPortrait draws a character's small portrait fitted into its cell,
// keeping its aspect, centered; a character without one leaves the cell alone.
func (l *Layout) smallPortrait(dst *DrawList, ps PortraitSizes, character int, cell rect) {
	if ps == nil {
		return
	}
	w, h, ok := ps.PortraitSize(character, false)
	if !ok {
		return
	}
	scale := min(cell.w/float32(w), cell.h/float32(h))
	fit := rect{w: float32(w) * scale, h: float32(h) * scale}
	fit.x, fit.y = cell.x+(cell.w-fit.w)/2, cell.y+(cell.h-fit.h)/2
	dst.Ops = append(dst.Ops, Op{
		Kind: OpPortrait, X: fit.x, Y: fit.y, W: fit.w, H: fit.h, Color: Color{1, 1, 1, 1},
		Portrait: PortraitRef{Character: character}, U1: 1, V1: 1,
	})
}

// cursors draws each shown side's cursor element in its cell, or the both
// element once when the two sides are on the same cell.
func (l *Layout) cursors(dst *DrawList, view View, c *screenpack.CharacterSelect, s SelectState, slot func(int) (rect, bool)) {
	if c.Cursor == nil {
		return
	}
	sc := l.pack.scope("characterSelect", "cursor")
	if s.Sides[0].Shown && s.Sides[1].Shown && s.Sides[0].Cursor == s.Sides[1].Cursor && c.Cursor.Both != nil {
		if at, ok := slot(s.Sides[0].Cursor); ok {
			l.element(dst, view, sc, c.Cursor.Both, at, s.ElapsedMs)
		}
		return
	}
	for side, element := range [2]*screenpack.Element{c.Cursor.P1, c.Cursor.P2} {
		if !s.Sides[side].Shown || element == nil {
			continue
		}
		if at, ok := slot(s.Sides[side].Cursor); ok {
			l.element(dst, view, sc, element, at, s.ElapsedMs)
		}
	}
}

// portrait draws a character's large portrait placed by a layout whose size
// keeps the portrait's aspect; mirror flips it. Nothing when the character
// has no portrait.
func (l *Layout) portrait(dst *DrawList, view View, lay *screenpack.Layout, ps PortraitSizes, character int, mirror bool, parent rect) {
	if ps == nil || lay == nil {
		return
	}
	w, h, ok := ps.PortraitSize(character, true)
	if !ok {
		return
	}
	bw, bh := boxSize(lay.Size, float32(w), float32(h), view)
	box := place(parent, lay, bw, bh, view.unit())
	u0, u1 := float32(0), float32(1)
	if mirror {
		u0, u1 = 1, 0
	}
	dst.Ops = append(dst.Ops, Op{
		Kind: OpPortrait, X: box.x, Y: box.y, W: box.w, H: box.h, Color: Color{1, 1, 1, 1},
		Portrait: PortraitRef{Character: character, Large: true}, U0: u0, U1: u1, V1: 1,
	})
}

// StageSelectState is what the game shows on the stage select.
type StageSelectState struct {
	Title  string
	Roster []game.Entry
	// Cells are the roster indexes the screen steps over, in order; Cursor
	// is a position in it.
	Cells     []int
	Cursor    int
	ElapsedMs int
}

// StageSelect adds the stage select to dst: background, title, the preview,
// the hovered stage's name and the arrows toward the other stages.
func (l *Layout) StageSelect(dst *DrawList, view View, s StageSelectState) {
	st := l.pack.resolved.Screens.StageSelect
	screen, from := view.rect(), len(dst.Ops)
	l.elements(dst, view, l.pack.scope("stageSelect", ""), st.Background, screen, s.ElapsedMs)
	if st.Title != nil {
		l.text(dst, view, l.pack.scope("stageSelect", "title"), &st.Title.Layout, st.Title.Style, s.Title, screen, 1, "")
	}
	if st.Preview != nil && st.Preview.Size != nil && st.Preview.Size[0] != nil && st.Preview.Size[1] != nil {
		unit := view.unit()
		box := place(screen, st.Preview, float32(*st.Preview.Size[0])*unit, float32(*st.Preview.Size[1])*unit, unit)
		dst.Ops = append(dst.Ops, Op{Kind: OpRect, X: box.x, Y: box.y, W: box.w, H: box.h, Color: parseColor(previewPlaceholder, 1)})
	}
	if st.Name != nil && s.Cursor < len(s.Cells) {
		l.text(dst, view, l.pack.scope("stageSelect", "name"), &st.Name.Layout, st.Name.Style, s.Roster[s.Cells[s.Cursor]].Display, screen, 1, "")
	}
	if st.Arrows != nil && len(s.Cells) > 1 {
		sc := l.pack.scope("stageSelect", "arrows")
		if st.Arrows.Left != nil {
			l.element(dst, view, sc, st.Arrows.Left, screen, s.ElapsedMs)
		}
		if st.Arrows.Right != nil {
			l.element(dst, view, sc, st.Arrows.Right, screen, s.ElapsedMs)
		}
	}
	l.elements(dst, view, l.pack.scope("stageSelect", ""), st.Foreground, screen, s.ElapsedMs)
	fadeIn(dst, from, &st.Screen, s.ElapsedMs)
}
