package gfx

import (
	"fmt"
	"github.com/openkakutou/game/native/internal/bmfont"
	"io/fs"
	"os"
	"strconv"
	"strings"

	"github.com/openkakutou/character/air"
	"github.com/openkakutou/game/native/internal/game"
	"github.com/openkakutou/lifebar"
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
	// glyphs are the font glyph textures uploaded so far; placed is the
	// layout buffer reused by every text.
	glyphs map[*bmfont.Glyph]*Texture
	placed []bmfont.Placed
}

func newLifebarHUD(hud *game.HUDLifebar, fsys fs.FS) *lifebarHUD {
	l := &lifebarHUD{hud: hud, sheet: map[spriteKey]*Texture{}, axis: map[spriteKey][2]int{}, tick: -1}
	l.load(fsys)
	return l
}

// load uploads every sprite the lifebar's elements can show, once.
func (l *lifebarHUD) load(fsys fs.FS) {
	if l.hud.Sprites == "" {
		return
	}
	file, err := openSprites(fsys, l.hud.Sprites)
	if err != nil {
		return
	}
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
		if e.HasAnim {
			for _, fr := range l.hud.Animations[e.Anim].Frames {
				upload(fr.Group, fr.Image)
			}
			continue
		}
		if e.HasSprite {
			upload(e.Sprite.Group, e.Sprite.Image)
		}
	}
}

// elements are the lifebar's elements the HUD draws.
func (l *lifebarHUD) elements() []lifebar.Element {
	var out []lifebar.Element
	for i := range 2 {
		for _, b := range []lifebar.Bar{l.hud.Life[i].Bar, l.hud.Power[i].Bar} {
			out = append(out, b.Bg0, b.Bg1, b.Bg2, b.Mid, b.Front, b.Top)
		}
		out = append(out, l.hud.Name[i].Bg, l.hud.Name[i].Top)
	}
	out = append(out, l.hud.Time.Bg, l.hud.Time.Top)
	r := &l.hud.Round
	for _, a := range r.Round {
		out = append(out, a.Element)
	}
	return append(out, r.RoundDefault.Element, r.Fight.Element, r.Ko.Element, r.Dko.Element, r.To.Element)
}

// screen is where the lifebar lands in the 320x240 view (Ikemen GO's
// fightscreen scale and offset for that game size).
func (l *lifebarHUD) screen() lifebar.Screen {
	return l.hud.Screen([2]int{VW, VH}, [2]int{VW, VH})
}

func (l *lifebarHUD) delete() {
	for _, t := range l.sheet {
		t.Delete()
	}
	for _, t := range l.glyphs {
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
		mid := l.hud.Life[i].Mid
		l.meters[i].Step(l.hud.Life[i], lifebar.LifeTick{Life: float32(lifeFraction(h, m.LifeMax[i])), GetHit: hit, HasMid: mid.HasAnim || mid.HasSprite})
		l.health[i] = h
	}
	l.tick = tick
}

// lifeFraction is how full a bar of life h is, for a fighter of life lifeMax.
func lifeFraction(h, lifeMax int) float64 {
	if lifeMax <= 0 {
		return 0
	}
	return float64(min(max(h, 0), lifeMax)) / float64(lifeMax)
}

// draw paints the match's lifebar.
func (l *lifebarHUD) draw(w *Window, f *game.Flow) {
	m := f.Match
	l.step(m)
	var power [2]float64
	var names [2]string
	for i := range power {
		power[i] = min(float64(m.State.Fighters[i].Power)/game.MaxPower, 1)
		names[i] = f.Fighters[i].Char.Name
	}
	l.paint(w, m.Ticks(), l.meters, power, names, timerText(f))
	l.drawCalls(w, f)
}

// drawCalls draws the round, fight and KO calls showing, at the [Round]
// pos (fs.go:4030-4200).
func (l *lifebarHUD) drawCalls(w *Window, f *game.Flow) {
	lb := l.hud.Lifebar
	s := l.screen()
	pos := lb.Round.Pos
	for _, shown := range f.Announcer.Showing(lb) {
		if shown.Call == game.CallNone {
			continue
		}
		a := game.CallOf(lb, shown.Call, f.Match.State.Round, f.Announcer.Finish())
		for layer := range 3 {
			l.drawElement(w, s, a.Element, pos, layer, shown.Elapsed)
		}
		if a.Text.Text != "" {
			l.print(w, s, a.Text, pos, callText(a.Text.Text, f.Match.State.Round))
		}
	}
}

// callText is a call's text, %i standing for the round number.
func callText(text string, roundNo int) string {
	return strings.ReplaceAll(text, "%i", strconv.Itoa(roundNo))
}

// sampleNames and sampleTimer are the texts of the sample lifebar the
// appearance screen shows.
var sampleNames = [2]string{"Player 1", "Player 2"}

const sampleTimer = "99"

// drawSample paints the lifebar as the appearance screen previews it: both
// life bars full, the power bars empty, ticks into the animations.
func (l *lifebarHUD) drawSample(w *Window, ticks int) {
	full := [2]lifebar.Meter{lifebar.NewMeter(), lifebar.NewMeter()}
	full[0].Top, full[1].Top = 1, 1
	l.paint(w, ticks, full, [2]float64{}, sampleNames, sampleTimer)
}

// paint draws the lifebar in Ikemen GO's order: layer by layer, life then
// power bars, each one's backgrounds, trailing fill, front fill, top
// (FightScreen.draw, fightscreen.go:5757).
func (l *lifebarHUD) paint(w *Window, ticks int, life [2]lifebar.Meter, power [2]float64, names [2]string, timer string) {
	s := l.screen()
	for layer := range 3 {
		for i := range 2 {
			l.drawBar(w, s, l.hud.Life[i].Bar, layer, ticks, life[i].Top, life[i].Mid)
		}
		for i := range 2 {
			l.drawBar(w, s, l.hud.Power[i].Bar, layer, ticks, float32(power[i]), float32(power[i]))
		}
		for i := range 2 {
			n := l.hud.Name[i]
			l.drawElement(w, s, n.Bg, n.Pos, layer, ticks)
		}
		l.drawElement(w, s, l.hud.Time.Bg, l.hud.Time.Pos, layer, ticks)
	}
	l.drawTexts(w, s, names, timer)
	for layer := range 3 {
		for i := range 2 {
			n := l.hud.Name[i]
			l.drawElement(w, s, n.Top, n.Pos, layer, ticks)
		}
		l.drawElement(w, s, l.hud.Time.Top, l.hud.Time.Pos, layer, ticks)
	}
}

func (l *lifebarHUD) drawBar(w *Window, s lifebar.Screen, b lifebar.Bar, layer, ticks int, top, mid float32) {
	for _, e := range []lifebar.Element{b.Bg0, b.Bg1, b.Bg2} {
		l.drawElement(w, s, e, b.Pos, layer, ticks)
	}
	front, trail := b.Clips(s, top, mid)
	w.Clip(float32(trail.X), float32(trail.Y), float32(trail.W), float32(trail.H))
	l.drawElement(w, s, b.Mid, b.Pos, layer, ticks)
	w.Clip(float32(front.X), float32(front.Y), float32(front.W), float32(front.H))
	l.drawElement(w, s, b.Front, b.Pos, layer, ticks)
	w.NoClip()
	l.drawElement(w, s, b.Top, b.Pos, layer, ticks)
}

func (l *lifebarHUD) drawElement(w *Window, s lifebar.Screen, e lifebar.Element, pos [2]int, layer, ticks int) {
	if e.Layout.LayerNo != layer {
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
	p := s.Place(e, fr, lifebar.SpriteBox{Axis: l.axis[k], Size: [2]int{t.W, t.H}}, pos)
	w.SpriteSized(t, float32(p.X), float32(p.Y), float32(p.W), float32(p.H), p.FlipX)
}

// drawTexts prints the names and the timer where the lifebar places them,
// in the lifebar's own font, else the built-in one.
func (l *lifebarHUD) drawTexts(w *Window, s lifebar.Screen, names [2]string, timer string) {
	for i, name := range l.hud.Name {
		if name.Name.Font.No >= 0 {
			l.print(w, s, name.Name, name.Pos, names[i])
		}
	}
	if l.hud.Time.Counter.Font.No >= 0 {
		l.print(w, s, l.hud.Time.Counter, l.hud.Time.Pos, timer)
	}
}

// textAnchor is where the lifebar places a text at pos: the origin of a
// zero-size sprite there.
func textAnchor(s lifebar.Screen, t lifebar.Text, pos [2]int) (x, y float64) {
	p := s.Place(lifebar.Element{Layout: t.Layout}, air.Frame{}, lifebar.SpriteBox{}, pos)
	return p.X, p.Y
}

func (l *lifebarHUD) print(w *Window, s lifebar.Screen, t lifebar.Text, pos [2]int, text string) {
	x, y := textAnchor(s, t, pos)
	scale := s.Scale * float64(t.Layout.Scale[0])
	f := l.hud.Fonts[t.Font.No]
	if f == nil {
		printAt(w, t, x, y, text)
		return
	}
	l.placed = f.Layout(text, t.Font.Bank, t.Font.Align, x, y, scale, l.placed)
	for _, p := range l.placed {
		tex := l.glyphs[p.Glyph]
		if tex == nil {
			if l.glyphs == nil {
				l.glyphs = map[*bmfont.Glyph]*Texture{}
			}
			tex = NewTexture(p.Glyph.Pixels, p.Glyph.W, p.Glyph.H)
			l.glyphs[p.Glyph] = tex
		}
		w.SpriteSized(tex, float32(p.X), float32(p.Y), float32(float64(p.Glyph.W)*scale), float32(float64(p.Glyph.H)*scale), false)
	}
}

func printAt(w *Window, t lifebar.Text, x, y float64, s string) {
	y -= glyphHeight
	width := float64(TextWidth(s))
	switch {
	case t.Font.Align < 0:
		x -= width
	case t.Font.Align == 0:
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
