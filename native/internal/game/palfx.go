package game

import (
	"math"

	"github.com/openkakutou/engine/v2/evaluator"
)

// PalFX is a fighter's palette effect (Ikemen GO PalFX, char.go palfx): for
// Time ticks (-1 for ever) its colours are multiplied by Mul/256 and Add is
// added, SinAdd adding a sine of period SinPeriod ticks on top.
// shortcut: color (desaturation) and invertall are not drawn.
type PalFX struct {
	Time      int
	Add, Mul  [3]float64
	SinAdd    [3]float64
	SinPeriod int
	elapsed   int
}

// Start begins the effect a PalFX event asks for: time, add (0,0,0), mul
// (256,256,256), sinadd (r, g, b, period).
func (p *PalFX) Start(e evaluator.ControllerEvent) {
	*p = PalFX{Time: 0, Mul: [3]float64{256, 256, 256}}
	if v := e.Param("time").Values; len(v) > 0 {
		p.Time = int(v[0])
	}
	copy(p.Add[:], e.Param("add").Values)
	if v := e.Param("mul").Values; len(v) > 0 {
		copy(p.Mul[:], v)
	}
	if v := e.Param("sinadd").Values; len(v) > 0 {
		copy(p.SinAdd[:], v)
		if len(v) > 3 {
			p.SinPeriod = int(v[3])
		}
	}
}

// Step counts the effect down one tick.
func (p *PalFX) Step() {
	if p.Time > 0 {
		p.Time--
		p.elapsed++
	}
}

// Active reports the effect showing.
func (p *PalFX) Active() bool { return p.Time != 0 }

// Tint is the effect now, as multipliers (0..1+) and additions (0..1) of
// red, green and blue; the identity when it is not active.
func (p *PalFX) Tint() (mul, add [3]float32) {
	mul = [3]float32{1, 1, 1}
	if !p.Active() {
		return mul, add
	}
	sin := 0.0
	if p.SinPeriod > 0 {
		sin = math.Sin(2 * math.Pi * float64(p.elapsed) / float64(p.SinPeriod))
	}
	for i := range 3 {
		mul[i] = float32(p.Mul[i] / 256)
		add[i] = float32((p.Add[i] + p.SinAdd[i]*sin) / 255)
	}
	return mul, add
}
