package game

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/openkakutou/character/air"
	"github.com/openkakutou/engine/v2"
	"github.com/openkakutou/engine/v2/evaluator"
	"github.com/openkakutou/engine/v2/match"
)

// HitSpark is a spark shown where a hit landed (the engine's HitSpark
// event, Ikemen GO hitspark): an animation of the common fightfx, or of the
// attacker's own when Own (the S prefix), at a world position, Age ticks
// after it appeared.
type HitSpark struct {
	Anim   int
	Own    bool
	Side   match.Side
	X, Y   float64
	Facing match.Facing
	Age    int
}

// Sparks are the hit sparks being shown.
type Sparks struct {
	Live []HitSpark
}

// Add starts the sparks of one tick's events; facing is each side's facing.
func (s *Sparks) Add(events [2][]evaluator.ControllerEvent, facing [2]match.Facing) {
	for side, evs := range events {
		for _, e := range evs {
			if e.Type != "HitSpark" {
				continue
			}
			anim, own, ok := sparkAnim(e.Param("anim").Raw)
			x, y := e.Param("x").Values, e.Param("y").Values
			if !ok || len(x) != 1 || len(y) != 1 {
				continue
			}
			s.Live = append(s.Live, HitSpark{Anim: anim, Own: own, Side: match.Side(side), X: x[0], Y: y[0], Facing: facing[side]})
		}
	}
}

// Step ages every spark a tick and drops those whose animation is over or
// missing: common ones read common, own ones their side's animations.
func (s *Sparks) Step(common []air.Animation, own [2][]air.Animation) {
	kept := s.Live[:0]
	for _, sp := range s.Live {
		sp.Age++
		anims := common
		if sp.Own {
			anims = own[sp.Side]
		}
		if length, ok := animationLength(anims, sp.Anim); ok && sp.Age < length {
			kept = append(kept, sp)
		}
	}
	s.Live = kept
}

// sparkAnim reads a sparkno as written: S for the character's own
// animation, F or nothing for the common fightfx.
func sparkAnim(raw string) (anim int, own, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false, false
	}
	switch raw[0] {
	case 'S', 's':
		own, raw = true, raw[1:]
	case 'F', 'f':
		raw = raw[1:]
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	return n, own, err == nil && n >= 0
}

// sparkMaxTicks bounds a spark whose animation never ends (a frame of time
// -1): it is shown this long.
const sparkMaxTicks = 60

// animationLength is the ticks animation number lasts, false when missing.
func animationLength(anims []air.Animation, number int) (int, bool) {
	for _, a := range anims {
		if a.Number != number || len(a.Frames) == 0 {
			continue
		}
		total := 0
		for _, f := range a.Frames {
			if f.Time < 0 {
				return sparkMaxTicks, true
			}
			total += f.Time
		}
		return total, true
	}
	return 0, false
}

// SetCommonAnimations reads the common fightfx animations (common/
// fightfx.air) the match's common sparks play.
func (m *Match) SetCommonAnimations(raw []byte) error {
	anims, err := air.Parse(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	m.commonAnims = anims
	return nil
}

// CommonAnimations are the common fightfx animations the sparks play.
func (m *Match) CommonAnimations() []air.Animation { return m.commonAnims }

// Projectiles are side's live projectiles, playing its own animations.
func (m *Match) Projectiles(side match.Side) []engine.Projectile {
	return m.runtimes[side].Projectiles
}

// Helpers are side's live helpers, drawn like a fighter with its
// animations.
func (m *Match) Helpers(side match.Side) []engine.Helper {
	return m.runtimes[side].Helpers
}

// Explods are side's live explods.
func (m *Match) Explods(side match.Side) []engine.Explod {
	return m.runtimes[side].Explods
}
