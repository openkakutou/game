package game

import "github.com/openkakutou/engine/v2/evaluator"

// EnvColor is the screen fill of an EnvColor (Ikemen GO envColor.Run,
// system.go:2888, 3817-3830): Color (255,255,255) for Time ticks (1; -1 for
// ever) over the stage, the fighters shown over it only when Under.
type EnvColor struct {
	Color [3]int
	Time  int
	Under bool
}

// Start begins the fill an EnvColor event asks for.
func (e *EnvColor) Start(ev evaluator.ControllerEvent) {
	*e = EnvColor{Color: [3]int{255, 255, 255}, Time: 1}
	if v := ev.Param("value").Values; len(v) >= 3 {
		e.Color = [3]int{int(v[0]), int(v[1]), int(v[2])}
	}
	if v := ev.Param("time").Values; len(v) > 0 {
		e.Time = int(v[0])
	}
	if v := ev.Param("under").Values; len(v) > 0 {
		e.Under = v[0] != 0
	}
}

// Active reports the fill showing.
func (e *EnvColor) Active() bool { return e.Time != 0 }

// Step counts the fill down a tick.
func (e *EnvColor) Step() {
	if e.Time > 0 {
		e.Time--
	}
}
