package game

import (
	"math"
	"strings"

	"github.com/openkakutou/engine/v2/evaluator"
	"github.com/openkakutou/engine/v2/match"
)

// maxAfterImageLength is Ikemen GO's MaxAimgLength.
const maxAfterImageLength = 60

// Ghost is one recorded frame of an afterimage: the fighter's animation and
// how far into it, where and facing which way.
type Ghost struct {
	Anim, AnimTime int
	Position       match.Position
	Facing         match.Facing
}

// GhostDraw is a ghost to draw, tinted as a PalFX: colours multiplied by Mul
// then Add added (0..1 scale).
type GhostDraw struct {
	Ghost
	Mul, Add [3]float32
}

// AfterImage is a fighter's afterimage trail (Ikemen GO AfterImage,
// char.go:1413-1790): every timegap ticks it records the fighter's frame;
// every framegap-th record shows behind the fighter, its colours darkening
// step by step (palbright + paladd per step, palcontrast times palmul per
// step), for time ticks.
// shortcut: palcolor, palinvertall, palhueshift and the alpha of trans are
// not applied; trans add / add1 draws additively.
type AfterImage struct {
	time, length, timeGap, frameGap int
	bright, contrast, postBright    [3]float64
	add, mul                        [3]float64
	Additive                        bool
	ring                            []Ghost
	idx, recCount, restGap, count   int
}

// Start begins the trail an AfterImage event asks for (afterImage.Run,
// bytecode.go:8040-8090): time 1, length 20, palbright 30,30,30,
// palcontrast 120,120,220, palpostbright 0, paladd 10,10,25, palmul
// .65,.65,.75, timegap 1, framegap 4, unless given.
func (a *AfterImage) Start(e evaluator.ControllerEvent) {
	*a = AfterImage{time: 1, length: 20, timeGap: 1, frameGap: 4,
		bright: [3]float64{30, 30, 30}, contrast: [3]float64{120, 120, 220},
		add: [3]float64{10, 10, 25}, mul: [3]float64{0.65, 0.65, 0.75}}
	ints := []struct {
		name string
		dst  *int
	}{{"time", &a.time}, {"length", &a.length}, {"timegap", &a.timeGap}, {"framegap", &a.frameGap}}
	for _, p := range ints {
		if v := e.Param(p.name).Values; len(v) > 0 {
			*p.dst = int(v[0])
		}
	}
	triples := []struct {
		name string
		dst  *[3]float64
	}{{"palbright", &a.bright}, {"palcontrast", &a.contrast}, {"palpostbright", &a.postBright}, {"paladd", &a.add}, {"palmul", &a.mul}}
	for _, p := range triples {
		copy(p.dst[:], e.Param(p.name).Values)
	}
	trans := strings.ToLower(e.Param("trans").Raw)
	a.Additive = strings.HasPrefix(trans, "add")
	a.length = min(max(a.length, 1), maxAfterImageLength)
	a.ring = make([]Ghost, a.length)
}

// SetTime is AfterImageTime (bytecode.go:8090-8104): time (1 counts as 0)
// starts the trail's time over.
func (a *AfterImage) SetTime(e evaluator.ControllerEvent) {
	t := 0
	if v := e.Param("time").Values; len(v) > 0 {
		t = int(v[0])
	}
	if t == 1 {
		t = 0
	}
	a.time, a.count = t, 0
}

// Active reports the trail running (isActive, char.go:1697-1704).
func (a *AfterImage) Active() bool {
	if a.time == 0 || a.timeGap < 1 || a.frameGap < 1 || len(a.ring) == 0 {
		return false
	}
	return a.time < 0 || a.count < a.timeGap*a.length+a.time-1
}

// Record records the fighter's frame this tick (recAfterImg,
// char.go:1587-1695).
func (a *AfterImage) Record(g Ghost) {
	if !a.Active() {
		return
	}
	if a.restGap <= 0 {
		a.ring[a.idx] = g
		a.idx = (a.idx + 1) % len(a.ring)
		a.recCount++
		a.restGap = a.timeGap
	}
	a.restGap--
	a.count++
}

// Ghosts appends to dst the ghosts to draw, newest first (recAndCue,
// char.go:1706-1790). mugen is a character without ikemenversion: MUGEN
// samples one record closer to the present.
func (a *AfterImage) Ghosts(dst []GhostDraw, mugen bool) []GhostDraw {
	dst = dst[:0]
	if !a.Active() {
		return dst
	}
	usable := min(a.recCount-1, a.length)
	if mugen {
		usable = min(a.recCount, a.length-1)
	}
	steps := max(usable, 0) / a.frameGap
	add, mul := a.bright, a.contrast
	for step := range steps {
		if step > 0 {
			for j := range 3 {
				add[j] += a.add[j]
				mul[j] *= a.mul[j]
			}
			if step == 1 {
				for j := range 3 {
					add[j] += a.postBright[j]
				}
			}
		}
		i := (step + 1) * a.frameGap
		idx := a.idx - i - 1
		if mugen {
			idx++
		}
		idx = ((idx % len(a.ring)) + len(a.ring)) % len(a.ring)
		if a.time > 0 && a.count/a.timeGap-i >= (a.time-2)/a.timeGap+1 {
			continue
		}
		var d GhostDraw
		d.Ghost = a.ring[idx]
		for j := range 3 {
			d.Mul[j] = float32(math.Floor(mul[j]) / 256)
			d.Add[j] = float32(add[j] / 255)
		}
		dst = append(dst, d)
	}
	return dst
}
