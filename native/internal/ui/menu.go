package ui

import "github.com/openkakutou/screenpack"

// Row is one menu entry: its label, up to two values drawn in columns to the
// right of it (a setting's value; a binding's key then button) and whether
// the game flags it as a warning.
type Row struct {
	Label   string
	Values  [2]string
	Flagged bool
}

// slots styles the values of a menu whose rows have two columns (a player's
// bindings), instead of the menu's valueStyle.
type slots struct {
	sc                  scope
	style, waitingStyle screenpack.StyleID
	column              int  // the highlighted column, -1 for none
	waiting             bool // the highlighted slot of the cursor row awaits a key
}

// noSlots is for a menu whose values all take the menu's valueStyle.
var noSlots = slots{column: -1}

// twoSlotColumns are where the right edges of a row's key and button columns
// lie, as fractions of valueOffset's x.
// shortcut: the format has no offset for a second value; these keep the two
// columns apart at 4:3 and 16:9. Upgrade when the format gets one.
var twoSlotColumns = [2]float32{0.65, 1.5}

// menu draws a vertical menu: its first entry is placed by the menu's layout
// (the pivot of a row as wide as the widest label, on the anchor), and each
// next one is spacing units below. Labels are aligned inside that width by
// their style's align, a value is aligned against the point valueOffset away
// from the reference point of the labels (their normal style's align, so
// columns do not move with the cursor), and the cursor element is placed
// inside the active row, behind its text. With visibleEntries, the window of
// rows moves to keep the cursor in view, centered on it when it can be.
// shortcut: only the vertical direction; upgrade with the first horizontal menu (options tabs).
func (l *Layout) menu(dst *DrawList, view View, sc scope, m *screenpack.Menu, rows []Row, cursor int, parent rect, elapsedMs int, sl slots) {
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
	for _, r := range rows {
		widest = max(widest, normal.width(r.Label))
	}
	visible := len(rows)
	if m.VisibleEntries > 0 {
		visible = min(visible, m.VisibleEntries)
	}
	first := max(0, min(cursor-visible/2, len(rows)-visible))
	unit := view.unit()
	pitch := float32(m.Spacing) * unit
	top := place(parent, &m.Layout, widest, pitch, unit)
	for i := first; i < first+visible; i++ {
		line := rect{top.x, top.y + float32(i-first)*pitch, top.w, pitch}
		style := normal
		if i == cursor {
			style = active
			if m.Cursor != nil {
				l.element(dst, view, sc, m.Cursor, line, elapsedMs)
			}
		}
		if rows[i].Flagged {
			style.style.Color = warningColor
		}
		af := alignFactor(style.style.Align)
		style.emit(dst, rows[i].Label, line.x+(line.w-style.width(rows[i].Label))*af, line.y+pitch/2, 1)
		l.values(dst, view, sc, m, rows[i], i == cursor, line.x+line.w*alignFactor(normal.style.Align), line.y+pitch/2, sl)
	}
}

// values draws a row's values, aligned against the points valueOffset to the
// right of ref (the label's reference x): the first column's, and the
// second's when the row has two.
func (l *Layout) values(dst *DrawList, view View, sc scope, m *screenpack.Menu, row Row, onCursor bool, ref, centerY float32, sl slots) {
	if m.ValueOffset == nil {
		return
	}
	unit := view.unit()
	columns := [2]float32{1, 0}
	if row.Values[1] != "" {
		columns = twoSlotColumns
	}
	for v, text := range row.Values {
		if text == "" {
			continue
		}
		id, from := m.ValueStyle, sc
		if sl.style != "" {
			id, from = sl.style, sl.sc
		}
		if v == sl.column {
			id, from = m.ActiveStyle, sc
			if onCursor && sl.waiting {
				id, from, text = sl.waitingStyle, sl.sc, "..."
			}
		}
		style, ok := l.styled(from, id, view)
		if !ok {
			continue
		}
		if row.Flagged {
			style.style.Color = warningColor
		}
		edge := ref + float32(m.ValueOffset[0])*unit*columns[v]
		style.emit(dst, text, edge-style.width(text)*alignFactor(style.style.Align), centerY+float32(m.ValueOffset[1])*unit, 1)
	}
}
