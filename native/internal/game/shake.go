package game

import (
	"math"

	"github.com/openkakutou/engine/v2/evaluator"
)

// Shake is the screen shake of EnvShake (Ikemen GO EnvShake, system.go:6975-
// 7061): Time ticks of a sine of Ampl at Freq degrees a tick from Phase, its
// amplitude multiplied by Mul each cycle.
// shortcut: dir, diradd and decay are not read: the shake is vertical.
type Shake struct {
	Time, left             int
	Freq, Ampl, Phase, Mul float64
	offset                 float64
}

// Start begins the shake an EnvShake event asks for: time, freq (60), ampl
// (-4), phase (90 from a freq of 90, else 0) and mul (1).
func (s *Shake) Start(e evaluator.ControllerEvent) {
	v := func(name string, def float64) float64 {
		if vs := e.Param(name).Values; len(vs) > 0 {
			return vs[0]
		}
		return def
	}
	*s = Shake{Time: int(v("time", 0)), Freq: v("freq", 60), Ampl: v("ampl", -4), Mul: v("mul", 1)}
	s.Phase = 0
	if s.Freq >= 90 {
		s.Phase = 90
	}
	s.Phase = v("phase", s.Phase)
	s.left = s.Time
}

// Step computes this tick's offset and counts the shake down (update).
func (s *Shake) Step() {
	if s.left <= 0 {
		s.offset = 0
		return
	}
	elapsed := float64(s.Time - s.left)
	ampl := s.Ampl
	if s.Mul != 0 && s.Mul != 1 {
		ampl *= math.Pow(s.Mul, math.Floor(s.Freq*elapsed/360))
	}
	s.offset = ampl * math.Sin((s.Phase+s.Freq*elapsed)*math.Pi/180)
	s.left--
}

// Offset is the screen's vertical offset this tick, in stage units (down
// positive).
func (s *Shake) Offset() float64 { return s.offset }
