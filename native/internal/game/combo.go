package game

import sharedlifebar "github.com/openkakutou/lifebar"

// koLieState is the state a knocked out fighter lies in (common1.cns 5150):
// Ikemen GO's SCF_over_ko, which ends the combo on it.
const koLieState = 5150

// ComboHits is the hits side's combo has landed on the opponent, while the
// opponent stays hit and is not lying knocked out (Ikemen GO receivedHits,
// fs.go:5593-5603).
// shortcut: guarded hits count, as the engine's ghv.hitcount does.
func (m *Match) ComboHits(side int) int {
	opp := &m.runtimes[1-side].Context
	if opp.MoveType != 'H' {
		return 0
	}
	if opp.Health <= 0 && opp.StateNo == koLieState {
		return 0
	}
	return opp.GetHit.HitCount
}

// ComboCounter is a side's lifebar combo counter (Ikemen GO
// FightScreenCombo.step, fs.go:2616-2704): it slides in once a combo
// reaches 2 hits, stays displaytime ticks after its last change, then slides
// out.
type ComboCounter struct {
	// Hits and Damage are the combo shown.
	Hits, Damage int
	// X is the counter's slide offset: 0 shown, start.x*2 hidden.
	X        float64
	rest     int
	newCombo bool
	started  bool
	life     int
}

// Step advances the counter a tick: hits are the combo's, life the
// opponent's life now.
func (c *ComboCounter) Step(cfg sharedlifebar.Combo, localWidth, hits, life int) {
	hidden := float64(cfg.StartX) * 2
	if !c.started {
		c.X, c.started = hidden, true
	}
	if hits < 2 && c.Hits >= 2 {
		c.newCombo = true
	}
	if hits <= 1 {
		c.life = life
	}
	switch {
	case c.rest > 0:
		c.X -= c.X / float64(cfg.ShowSpeed)
		if c.X > -1 && c.X < 1 {
			c.X = 0
		}
	case hits < 2:
		c.X -= float64(cfg.HideSpeed) * float64(localWidth) / 320
		if c.X < hidden {
			c.X = hidden
		}
	}
	if c.X == 0 {
		c.rest--
	}
	damage := max(c.life-life, 0)
	if hits >= 2 && (c.newCombo || c.Hits != hits || c.Damage != damage) {
		c.rest = cfg.DisplayTime
		c.Hits, c.Damage, c.newCombo = hits, damage, false
	}
}

// Visible reports the counter on screen.
func (c *ComboCounter) Visible(cfg sharedlifebar.Combo) bool {
	return c.started && (c.rest > 0 || c.X != float64(cfg.StartX)*2)
}
