package ui

import "github.com/openkakutou/screenpack"

// menu draws a vertical menu: a box as wide as its widest entry by one row
// per visible entry, placed by the menu's layout, each row spacing units
// high. Entries are aligned inside the box by their style's align, and the
// cursor element is placed inside the active row, behind its text. With
// visibleEntries, the window of rows moves to keep the cursor in view,
// centered on it when it can be.
// shortcut: only the vertical direction; upgrade with the first horizontal menu (options tabs).
func (l *Layout) menu(dst *DrawList, view View, sc scope, m *screenpack.Menu, entries []string, cursor int, parent rect, elapsedMs int) {
	normal, ok := l.styled(sc, m.Style, view)
	if !ok {
		return
	}
	active := normal
	if m.ActiveStyle != "" {
		active, ok = l.styled(sc, m.ActiveStyle, view)
		if !ok {
			return
		}
	}
	var widest float32
	for _, e := range entries {
		widest = max(widest, normal.width(e))
	}
	visible := len(entries)
	if m.VisibleEntries > 0 {
		visible = min(visible, m.VisibleEntries)
	}
	first := max(0, min(cursor-visible/2, len(entries)-visible))
	unit := view.unit()
	row := float32(m.Spacing) * unit
	box := place(parent, &m.Layout, widest, row*float32(visible), unit)
	for i := first; i < first+visible; i++ {
		line := rect{box.x, box.y + float32(i-first)*row, box.w, row}
		style := normal
		if i == cursor {
			style = active
			if m.Cursor != nil {
				l.element(dst, view, sc, m.Cursor, line, elapsedMs)
			}
		}
		left := line.x
		switch style.style.Align {
		case "center":
			left += (line.w - style.width(entries[i])) / 2
		case "right":
			left += line.w - style.width(entries[i])
		}
		style.emit(dst, entries[i], left, line.y+row/2, 1)
	}
}
