package game

import (
	"os"
	"testing"

	"github.com/openkakutou/character/air"
	"github.com/openkakutou/engine/v2/evaluator"
	"github.com/openkakutou/engine/v2/input"
	"github.com/openkakutou/engine/v2/match"
)

func sparkEvent(anim string, x, y float64) evaluator.ControllerEvent {
	return evaluator.ControllerEvent{Type: "HitSpark", Params: []evaluator.EventParam{
		{Name: "anim", Raw: anim}, {Name: "x", Values: []float64{x}}, {Name: "y", Values: []float64{y}},
	}}
}

// A 3 + 2 tick spark animation.
var shortSpark = []air.Animation{{Number: 2, Frames: []air.Frame{{Time: 3}, {Time: 2}}}}

func TestSparks_AHitSparkIsShownUntilItsAnimationEnds(t *testing.T) {
	var s Sparks
	s.Add([2][]evaluator.ControllerEvent{{sparkEvent("2", 10, 30)}, nil}, [2]match.Facing{match.FacingRight, match.FacingLeft})
	if len(s.Live) != 1 {
		t.Fatalf("sparks = %+v, want one", s.Live)
	}
	sp := s.Live[0]
	if sp.Anim != 2 || sp.Own || sp.Side != match.SideP1 || sp.X != 10 || sp.Y != 30 || sp.Facing != match.FacingRight {
		t.Errorf("spark = %+v", sp)
	}
	for range 4 {
		s.Step(shortSpark, [2][]air.Animation{})
	}
	if len(s.Live) != 1 || s.Live[0].Age != 4 {
		t.Fatalf("after 4 ticks: %+v, want still shown, age 4", s.Live)
	}
	s.Step(shortSpark, [2][]air.Animation{})
	if len(s.Live) != 0 {
		t.Errorf("after 5 ticks: %+v, want gone (animation of 5 ticks)", s.Live)
	}
}

func TestSparks_OwnSparksUseTheCharactersAnimationsAndUnknownOnesAreDropped(t *testing.T) {
	var s Sparks
	s.Add([2][]evaluator.ControllerEvent{nil, {sparkEvent("S2", 0, 0), sparkEvent("77", 0, 0), sparkEvent("x", 0, 0)}}, [2]match.Facing{})
	if len(s.Live) != 2 || !s.Live[0].Own || s.Live[0].Anim != 2 || s.Live[1].Anim != 77 {
		t.Fatalf("sparks = %+v, want own 2 then common 77 (x unreadable)", s.Live)
	}
	s.Step(nil, [2][]air.Animation{nil, shortSpark})
	if len(s.Live) != 1 || !s.Live[0].Own {
		t.Errorf("sparks = %+v, want the own spark kept, the common one without animation dropped", s.Live)
	}
}

// End to end: kfm's light kick on kfm makes the match show a spark.
func TestMatch_ALandedHitShowsItsSpark(t *testing.T) {
	a, b := kfmPair(t)
	m, err := NewMatch(a, b, wideStage(), Setup{Rounds: 1})
	if err != nil {
		t.Fatal(err)
	}
	m.State.Fighters[1].Position.X = m.State.Fighters[0].Position.X + 40
	m.runtimes[1].Context.Position.X = m.State.Fighters[1].Position.X
	raw, err := os.ReadFile("../../../common/fightfx.air")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetCommonAnimations(raw); err != nil {
		t.Fatal(err)
	}
	seen := false
	for i := 0; i < 40 && !seen; i++ {
		in := [2]input.TickInput{}
		if i == 0 {
			in[0] = input.TickInput{Buttons: map[string]bool{"a": true}}
		}
		if err := m.Step(in); err != nil {
			t.Fatal(err)
		}
		seen = len(m.Sparks.Live) > 0
	}
	if !seen {
		t.Fatal("the light kick never showed a spark")
	}
}
