package gfx

import (
	"testing"

	"github.com/openkakutou/character"
	"github.com/openkakutou/engine/v2/match"
	"github.com/openkakutou/engine/v2/round"
	"github.com/openkakutou/game/native/internal/game"
	"github.com/openkakutou/lifebar"
)

// The life bar's fill is the fighter's life over its maximum, clamped to
// [0, 1]; a maximum of 0 or below draws an empty bar instead of dividing by 0.
func TestLifeFraction_ClampsAndGuardsTheMaximum(t *testing.T) {
	for _, c := range []struct {
		name          string
		life, lifeMax int
		want          float64
	}{
		{"half", 500, 1000, 0.5},
		{"below zero", -30, 1000, 0},
		{"above the maximum", 1200, 1000, 1},
		{"maximum of zero", 500, 0, 0},
		{"negative maximum", 500, -10, 0},
	} {
		if got := lifeFraction(c.life, c.lifeMax); got != c.want {
			t.Errorf("%s: lifeFraction(%d, %d) = %v, want %v", c.name, c.life, c.lifeMax, got, c.want)
		}
	}
}

// Each win type shows its own [WinIcon] element; a type the lifebar has no
// element for falls back to the normal win icon.
func TestWinIcon_PicksTheElementOfEachWinType(t *testing.T) {
	icon := func(n int) lifebar.Element { return lifebar.Element{HasAnim: true, Anim: n} }
	icons := lifebar.WinIcons{
		Normal: icon(1), Special: icon(2), Hyper: icon(3), Cheese: icon(4),
		Time: icon(5), Throw: icon(6), Suicide: icon(7),
	}
	for _, c := range []struct {
		t    game.WinType
		want lifebar.Element
	}{
		{game.WinNormal, icons.Normal},
		{game.WinSpecial, icons.Special},
		{game.WinHyper, icons.Hyper},
		{game.WinCheese, icons.Cheese},
		{game.WinTime, icons.Time},
		{game.WinThrow, icons.Throw},
		{game.WinSuicide, icons.Suicide},
		{game.WinType(99), icons.Normal},
	} {
		if got := winIcon(icons, c.t); got != c.want {
			t.Errorf("winIcon(%d) = %+v, want %+v", c.t, got, c.want)
		}
	}
}

// The win call's %s is the round winner's name; a round without a winner
// (draw, double KO, still running), a side out of range or a fighter without
// a character gives no name.
func TestWinnerName_NamesOnlyARealWinner(t *testing.T) {
	fighter := func(name string) *game.Fighter { return &game.Fighter{Char: &character.Character{Name: name}} }
	both := [2]*game.Fighter{fighter("Kung Fu Man"), fighter("Rival")}
	for _, c := range []struct {
		name     string
		fighters [2]*game.Fighter
		r        round.RoundResult
		want     string
	}{
		{"KO by P2", both, round.RoundResult{Outcome: round.OutcomeKO, Winner: match.SideP2}, "Rival"},
		{"time over won by P1", both, round.RoundResult{Outcome: round.OutcomeTimeout, Winner: match.SideP1}, "Kung Fu Man"},
		{"time over draw", both, round.RoundResult{Outcome: round.OutcomeTimeoutDraw}, ""},
		{"double KO", both, round.RoundResult{Outcome: round.OutcomeDoubleKO}, ""},
		{"round still running", both, round.RoundResult{}, ""},
		{"side out of range", both, round.RoundResult{Outcome: round.OutcomeKO, Winner: match.Side(2)}, ""},
		{"negative side", both, round.RoundResult{Outcome: round.OutcomeKO, Winner: match.Side(-1)}, ""},
		{"no character", [2]*game.Fighter{{}, fighter("Rival")}, round.RoundResult{Outcome: round.OutcomeKO, Winner: match.SideP1}, ""},
	} {
		if got := winnerName(c.fighters, c.r); got != c.want {
			t.Errorf("%s: winnerName = %q, want %q", c.name, got, c.want)
		}
	}
}
