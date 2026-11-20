package ui

import (
	"github.com/openkakutou/screenpack"
)

// elements draws a list of elements in order, later on top.
func (l *Layout) elements(dst *DrawList, view View, sc scope, list []screenpack.Element, parent rect, elapsedMs int) {
	for i := range list {
		l.element(dst, view, sc, &list[i], parent, elapsedMs)
	}
}

// element draws one image, sprite, rect or text element placed inside parent.
// shortcut: a rect's radius is drawn square, and a tiled image's scroll and a
// text element's transition are ignored; upgrade when a screenpack needs them.
func (l *Layout) element(dst *DrawList, view View, sc scope, e *screenpack.Element, parent rect, elapsedMs int) {
	if e.Visible != nil && !*e.Visible {
		return
	}
	alpha := float32(1)
	if e.Opacity != nil {
		alpha = float32(*e.Opacity)
	}
	switch e.Type {
	case "rect":
		l.rectElement(dst, view, e, parent, alpha, elapsedMs)
	case "image":
		l.imageElement(dst, view, sc, e, parent, alpha, elapsedMs)
	case "text":
		l.text(dst, view, sc, &e.Layout, e.Style, e.Text, parent, alpha, "")
	}
}

// rectElement fills the element's box, the whole parent when it has no size.
func (l *Layout) rectElement(dst *DrawList, view View, e *screenpack.Element, parent rect, alpha float32, elapsedMs int) {
	w, h := parent.w, parent.h
	if e.Size != nil && e.Size[0] != nil {
		w = float32(*e.Size[0]) * view.unit()
	}
	if e.Size != nil && e.Size[1] != nil {
		h = float32(*e.Size[1]) * view.unit()
	}
	box, alpha := enter(e.Transition, elapsedMs, view, place(parent, &e.Layout, w, h, view.unit()), alpha)
	dst.Ops = append(dst.Ops, Op{Kind: OpRect, X: box.x, Y: box.y, W: box.w, H: box.h, Color: parseColor(e.Color, alpha)})
}

// picture is a region of an image file and its natural size in pixels.
type picture struct {
	asset          Asset
	u0, v0, u1, v1 float32
	w, h           float32
}

// pictureOf finds what an image element shows: a still image, or a frame of
// its sprite sheet (the animation's frame at elapsedMs). ok is false when the
// file cannot be read.
func (l *Layout) pictureOf(sc scope, e *screenpack.Element, elapsedMs int) (pic picture, ok bool) {
	if e.Sheet == "" {
		pic.asset = Asset{sc.src, string(e.Image)}
		w, h, ok := l.pack.imageSize(pic.asset)
		pic.u1, pic.v1, pic.w, pic.h = 1, 1, float32(w), float32(h)
		return pic, ok
	}
	file, sheet := l.pack.sheetImage(sc.src, e.Sheet)
	if sheet == nil {
		return pic, false
	}
	frame := e.Frame
	if e.Animation != "" {
		frame = animationFrame(sheet.Animations[e.Animation], elapsedMs)
	}
	region := sheet.Frames[frame] // x, y, w, h in image pixels
	pic.asset = Asset{sc.src, file}
	imgW, imgH, ok := l.pack.imageSize(pic.asset)
	if !ok || region[2] <= 0 || region[3] <= 0 {
		return pic, false
	}
	pic.u0, pic.v0 = float32(region[0])/float32(imgW), float32(region[1])/float32(imgH)
	pic.u1, pic.v1 = float32(region[0]+region[2])/float32(imgW), float32(region[1]+region[3])/float32(imgH)
	pic.w, pic.h = float32(region[2]), float32(region[3])
	return pic, true
}

// animationFrame is the frame id an animation shows after elapsedMs.
func animationFrame(a screenpack.Animation, elapsedMs int) string {
	total := 0
	for _, f := range a.Frames {
		total += f.Ms
	}
	if total == 0 {
		return ""
	}
	t := elapsedMs
	if a.Loop == nil || *a.Loop {
		t %= total
	} else {
		t = min(t, total-1)
	}
	for _, f := range a.Frames {
		if t < f.Ms {
			return f.Frame
		}
		t -= f.Ms
	}
	return ""
}

// imageElement draws an image or a sprite frame. A tiled still image is one
// quad whose texture coordinates run past 1: the cell is the image at its
// natural size on a tiled axis, and the quad's own size on the other.
func (l *Layout) imageElement(dst *DrawList, view View, sc scope, e *screenpack.Element, parent rect, alpha float32, elapsedMs int) {
	pic, ok := l.pictureOf(sc, e, elapsedMs)
	if !ok {
		return
	}
	w, h := boxSize(e.Size, pic.w, pic.h, view)
	if e.Sheet == "" {
		scale := float32(view.H) / referenceLines
		if e.Tile == "x" || e.Tile == "both" {
			pic.u1 = w / (pic.w * scale)
		}
		if e.Tile == "y" || e.Tile == "both" {
			pic.v1 = h / (pic.h * scale)
		}
	}
	box, alpha := enter(e.Transition, elapsedMs, view, place(parent, &e.Layout, w, h, view.unit()), alpha)
	dst.Ops = append(dst.Ops, Op{
		Kind: OpImage, X: box.x, Y: box.y, W: box.w, H: box.h,
		Color: Color{1, 1, 1, alpha}, Asset: pic.asset, U0: pic.u0, V0: pic.v0, U1: pic.u1, V1: pic.v1,
	})
}

// enter applies the "in" effect of a transition, elapsedMs after the screen
// appeared, to an element's box and opacity.
func enter(t *screenpack.Transition, elapsedMs int, view View, box rect, alpha float32) (rect, float32) {
	if t == nil || t.In == nil {
		return box, alpha
	}
	p := progress(t.In, elapsedMs)
	switch t.In.Kind {
	case "fade":
		alpha *= p
	case "scale":
		from, _ := t.In.From.(float64)
		s := float32(from) + (1-float32(from))*p
		box = rect{box.x + box.w*(1-s)/2, box.y + box.h*(1-s)/2, box.w * s, box.h * s}
	case "slide":
		side, _ := t.In.From.(string)
		away := 1 - p
		switch side {
		case "left":
			box.x -= (box.x + box.w) * away
		case "right":
			box.x += (float32(view.W) - box.x) * away
		case "top":
			box.y -= (box.y + box.h) * away
		case "bottom":
			box.y += (float32(view.H) - box.y) * away
		}
	}
	return box, alpha
}

// progress is how far an effect has come, 0 to 1, after its delay and easing.
func progress(e *screenpack.Effect, elapsedMs int) float32 {
	if e.Ms <= 0 {
		return 1
	}
	t := float32(elapsedMs-e.DelayMs) / float32(e.Ms)
	t = max(0, min(1, t))
	switch e.Easing {
	case "linear":
		return t
	case "easeIn":
		return t * t
	case "easeInOut":
		if t < 0.5 {
			return 2 * t * t
		}
		return 1 - 2*(1-t)*(1-t)
	}
	return 1 - (1-t)*(1-t) // easeOut, the default
}
