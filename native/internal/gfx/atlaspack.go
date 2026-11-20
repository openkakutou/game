package gfx

// shelfPacker places rectangles in a square page row by row: each goes right
// of the previous one on the current row, or starts a new row below the
// tallest of the last. Rectangles are one pixel apart.
type shelfPacker struct {
	size       int
	x, y, rowH int
}

// place returns the top-left corner for a w x h rectangle; ok is false when
// the page has no room left for it, or it is larger than the page.
func (p *shelfPacker) place(w, h int) (x, y int, ok bool) {
	if w > p.size || h > p.size {
		return 0, 0, false
	}
	if p.x+w > p.size {
		p.x, p.y, p.rowH = 0, p.y+p.rowH+1, 0
	}
	if p.y+h > p.size {
		return 0, 0, false
	}
	x, y = p.x, p.y
	p.x += w + 1
	p.rowH = max(p.rowH, h)
	return x, y, true
}
