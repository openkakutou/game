package game

import (
	"testing"

	sharedlifebar "github.com/openkakutou/lifebar"
)

func comboCfg() sharedlifebar.Combo {
	return sharedlifebar.Combo{StartX: -40, DisplayTime: 10, ShowSpeed: 2, HideSpeed: 40}
}

// fs.go:2616-2704: hidden at start.x*2, slides in from a 2-hit combo, stays
// displaytime ticks once shown, then slides back out.
func TestComboCounter_SlidesInStaysThenOut(t *testing.T) {
	cfg := comboCfg()
	var c ComboCounter
	c.Step(cfg, 320, 0, 1000)
	if c.Visible(cfg) || c.X != -80 {
		t.Fatalf("idle counter visible %v at %v, want hidden at -80", c.Visible(cfg), c.X)
	}
	c.Step(cfg, 320, 1, 950)
	c.Step(cfg, 320, 2, 900)
	if !c.Visible(cfg) || c.Hits != 2 || c.Damage != 50 {
		t.Fatalf("2-hit combo: visible %v hits %d damage %d, want shown 2 hits for 50", c.Visible(cfg), c.Hits, c.Damage)
	}
	ticks := 0
	for ; c.X != 0 && ticks < 50; ticks++ {
		c.Step(cfg, 320, 2, 900)
	}
	if c.X != 0 {
		t.Fatal("never slid in")
	}
	for range 30 {
		c.Step(cfg, 320, 0, 900)
	}
	if c.Visible(cfg) {
		t.Errorf("still visible at %v after the combo ended and its display time", c.X)
	}
}

func TestComboCounter_OneHitShowsNothing(t *testing.T) {
	cfg := comboCfg()
	var c ComboCounter
	for range 20 {
		c.Step(cfg, 320, 1, 900)
	}
	if c.Visible(cfg) || c.Hits != 0 {
		t.Errorf("single hit shown: %+v", c)
	}
}

func TestMatch_ComboHitsAreTheOpponentsWhileItIsHit(t *testing.T) {
	m := &Match{}
	m.runtimes[1].Context.MoveType = 'H'
	m.runtimes[1].Context.GetHit.HitCount = 3
	if got := m.ComboHits(0); got != 3 {
		t.Errorf("P1 combo = %d, want 3", got)
	}
	m.runtimes[1].Context.MoveType = 'I'
	if got := m.ComboHits(0); got != 0 {
		t.Errorf("opponent recovered: combo = %d, want 0", got)
	}
	if got := m.ComboHits(1); got != 0 {
		t.Errorf("P2 combo = %d, want 0", got)
	}
}

// fs.go:5598: a knocked out opponent lying in 5150 ends the combo.
func TestMatch_ComboEndsOnAnOpponentLyingKO(t *testing.T) {
	m := &Match{}
	opp := &m.runtimes[1].Context
	opp.MoveType, opp.GetHit.HitCount, opp.Health = 'H', 4, 0
	opp.StateNo = 5100
	if got := m.ComboHits(0); got != 4 {
		t.Errorf("falling KO: combo = %d, want 4", got)
	}
	opp.StateNo = koLieState
	if got := m.ComboHits(0); got != 0 {
		t.Errorf("lying KO: combo = %d, want 0", got)
	}
}
