package lifebar

// Meter is the per-player fill state of a life bar, stepped once per tick:
// Top is the front fill, Mid the trailing one (Ikemen GO LifeBar.step,
// fightscreen.go:554-603, with the defaults mid.freeze = 1, mid.delay = 30,
// mid.mult = 1, mid.steps = 8 of newLifeBar).
type Meter struct {
	Top, Mid    float64
	old, midMin float64
	delay       int
}

const (
	midDelay = 30
	midMult  = 1.0
	midSteps = 8.0
)

// NewMeter is a full bar.
func NewMeter() Meter { return Meter{Top: 1, Mid: 1, old: 1, midMin: 1} }

// Step advances one tick at life (0..1). gettingHit is Ikemen GO's getHit:
// the fighter is being hit, during which the trailing fill freezes. hasMid
// is whether the bar draws a mid element at all.
func (m *Meter) Step(life float64, gettingHit, hasMid bool) {
	if m.Top > life {
		m.Top += (life - m.Top) / 2
	} else {
		m.Top = life
	}
	if gettingHit && hasMid {
		if m.delay < midDelay {
			m.delay = midDelay
			m.Mid = m.old
			m.midMin = m.old
		}
	} else {
		if m.delay > 0 {
			m.delay--
		}
		if hasMid && m.delay <= 0 && life < m.midMin {
			m.midMin += (life - m.midMin) * (1 / (12 - (life-m.midMin)*144)) * midMult
			m.midMin = max(m.midMin, life)
		} else {
			m.midMin = life
		}
		if (!hasMid || m.delay <= 0) && m.Mid > m.midMin {
			m.Mid += (m.midMin - m.Mid) / midSteps
		}
		m.old = life
	}
	floor := max(m.midMin, life)
	if m.Mid < floor {
		m.Mid += (floor - m.Mid) / 2
	}
}
