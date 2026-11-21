package game

import (
	"testing"

	"github.com/openkakutou/engine/v2/evaluator"
	"github.com/openkakutou/engine/v2/match"
)

func aiEvent(typ string, params map[string][]float64) evaluator.ControllerEvent {
	e := evaluator.ControllerEvent{Type: typ}
	for k, v := range params {
		e.Params = append(e.Params, evaluator.EventParam{Name: k, Values: v})
	}
	return e
}

// char.go:1706-1790: with framegap 2 every second record shows, newest
// first, darker each step (palcontrast * palmul^step, palbright + paladd *
// step).
func TestAfterImage_ShowsEveryFrameGapthRecord(t *testing.T) {
	var a AfterImage
	a.Start(aiEvent("AfterImage", map[string][]float64{"time": {-1}, "length": {8}, "framegap": {2}}))
	for x := range 10 {
		a.Record(Ghost{Position: match.Position{X: float64(x)}})
	}
	ghosts := a.Ghosts(nil, false)
	// 10 records, usable min(9, 8) = 8: 4 steps at 2, 4, 6, 8 records back
	// from the newest slot.
	if len(ghosts) != 4 {
		t.Fatalf("ghosts = %d, want 4", len(ghosts))
	}
	if ghosts[0].Position.X != 7 || ghosts[1].Position.X != 5 {
		t.Errorf("ghost positions %v, %v, want 7 then 5", ghosts[0].Position.X, ghosts[1].Position.X)
	}
	if ghosts[0].Mul != [3]float32{120.0 / 256, 120.0 / 256, 220.0 / 256} || ghosts[0].Add[0] != 30.0/255 {
		t.Errorf("first ghost tint %v %v, want palcontrast and palbright", ghosts[0].Mul, ghosts[0].Add)
	}
	if !(ghosts[1].Mul[0] < ghosts[0].Mul[0]) || !(ghosts[1].Add[2] > ghosts[0].Add[2]) {
		t.Errorf("second ghost %v %v, want darker mul and more add", ghosts[1].Mul, ghosts[1].Add)
	}
}

// A MUGEN character samples one record closer (issue 1053).
func TestAfterImage_MugenCharactersSampleOneCloser(t *testing.T) {
	var a AfterImage
	a.Start(aiEvent("AfterImage", map[string][]float64{"time": {-1}, "length": {8}, "framegap": {2}}))
	for x := range 10 {
		a.Record(Ghost{Position: match.Position{X: float64(x)}})
	}
	ghosts := a.Ghosts(nil, true)
	if len(ghosts) != 3 || ghosts[0].Position.X != 8 {
		t.Errorf("MUGEN ghosts = %d first at %v, want 3 starting at 8", len(ghosts), ghosts[0].Position.X)
	}
}

// AfterImageTime 0 (or 1) ends the trail; a timed trail ends on its own.
func TestAfterImage_EndsWithItsTime(t *testing.T) {
	var a AfterImage
	a.Start(aiEvent("AfterImage", map[string][]float64{"time": {-1}}))
	a.Record(Ghost{})
	a.SetTime(aiEvent("AfterImageTime", map[string][]float64{"time": {1}}))
	if a.Active() {
		t.Error("AfterImageTime 1 left the trail running")
	}
	a.Start(aiEvent("AfterImage", map[string][]float64{"time": {3}, "length": {2}}))
	for range 10 {
		a.Record(Ghost{})
	}
	if a.Active() {
		t.Error("a time 3 trail still running after 10 ticks")
	}
}
