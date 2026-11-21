package gfx

import (
	"github.com/openkakutou/engine/v2/evaluator"
	"github.com/openkakutou/lifebar"
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
	if got := callText("Round %i", 3, ""); got != "Round 3" {
		t.Errorf("got %q", got)
	}
	if got := callText("%s Wins", 3, "Kung Fu Man"); got != "Kung Fu Man Wins" {
		t.Errorf("got %q", got)
	}
}

func TestAtValue_PicksTheBarsVariants(t *testing.T) {
	b := lifebar.Bar{Front: lifebar.Element{Anim: 1}, FrontVariants: []lifebar.Variant{{From: 50, Element: lifebar.Element{Anim: 2}}}}
	if got := atValue(b, 60, 100).Front.Anim; got != 2 {
		t.Errorf("at 60%%: front anim %d, want 2", got)
	}
	if got := atValue(b, 40, 100).Front.Anim; got != 1 {
		t.Errorf("at 40%%: front anim %d, want 1", got)
	}
}

func TestSpriteBlend_FollowsTheExplodsTrans(t *testing.T) {
	cases := []struct {
		trans, frame string
		blend        Blend
		alpha        float32
	}{
		{"add", "", BlendAdd, 1},
		{"addalpha", "", BlendNormal, 128.0 / 255},
		{"sub", "", BlendSub, 1},
		{"none", "A", BlendNormal, 1},
		{"", "A", BlendAdd, 1},
		{"", "", BlendNormal, 1},
	}
	for _, c := range cases {
		b, a := spriteBlend(c.trans, [2]int{128, 128}, c.frame)
		if b != c.blend || a != c.alpha {
			t.Errorf("%q over %q: %v %v, want %v %v", c.trans, c.frame, b, a, c.blend, c.alpha)
		}
	}
}

// char.go:13533-13540: an AngleDraw turns and scales the drawing, its angle
// negated facing left; without one the context draws at its own size.
func TestAngleDrawn(t *testing.T) {
	c := evaluator.Context{AngleDraw: true, Angle: 30, AngleDrawScale: [2]float64{2, 1}}
	if at := angleDrawn(match.Position{}, match.FacingRight, &c); at.angle != 30 || at.scale != [2]float64{2, 1} {
		t.Errorf("facing right: %+v", at)
	}
	if at := angleDrawn(match.Position{}, match.FacingLeft, &c); at.angle != -30 {
		t.Errorf("facing left: angle %v, want -30", at.angle)
	}
	c.AngleDraw = false
	if at := angleDrawn(match.Position{}, match.FacingRight, &c); at.angle != 0 || at.scale != [2]float64{1, 1} {
		t.Errorf("no AngleDraw: %+v", at)
	}
}
