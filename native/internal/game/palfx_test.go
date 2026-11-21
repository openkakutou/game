package game

import (
	"testing"

	"github.com/openkakutou/engine/v2/evaluator"
)

func TestPalFX_TintsForItsTime(t *testing.T) {
	var p PalFX
	p.Start(evaluator.ControllerEvent{Type: "PalFX", Params: []evaluator.EventParam{
		{Name: "time", Values: []float64{2}}, {Name: "add", Values: []float64{255, 0, 51}}, {Name: "mul", Values: []float64{128, 256, 256}},
	}})
	mul, add := p.Tint()
	if mul != [3]float32{0.5, 1, 1} || add != [3]float32{1, 0, 0.2} {
		t.Errorf("tint = %v %v", mul, add)
	}
	p.Step()
	p.Step()
	if p.Active() {
		t.Error("still active after its time")
	}
	if mul, add := p.Tint(); mul != [3]float32{1, 1, 1} || add != [3]float32{} {
		t.Errorf("inactive tint = %v %v, want the identity", mul, add)
	}
}

func TestPalFX_ForEverAndDefaults(t *testing.T) {
	var p PalFX
	p.Start(evaluator.ControllerEvent{Type: "PalFX", Params: []evaluator.EventParam{{Name: "time", Values: []float64{-1}}}})
	for range 100 {
		p.Step()
	}
	if !p.Active() || p.Mul != [3]float64{256, 256, 256} {
		t.Errorf("time -1: active %v mul %v", p.Active(), p.Mul)
	}
}
