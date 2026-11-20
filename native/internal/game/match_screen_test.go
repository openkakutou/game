package game

import (
	"os"
	"testing"

	"github.com/openkakutou/engine/v2/input"
	"github.com/openkakutou/stage"
)

// wideStage lets the fighters walk 1000 units each way, and the camera not
// at all: the screen, 320 wide around 0, holds them.
func wideStage() *stage.Stage {
	return &stage.Stage{StageBoundaries: stage.StageBoundaries{Left: -1000, Right: 1000}}
}

func kfmPair(t *testing.T) (*Fighter, *Fighter) {
	t.Helper()
	a, err := LoadFighter(os.DirFS(reference), "chars/kfm/kfm.def", "common")
	if err != nil {
		t.Fatal(err)
	}
	b, err := LoadFighter(os.DirFS(reference), "chars/kfm/kfm.def", "common")
	if err != nil {
		t.Fatal(err)
	}
	return a, b
}

// system.go:2871-2876: the screen edges are the camera's view less the
// stage's screenleft/screenright, 15 by default at a 320-wide localcoord.
func TestMatch_TheScreenEdgesFollowTheCameraLessTheMargins(t *testing.T) {
	p1, p2 := kfmPair(t)
	m, err := NewMatch(p1, p2, wideStage(), Setup{Rounds: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Screen(); got != [2]float64{-145, 145} {
		t.Fatalf("screen = %v, want [-145 145]", got)
	}
	left, right := 20, 30
	st := wideStage()
	st.StageBoundaries.ScreenLeft, st.StageBoundaries.ScreenRight = &left, &right
	m, _ = NewMatch(p1, p2, st, Setup{Rounds: 1})
	if got := m.Screen(); got != [2]float64{-140, 130} {
		t.Fatalf("screen with margins 20/30 = %v, want [-140 130]", got)
	}
}

func TestMatch_AFighterWalkingAwayStopsAtTheScreenEdge(t *testing.T) {
	p1, p2 := kfmPair(t)
	m, err := NewMatch(p1, p2, wideStage(), Setup{Rounds: 1})
	if err != nil {
		t.Fatal(err)
	}
	for range 400 {
		if err := m.Step([2]input.TickInput{{Left: true}, {}}); err != nil {
			t.Fatal(err)
		}
	}
	if x := m.State.Fighters[0].Position.X; x != -145 {
		t.Errorf("P1 x after walking left 400 ticks = %v, want held on the screen edge -145", x)
	}
}

func TestMatch_TheCameraFollowsTheFightersWithinItsBounds(t *testing.T) {
	p1, p2 := kfmPair(t)
	st := wideStage()
	st.CameraBounds.Left, st.CameraBounds.Right = -500, 500
	m, err := NewMatch(p1, p2, st, Setup{Rounds: 1})
	if err != nil {
		t.Fatal(err)
	}
	for range 400 {
		if err := m.Step([2]input.TickInput{{Left: true}, {Left: true}}); err != nil {
			t.Fatal(err)
		}
	}
	if m.CameraX() >= 0 {
		t.Errorf("camera x = %v after both fighters walked left, want it to follow them left", m.CameraX())
	}
	if s := m.Screen(); m.State.Fighters[0].Position.X < s[0] {
		t.Errorf("P1 at %v is off the screen %v", m.State.Fighters[0].Position.X, s)
	}
}
