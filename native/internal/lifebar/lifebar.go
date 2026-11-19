// Package lifebar reads a MUGEN / Ikemen GO lifebar (fight.def) and works
// out what its bars show, following Ikemen GO's fightscreen.go (commit
// 54d4148a). It draws nothing: the gfx package renders what it describes.
package lifebar

import (
	"fmt"
	"strings"

	"github.com/openkakutou/character/air"
)

// DefaultLocalcoord is the coordinate space of a fight.def that declares
// none in [Info] (Ikemen GO fightscreen.go:4822, without a screenpack).
var DefaultLocalcoord = [2]int{320, 240}

// Element is an Ikemen GO AnimLayout (common.go:1507): a sprite or an
// animation from the lifebar's own .sff, placed at Offset from its owner.
type Element struct {
	// Sprite is the (group, image) shown when Anim is not set.
	Sprite [2]int
	// Anim is the [Begin Action] number shown; -1 when none.
	Anim    int
	Offset  [2]float64
	Facing  int
	VFacing int
	// Layer is layerno, at most 2 (common.go:1366).
	Layer int
	Scale [2]float64
	set   bool
}

// Shown reports whether the element names a sprite or an animation.
func (e Element) Shown() bool { return e.set }

// Bar is one player's [Lifebar] or [Powerbar] entry (fightscreen.go:440).
type Bar struct {
	Pos    [2]int
	RangeX [2]int
	RangeY [2]int
	// Bg are bg0, bg1 and bg2, drawn first and never clipped.
	Bg [3]Element
	// Mid is the trailing fill, Front the current one, both clipped to the
	// range; Top is drawn last, unclipped.
	Mid, Front, Top Element
}

// Text is a position where the lifebar prints a value with one of its
// fonts: Pos plus the text's own offset.
type Text struct {
	Pos    [2]int
	Offset [2]float64
	Font   int
	// Align is the font's third value: 1 left, 0 centered, -1 right.
	Align int
	set   bool
}

// Shown reports whether the lifebar places this text.
func (t Text) Shown() bool { return t.set }

// Lifebar is a parsed fight.def, single-player layout (Ikemen GO's
// lifeBars[0], "[Lifebar]" with p1/p2 prefixes).
type Lifebar struct {
	Localcoord [2]int
	// Sprites is the [Files] sff entry as written, relative to the def.
	Sprites string
	Life    [2]Bar
	Power   [2]Bar
	Name    [2]Text
	Time    Text
	// Anims are the def's [Begin Action] blocks by number.
	Anims map[int]air.Animation
}

// Parse reads a fight.def. It fails when the text has no [Lifebar] section,
// the one thing a lifebar cannot do without.
func Parse(src string) (*Lifebar, error) {
	secs := sections(src)
	life, ok := secs["lifebar"]
	if !ok {
		return nil, fmt.Errorf("no [Lifebar] section")
	}
	lb := &Lifebar{Localcoord: DefaultLocalcoord, Anims: map[int]air.Animation{}}
	if anims, err := air.Parse(strings.NewReader(src)); err == nil {
		for _, a := range anims {
			lb.Anims[a.Number] = a
		}
	}
	if info, has := secs["info"]; has {
		info.ints("localcoord", &lb.Localcoord[0], &lb.Localcoord[1])
	}
	if files, has := secs["files"]; has {
		lb.Sprites = files["sff"]
	}
	power := secs["powerbar"]
	name := secs["name"]
	for i := range 2 {
		pre := fmt.Sprintf("p%d.", i+1)
		lb.Life[i] = lb.readBar(life, pre)
		lb.Power[i] = lb.readBar(power, pre)
		lb.Name[i] = readText(name, pre, pre+"name.")
	}
	lb.Time = readText(secs["time"], "", "counter.")
	return lb, nil
}

func (lb *Lifebar) readBar(s section, pre string) Bar {
	var b Bar
	if s == nil {
		return b
	}
	s.ints(pre+"pos", &b.Pos[0], &b.Pos[1])
	s.ints(pre+"range.x", &b.RangeX[0], &b.RangeX[1])
	s.ints(pre+"range.y", &b.RangeY[0], &b.RangeY[1])
	for i := range b.Bg {
		b.Bg[i] = lb.readElement(s, fmt.Sprintf("%sbg%d.", pre, i))
	}
	b.Mid = lb.readElement(s, pre+"mid.")
	b.Front = lb.readElement(s, pre+"front.")
	b.Top = lb.readElement(s, pre+"top.")
	return b
}

// readElement is ReadAnimLayout (common.go:1521-1550): spr sets a one-frame
// sprite, anim replaces it when the action exists, then the layout keys.
func (lb *Lifebar) readElement(s section, pre string) Element {
	e := Element{Anim: -1, Facing: 1, VFacing: 1, Scale: [2]float64{1, 1}}
	if s.ints(pre+"spr", &e.Sprite[0], &e.Sprite[1]) {
		e.set = true
	}
	var n int
	if s.ints(pre+"anim", &n) {
		if _, ok := lb.Anims[n]; ok {
			e.Anim = n
			e.set = true
		}
	}
	s.floats(pre+"offset", &e.Offset[0], &e.Offset[1])
	if v, ok := s[pre+"facing"]; ok && v != "" && atoi(v) < 0 {
		e.Facing = -1
	}
	if v, ok := s[pre+"vfacing"]; ok && v != "" && atoi(v) < 0 {
		e.VFacing = -1
	}
	s.ints(pre+"layerno", &e.Layer)
	e.Layer = min(2, e.Layer)
	s.floats(pre+"scale", &e.Scale[0], &e.Scale[1])
	return e
}

func readText(s section, posPre, textPre string) Text {
	var t Text
	if s == nil {
		return t
	}
	s.ints(posPre+"pos", &t.Pos[0], &t.Pos[1])
	s.floats(textPre+"offset", &t.Offset[0], &t.Offset[1])
	var bank int
	if s.ints(textPre+"font", &t.Font, &bank, &t.Align) {
		t.set = true
	}
	return t
}

// Frame returns the sprite an element shows ticks frames after the match
// started, and false when it shows nothing (blank frame or no sprite).
func (lb *Lifebar) Frame(e Element, ticks int) (air.Frame, bool) {
	if !e.set {
		return air.Frame{}, false
	}
	if e.Anim < 0 {
		return air.Frame{Group: e.Sprite[0], Image: e.Sprite[1]}, true
	}
	fr, ok := animFrame(lb.Anims[e.Anim], ticks)
	if !ok || fr.IsBlank() {
		return air.Frame{}, false
	}
	return fr, true
}

// animFrame is the frame of a showing ticks into its playback, looping from
// LoopStart; a frame of time -1 holds forever.
func animFrame(a air.Animation, ticks int) (air.Frame, bool) {
	if len(a.Frames) == 0 {
		return air.Frame{}, false
	}
	total := 0
	for _, fr := range a.Frames {
		if fr.Time < 0 {
			break
		}
		total += fr.Time
	}
	t := ticks
	if t >= total {
		loop := 0
		for _, fr := range a.Frames[min(a.LoopStart, len(a.Frames)-1):] {
			loop += max(fr.Time, 0)
		}
		if loop <= 0 {
			return a.Frames[len(a.Frames)-1], true
		}
		start := total - loop
		t = start + (t-start)%loop
	}
	acc := 0
	for _, fr := range a.Frames {
		if fr.Time < 0 {
			return fr, true
		}
		acc += fr.Time
		if t < acc {
			return fr, true
		}
	}
	return a.Frames[len(a.Frames)-1], true
}

// Scale is the factor from the lifebar's localcoord to a game view vw wide,
// for a 4:3 view (Ikemen GO FightScreen.setScale, fightscreen.go:5972: with
// a 4:3 game both viewports match, offsetX is 0 and scale is 320/localW
// times the view's own scale from 320).
func (lb *Lifebar) Scale(vw float64) float64 {
	if lb.Localcoord[0] <= 0 {
		return vw / float64(DefaultLocalcoord[0])
	}
	return vw / float64(lb.Localcoord[0])
}

// FillSpan is Ikemen GO calcBarFillRect (fightscreen.go:418) for one axis:
// the visible start and size, in view pixels, of a bar whose range is rng
// (descending when it empties toward its first value), filled at fill.
func FillSpan(pos int, rng [2]int, scale, fill float64) (start, size int) {
	r0, r1 := rng[0], rng[1]
	descending := r0 > r1
	if descending {
		r0, r1 = r1, r0
	}
	size = int(float64(r1-r0+1)*scale*fill + 0.5)
	start = int(float64(pos+r0)*scale + 0.5)
	if descending {
		start = int(float64(pos+r1+1)*scale+0.5) - size
	}
	return start, size
}
