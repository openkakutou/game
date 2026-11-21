package gfx

import (
	"testing"

	"github.com/openkakutou/engine/v2/match"
	"github.com/openkakutou/engine/v2/round"
)

// The engine's Y points up (engine statemachine worldY): a fighter 30 units
// in the air is drawn 30 view pixels above the ground line, not below it.
func TestFighterScreenY_AirborneIsAboveTheGround(t *testing.T) {
	if got := fighterScreenY(180, 30); got != 150 {
		t.Errorf("y 30 over ground 180 = %v, want 150", got)
	}
	if got := fighterScreenY(180, 0); got != 180 {
		t.Errorf("on the ground = %v, want 180", got)
	}
}

// A frame's blend in the .air (Ikemen GO anim.go): A, A1 and AS…D… add the
// sprite to what is behind, so its black stays invisible.
func TestAdditiveBlend(t *testing.T) {
	for blend, want := range map[string]bool{"A": true, "a1": true, "AS256D256": true, "": false, "S": false, " A ": true} {
		if got := additiveBlend(blend); got != want {
			t.Errorf("additiveBlend(%q) = %v, want %v", blend, got, want)
		}
	}
}

func TestBanner_FollowsTheRound(t *testing.T) {
	const total = 60 * 99
	for _, c := range []struct {
		outcome      round.Outcome
		training     bool
		state, timer int
		want         string
	}{
		{round.OutcomeNone, false, 0, total, "Round 2"},
		{round.OutcomeNone, false, 1, total, "Round 2"},
		{round.OutcomeNone, false, 2, total - 10, "Fight!"},
		{round.OutcomeNone, false, 2, total - 200, ""},
		{round.OutcomeKO, false, 3, 1000, "KO"},
		{round.OutcomeTimeout, false, 4, 0, "Time over"},
		{round.OutcomeNone, true, 2, total, ""},
	} {
		if got := banner(c.outcome, c.training, c.state, 2, c.timer, total); got != c.want {
			t.Errorf("%+v: banner %q, want %q", c, got, c.want)
		}
	}
}

func TestSuperDarkness(t *testing.T) {
	cases := []struct {
		s    match.MatchState
		want float32
	}{
		{match.MatchState{SuperTime: 10, SuperDarken: true}, 0.5},
		{match.MatchState{SuperTime: 10}, 0},
		{match.MatchState{SuperDarken: true}, 0},
	}
	for _, c := range cases {
		if got := superDarkness(c.s); got != c.want {
			t.Errorf("%+v: %v, want %v", c.s, got, c.want)
		}
	}
}

func TestCallText_PutsTheRoundNumber(t *testing.T) {
	if got := callText("Round %i", 3); got != "Round 3" {
		t.Errorf("got %q", got)
	}
	if got := callText("Fight!", 3); got != "Fight!" {
		t.Errorf("got %q", got)
	}
}
