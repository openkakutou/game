package ui

import (
	"github.com/openkakutou/game/native/internal/game"
	"github.com/openkakutou/screenpack"
)

// listMargin is the space, in units, between the list and each side of the
// screen when its layout gives no width.
// shortcut: the format gives the list no width besides its layout's size;
// upgrade when it does.
const listMargin = 8

// MoveListState is what the game shows on the move list screen, over the
// paused match.
type MoveListState struct {
	Title string
	Lines []game.MoveLine
	// Top is the index of the first line shown.
	Top       int
	ElapsedMs int
}

// MoveList adds the move list screen to dst: background, the title, the
// visible lines with their button icons, the scrollbar when the list is
// longer than the window, foreground. It allocates nothing once warm.
func (l *Layout) MoveList(dst *DrawList, view View, s MoveListState) {
	m := l.pack.resolved.Screens.MoveList
	screen, from := view.rect(), len(dst.Ops)
	l.elements(dst, view, l.pack.scope("moveList", ""), m.Background, screen, s.ElapsedMs)
	if m.Title != nil {
		l.text(dst, view, l.pack.scope("moveList", "title"), &m.Title.Layout, m.Title.Style, s.Title, screen, 1, "")
	}
	if m.List != nil {
		l.moveLines(dst, view, m, s, screen)
	}
	if m.Scrollbar != nil && m.List != nil && len(s.Lines) > m.List.VisibleLines && m.List.VisibleLines > 0 {
		l.scrollbar(dst, view, m.Scrollbar, s.Top, len(s.Lines)-m.List.VisibleLines, screen, s.ElapsedMs)
	}
	l.elements(dst, view, l.pack.scope("moveList", ""), m.Foreground, screen, s.ElapsedMs)
	fadeIn(dst, from, &m.Screen, s.ElapsedMs)
}

// moveBrush draws the runs of a line: text in the list's style, icons as tall
// as the text.
type moveBrush struct {
	l      *Layout
	style  styled
	icons  scope
	frames map[string]string
	sheet  screenpack.Path
	height float32
}

// measure is a run's width, and the icon to draw when it has one the pack
// shows.
func (b *moveBrush) measure(r game.MoveRun) (width float32, pic picture, icon bool) {
	frame := b.frames[r.Icon]
	if r.Icon != "" && frame != "" {
		e := screenpack.Element{Sheet: b.sheet, Frame: frame}
		if pic, ok := b.l.pictureOf(b.icons, &e, 0); ok {
			return b.height * pic.w / pic.h, pic, true
		}
	}
	return b.style.width(r.Text), picture{}, false
}

func (b *moveBrush) cellWidth(c game.MoveCell) float32 {
	var w float32
	for _, r := range c.Runs {
		rw, _, _ := b.measure(r)
		w += rw
	}
	return w
}

// draw adds the cell's runs from x, on the line centered at centerY.
func (b *moveBrush) draw(dst *DrawList, c game.MoveCell, x, centerY float32) {
	for _, r := range c.Runs {
		w, pic, icon := b.measure(r)
		if icon {
			dst.Ops = append(dst.Ops, Op{
				Kind: OpImage, X: x, Y: centerY - b.height/2, W: w, H: b.height,
				Color: Color{1, 1, 1, 1}, Asset: pic.asset, U0: pic.u0, V0: pic.v0, U1: pic.u1, V1: pic.v1,
			})
		} else {
			b.style.emit(dst, r.Text, x, centerY, 1)
		}
		x += w
	}
}

// moveLines draws the window of lines. A line's cells sit as Ikemen GO's
// move list puts them (menu.lua:886-952): left cells one after the other from
// the left edge, the first center cell centered and the next ones after it,
// right cells from the right edge, each to the left of the one before.
func (l *Layout) moveLines(dst *DrawList, view View, m *screenpack.MoveList, s MoveListState, parent rect) {
	sc := l.pack.scope("moveList", "list")
	style, ok := l.styled(sc, m.List.Style, view)
	if !ok {
		return
	}
	unit := view.unit()
	width := parent.w - 2*listMargin*unit
	if m.List.Size != nil && m.List.Size[0] != nil {
		width = float32(*m.List.Size[0]) * unit
	}
	pitch := float32(m.List.LineSpacing) * unit
	box := place(parent, &m.List.Layout, width, float32(m.List.VisibleLines)*pitch, unit)
	b := moveBrush{l: l, style: style, height: float32(style.style.Size) * unit}
	if m.Buttons != nil {
		b.icons, b.sheet, b.frames = l.pack.scope("moveList", "buttons"), m.Buttons.Sheet, m.Buttons.Frames
	}
	top := max(0, min(s.Top, len(s.Lines)))
	end := min(len(s.Lines), top+m.List.VisibleLines)
	for i := top; i < end; i++ {
		centerY := box.y + float32(i-top)*pitch + pitch/2
		var used [3]float32 // the width taken by the left, center and right cells
		centered := false
		for _, c := range s.Lines[i] {
			w := b.cellWidth(c)
			var x float32
			switch c.Align {
			case game.MoveLeft:
				x = box.x + used[0]
				used[0] += w
			case game.MoveCenter:
				if centered {
					x = box.x + used[1]
					used[1] += w
				} else {
					x = box.x + (width-w)/2
					used[1], centered = (width+w)/2, true
				}
			case game.MoveRight:
				used[2] += w
				x = box.x + width - used[2]
			}
			b.draw(dst, c, x, centerY)
		}
	}
}

// scrollbar draws the track, and the thumb along it where the first line shown
// stands among the lines that can be first (0 to last).
func (l *Layout) scrollbar(dst *DrawList, view View, bar *screenpack.Scrollbar, top, last int, parent rect, elapsedMs int) {
	sc := l.pack.scope("moveList", "scrollbar")
	trackAt := len(dst.Ops)
	if bar.Track != nil {
		l.element(dst, view, sc, bar.Track, parent, elapsedMs)
	}
	if bar.Thumb == nil || trackAt == len(dst.Ops) {
		return
	}
	track := dst.Ops[trackAt]
	thumbAt := len(dst.Ops)
	l.element(dst, view, sc, bar.Thumb, parent, elapsedMs)
	if thumbAt == len(dst.Ops) {
		return
	}
	thumb := dst.Ops[thumbAt]
	share := float32(max(0, min(top, last))) / float32(last)
	dy := track.Y + (track.H-thumb.H)*share - thumb.Y
	for i := thumbAt; i < len(dst.Ops); i++ {
		dst.Ops[i].Y += dy
	}
}
