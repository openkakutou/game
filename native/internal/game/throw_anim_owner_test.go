package game

import (
	"testing"

	"github.com/openkakutou/engine/v2/evaluator"
	"github.com/openkakutou/engine/v2/input"
	"github.com/openkakutou/engine/v2/match"
)

// throwKfm plays kfm throwing kfm: P1 walks into P2, then holds forward and
// y (kfm.cmd "Kung Fu Throw"). P2 runs an endless afterimage. It returns
// the match on the first tick P2 shows P1's animation (kfm.cns state 820,
// ChangeAnim2), failing when the throw never comes.
func throwKfm(t *testing.T) *Match {
	t.Helper()
	a, b := kfmPair(t)
	m, err := NewMatch(a, b, wideStage(), Setup{Rounds: 1})
	if err != nil {
		t.Fatal(err)
	}
	untilFight(t, m)
	m.AfterImages[match.SideP2].Start(evaluator.ControllerEvent{Type: "AfterImage", Params: []evaluator.EventParam{{Name: "time", Values: []float64{-1}}}})
	walk := input.TickInput{Right: true}
	throw := input.TickInput{Right: true, Buttons: map[string]bool{"y": true}}
	for n := 0; n < 600; n++ {
		in := walk
		if n%2 == 1 && m.State.Fighters[match.SideP2].Position.X-m.State.Fighters[match.SideP1].Position.X <= 34 {
			in = throw
		}
		if err := m.Step([2]input.TickInput{in, {}}); err != nil {
			t.Fatalf("tick %d: %v", n+1, err)
		}
		if m.runtimes[match.SideP2].Context.AnimFromOpponent {
			return m
		}
	}
	t.Fatal("P1 never threw P2 (P2 never showed P1's animation)")
	return nil
}

// A thrown kfm shows the thrower's animation (ChangeAnim2): the match's
// animation and the afterimage ghost recorded that tick both name P1 as the
// owner of the animations and sprites to draw.
func TestMatch_AThrownKfmsAnimationAndGhostNameTheThrower(t *testing.T) {
	m := throwKfm(t)
	if got := m.Anim[match.SideP2]; got.Owner != match.SideP1 || got.No != m.runtimes[match.SideP2].Context.Anim {
		t.Errorf("P2's animation = %+v, want anim %d owned by P1", got, m.runtimes[match.SideP2].Context.Anim)
	}
	a := &m.AfterImages[match.SideP2]
	last := a.ring[(a.idx+len(a.ring)-1)%len(a.ring)]
	if last.Anim != m.Anim[match.SideP2] {
		t.Errorf("last ghost = %+v, want P2's animation %+v", last.Anim, m.Anim[match.SideP2])
	}
}
