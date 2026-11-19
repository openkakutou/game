package gfx

import (
	"fmt"
	"os"

	"github.com/openkakutou/game/native/internal/game"
	"github.com/openkakutou/game/native/internal/lifebar"
	"github.com/openkakutou/sff"
)

// lifebarSheet is the spriteKey.fighter value of the lifebar's sprite sheet.
const lifebarSheet = -2

// lifebarHUD draws a MUGEN / Ikemen GO lifebar over the match.
type lifebarHUD struct {
	hud    *game.HUDLifebar
	sheet  map[spriteKey]*Texture
	axis   map[spriteKey][2]int
	meters [2]lifebar.Meter
	health [2]int
	tick   int
}

func newLifebarHUD(hud *game.HUDLifebar) *lifebarHUD {
	l := &lifebarHUD{hud: hud, sheet: map[spriteKey]*Texture{}, axis: map[spriteKey][2]int{}, tick: -1}
	l.load()
	return l
}

// load uploads every sprite the lifebar's elements can show, once.
func (l *lifebarHUD) load() {
	if l.hud.Sprites == "" {
		return
	}
	file, err := os.Open(l.hud.Sprites)
	if err != nil {
		return
	}
	defer file.Close()
	groups, err := sff.Load(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "openkakutou: lifebar sprites:", err)
		return
	}
	for _, g := range groups {
		for _, sp := range g.Sprites {
			l.axis[spriteKey{lifebarSheet, sp.Group, sp.Image}] = [2]int{sp.AxisX, sp.AxisY}
		}
	}
	upload := func(group, image int) {
		k := spriteKey{lifebarSheet, group, image}
		if group < 0 || image < 0 || l.sheet[k] != nil {
			return
		}
		px, wd, ht, err := sff.ResolveSpritePixels(file, group, image, nil)
		if err != nil || wd == 0 || ht == 0 {
			return
		}
		l.sheet[k] = NewTexture(px, wd, ht)
	}
	for _, e := range l.elements() {
		if e.Anim < 0 {
			upload(e.Sprite[0], e.Sprite[1])
			continue
		}
		for _, fr := range l.hud.Anims[e.Anim].Frames {
			upload(fr.Group, fr.Image)
		}
	}
}

func (l *lifebarHUD) elements() []lifebar.Element {
	var out []lifebar.Element
	for _, bars := range [][2]lifebar.Bar{l.hud.Life, l.hud.Power} {
		for _, b := range bars {
			out = append(out, b.Bg[0], b.Bg[1], b.Bg[2], b.Mid, b.Front, b.Top)
		}
	}
	return out
}

func (l *lifebarHUD) delete() {
	for _, t := range l.sheet {
		t.Delete()
	}
}

// step advances the life meters once per simulated tick; a tick going
// back means a new round, which starts full.
func (l *lifebarHUD) step(m *game.Match) {
	tick := m.Ticks()
	if tick == l.tick {
		return
	}
	if tick < l.tick || l.tick < 0 {
		for i := range l.meters {
			l.meters[i] = lifebar.NewMeter()
			l.health[i] = m.State.Fighters[i].Health
		}
	}
	for i := range l.meters {
		h := m.State.Fighters[i].Health
		// The engine exposes no move type: a drop in life stands in for
		// Ikemen GO's getHit, which freezes the trailing fill.
		hit := h < l.health[i]
		l.meters[i].Step(lifeFraction(h), hit, l.hud.Life[i].Mid.Shown())
		l.health[i] = h
	}
	l.tick = tick
}

func lifeFraction(h int) float64 {
	return float64(min(max(h, 0), game.DefaultHealth)) / game.DefaultHealth
}

// draw paints the lifebar in Ikemen GO's order: layer by layer, life then
// power bars, each one's backgrounds, trailing fill, front fill, top
// (FightScreen.draw, fightscreen.go:5757).
func (l *lifebarHUD) draw(w *Window, f *game.Flow) {
	m := f.Match
	l.step(m)
	scale := l.hud.Scale(VW)
	ticks := m.Ticks()
	for layer := range 3 {
		for i := range 2 {
			l.drawBar(w, l.hud.Life[i], layer, ticks, scale, l.meters[i].Top, l.meters[i].Mid)
		}
		for i := range 2 {
			power := min(float64(m.State.Fighters[i].Power)/game.MaxPower, 1)
			l.drawBar(w, l.hud.Power[i], layer, ticks, scale, power, power)
		}
	}
	l.drawTexts(w, f, scale)
}

func (l *lifebarHUD) drawBar(w *Window, b lifebar.Bar, layer, ticks int, scale, top, mid float64) {
	for _, e := range b.Bg {
		l.drawElement(w, e, b.Pos, layer, ticks, scale)
	}
	front, trail := b.Clips(scale, VW, VH, top, mid)
	w.Clip(float32(trail.X), float32(trail.Y), float32(trail.W), float32(trail.H))
	l.drawElement(w, b.Mid, b.Pos, layer, ticks, scale)
	w.Clip(float32(front.X), float32(front.Y), float32(front.W), float32(front.H))
	l.drawElement(w, b.Front, b.Pos, layer, ticks, scale)
	w.NoClip()
	l.drawElement(w, b.Top, b.Pos, layer, ticks, scale)
}

func (l *lifebarHUD) drawElement(w *Window, e lifebar.Element, pos [2]int, layer, ticks int, scale float64) {
	if e.Layer != layer {
		return
	}
	fr, ok := l.hud.Frame(e, ticks)
	if !ok {
		return
	}
	k := spriteKey{lifebarSheet, fr.Group, fr.Image}
	t := l.sheet[k]
	if t == nil {
		return
	}
	r, flip := lifebar.Place(e, fr, l.axis[k], [2]int{t.W, t.H}, pos, scale)
	w.SpriteSized(t, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), flip)
}

// drawTexts prints the names and the timer where the lifebar places them,
// in the built-in font: the lifebar's own fonts are not read yet.
func (l *lifebarHUD) drawTexts(w *Window, f *game.Flow, scale float64) {
	for i, name := range l.hud.Name {
		if name.Shown() {
			printAt(w, name, scale, f.Fighters[i].Char.Name)
		}
	}
	if l.hud.Time.Shown() {
		printAt(w, l.hud.Time, scale, timerText(f))
	}
}

func printAt(w *Window, t lifebar.Text, scale float64, s string) {
	x := (float64(t.Pos[0]) + t.Offset[0]) * scale
	y := (float64(t.Pos[1])+t.Offset[1])*scale - glyphHeight
	width := float64(TextWidth(s))
	switch {
	case t.Align < 0:
		x -= width
	case t.Align == 0:
		x -= width / 2
	}
	w.Text(float32(x), float32(y), 1, white, s)
}

// glyphHeight is the built-in font's height: lifebar text positions are
// baselines.
const glyphHeight = 7

func timerText(f *game.Flow) string {
	if f.Setup().TimeLimitSeconds <= 0 {
		return "--"
	}
	return fmt.Sprintf("%02d", (f.Match.State.RoundTimer+game.TickRateHz-1)/game.TickRateHz)
}
