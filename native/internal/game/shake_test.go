package game

import (
	"math"
	"testing"

	"github.com/openkakutou/engine/v2/evaluator"
)

func shakeEvent(params ...evaluator.EventParam) evaluator.ControllerEvent {
	return evaluator.ControllerEvent{Type: "EnvShake", Params: params}
}

func val(name string, v float64) evaluator.EventParam {
	return evaluator.EventParam{Name: name, Values: []float64{v}}
}

// system.go:7015-7058: ampl * sin(phase + freq * elapsed), for time ticks.
func TestShake_FollowsIkemenGOsSine(t *testing.T) {
	var s Shake
	s.Start(shakeEvent(val("time", 3), val("freq", 90), val("ampl", 4)))
	var got []float64
	for range 4 {
		s.Step()
		got = append(got, math.Round(s.Offset()*1000)/1000)
	}
	// freq 90 starts at phase 90: 4 sin(90), 4 sin(180), 4 sin(270), then over.
	want := []float64{4, 0, -4, 0}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("offsets = %v, want %v", got, want)
		}
	}
}

func TestShake_DefaultsAndMul(t *testing.T) {
	var s Shake
	s.Start(shakeEvent(val("time", 10)))
	if s.Freq != 60 || s.Ampl != -4 || s.Phase != 0 || s.Mul != 1 {
		t.Errorf("defaults = %+v", s)
	}
	s.Start(shakeEvent(val("time", 20), val("freq", 180), val("ampl", 8), val("mul", 0.5), val("phase", 90)))
	s.Step()
	s.Step()
	s.Step() // elapsed 2: one full cycle (360) passed, ampl halved
	if math.Abs(math.Abs(s.Offset())-4) > 1e-9 {
		t.Errorf("after a cycle offset = %v, want +-4", s.Offset())
	}
}

// system.go:2986-2997: after a KO the match runs at slow.speed, easing back
// to full speed over the last slow.fadetime; without one every frame steps.
func TestMatch_KOSlowMotion(t *testing.T) {
	m := &Match{Setup: Setup{Slow: SlowMotion{Time: 8, FadeTime: 4, Speed: 0.25}}}
	for range 3 {
		if !m.Advances() {
			t.Fatal("a frame without slow motion must step")
		}
	}
	m.slowLeft = 8
	var steps []bool
	for range 10 {
		steps = append(steps, m.Advances())
	}
	stepped := 0
	for _, s := range steps[:8] {
		if s {
			stepped++
		}
	}
	// 4 frames at 0.25, then 0.25+0.75*(0/4..3/4): 0.25,0.25,0.25,0.25 (1), 0.25 0.4375 0.625 0.8125 (2.125): 3 steps.
	if stepped != 3 {
		t.Errorf("steps in the slow motion = %d (%v), want 3", stepped, steps)
	}
	if !steps[8] || !steps[9] {
		t.Errorf("after the slow motion every frame steps, got %v", steps[8:])
	}
}
