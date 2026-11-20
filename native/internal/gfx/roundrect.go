package gfx

import "math"

// cornerSegments is the number of triangles per rounded corner.
const cornerSegments = 8

// roundedTriangles appends to dst[:0] the triangles (x, y pairs, 3 vertices
// each) filling the rectangle x, y, w, h with corners of radius r: a middle
// column, two side columns, and a fan per corner.
func roundedTriangles(x, y, w, h, r float32, dst []float32) []float32 {
	dst = dst[:0]
	quad := func(x0, y0, x1, y1 float32) {
		dst = append(dst, x0, y0, x1, y0, x1, y1, x0, y0, x1, y1, x0, y1)
	}
	quad(x+r, y, x+w-r, y+h)
	quad(x, y+r, x+r, y+h-r)
	quad(x+w-r, y+r, x+w, y+h-r)
	corners := [4][3]float32{
		{x + r, y + r, math.Pi},           // top left: from left to up
		{x + w - r, y + r, 1.5 * math.Pi}, // top right
		{x + w - r, y + h - r, 0},         // bottom right
		{x + r, y + h - r, 0.5 * math.Pi}, // bottom left
	}
	for _, c := range corners {
		for i := range cornerSegments {
			a0 := float64(c[2]) + float64(i)*math.Pi/2/cornerSegments
			a1 := a0 + math.Pi/2/cornerSegments
			dst = append(dst, c[0], c[1],
				c[0]+r*float32(math.Cos(a0)), c[1]+r*float32(math.Sin(a0)),
				c[0]+r*float32(math.Cos(a1)), c[1]+r*float32(math.Sin(a1)))
		}
	}
	return dst
}

// RoundRect queues a filled rectangle with rounded corners.
func (w *Window) RoundRect(x, y, wd, ht, r float32, c Color) {
	if r <= 0 {
		w.Rect(x, y, wd, ht, c)
		return
	}
	w.round = roundedTriangles(x, y, wd, ht, r, w.round)
	for i := 0; i < len(w.round); i += 2 {
		w.verts = append(w.verts, w.round[i], w.round[i+1], c.R, c.G, c.B, c.A)
	}
	w.nverts += len(w.round) / 2
}
