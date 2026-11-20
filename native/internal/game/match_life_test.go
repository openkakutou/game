package game

import (
	"strconv"
	"strings"
	"testing"

	"github.com/openkakutou/engine/v2/evaluator"
)

func withLife(t *testing.T, f *Fighter, life int) *Fighter {
	t.Helper()
	c, err := evaluator.ParseConstants(strings.NewReader("[Data]\nlife = " + strconv.Itoa(life) + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	g := *f
	g.Program.Constants = c
	return &g
}

// system.go:4419-4423: a fighter starts round 1 with its lifeMax, its [Data]
// life; -p<n>.life is clamped to it.
func TestMatch_EachFighterStartsWithItsOwnLife(t *testing.T) {
	a, b := loadPair(t)
	m, err := NewMatch(withLife(t, a, 1200), withLife(t, b, 800), closeStage(), Setup{Rounds: 1})
	if err != nil {
		t.Fatal(err)
	}
	if m.State.Fighters[0].Health != 1200 || m.State.Fighters[1].Health != 800 {
		t.Errorf("health = %d, %d; want 1200, 800", m.State.Fighters[0].Health, m.State.Fighters[1].Health)
	}
	if m.LifeMax != [2]int{1200, 800} {
		t.Errorf("LifeMax = %v, want [1200 800]", m.LifeMax)
	}
	m, _ = NewMatch(withLife(t, a, 800), withLife(t, b, 800), closeStage(), Setup{Rounds: 1, Start: [2]StartValues{{Life: 1000}}})
	if m.State.Fighters[0].Health != 800 {
		t.Errorf("-p1.life 1000 on an 800-life character = %d, want clamped to 800", m.State.Fighters[0].Health)
	}
	if err := m.NextRound(); err != nil {
		t.Fatal(err)
	}
	if m.State.Fighters[1].Health != 800 {
		t.Errorf("next round health = %d, want 800", m.State.Fighters[1].Health)
	}
}
