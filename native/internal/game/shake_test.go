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
