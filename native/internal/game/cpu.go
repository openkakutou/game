package game

import (
	"math/rand/v2"

	"github.com/openkakutou/engine/v2/input"
)

// DefaultAILevel is Ikemen GO's default difficulty (Options.Difficulty 4).
const DefaultAILevel = 4

// cpuButtons are the buttons the CPU jams, in Ikemen GO's order (a b c x y z,
// then d and w, which OpenKakutou characters have no use for, then start).
var cpuButtons = [...]string{"a", "b", "c", "x", "y", "z"}

// CPU drives a side as Ikemen GO drives an AI player: it jams random
// directions and buttons, more often at a higher level (AiInput.Update,
// input.go:1941-2013, commit 54d4148a), and the match gives the side's
// state machine its level, so the character's own AI (its states gated on
// AILevel) plays too.
type CPU struct {
	// Level is the AI level, 1 to 8 (Ikemen GO's sys.aiLevel).
	Level float64
	draw  func(lo, hi int) int
	dir   int
	// dirTime, buttonTime and startTime count down the ticks left of a
	// jammed direction or button; 0 is released.
	dirTime    int
	buttonTime [len(cpuButtons)]int
	startTime  int
}

// NewCPU builds a CPU of the given level. draw returns an integer in
// [lo, hi], Ikemen GO's Rand; nil uses an unseeded source.
func NewCPU(level float64, draw func(lo, hi int) int) *CPU {
	if draw == nil {
		draw = func(lo, hi int) int { return lo + rand.IntN(hi-lo+1) }
	}
	return &CPU{Level: level, draw: draw}
}

// SeededDraw is a deterministic draw for NewCPU (tests, replays).
func SeededDraw(seed uint64) func(lo, hi int) int {
	r := rand.New(rand.NewPCG(seed, seed))
	return func(lo, hi int) int { return lo + r.IntN(hi-lo+1) }
}

// jam counts a timer down and, once it runs out, draws whether to press
// again (1 in chance) and for how many ticks (1 to duration); it reports a
// new press.
func (c *CPU) jam(t *int, chance, duration int) bool {
	*t--
	if *t > 0 {
		return false
	}
	*t = 0
	if c.draw(1, chance) != 1 {
		return false
	}
	*t = c.draw(1, duration)
	return true
}

// Decide advances the CPU one tick and writes its input into dst in place,
// reusing its button map.
func (c *CPU) Decide(dst *input.TickInput) {
	if c.jam(&c.dirTime, 15, 60) {
		c.dir = c.draw(0, 7) // a single direction, drawn with each press
	}
	chance := int((-11.25*c.Level + 165) * 7)
	for i := range c.buttonTime {
		c.jam(&c.buttonTime[i], chance, 30)
	}
	c.jam(&c.startTime, 3600, 30) // start is jammed less often

	clearInput(dst)
	if c.dirTime != 0 {
		// 0 U, 1 UR, 2 R, 3 DR, 4 D, 5 DL, 6 L, 7 UL (input.go:2000-2003).
		dst.Up = c.dir == 7 || c.dir == 0 || c.dir == 1
		dst.Down = c.dir == 3 || c.dir == 4 || c.dir == 5
		dst.Left = c.dir == 5 || c.dir == 6 || c.dir == 7
		dst.Right = c.dir == 1 || c.dir == 2 || c.dir == 3
	}
	for i, name := range cpuButtons {
		dst.Buttons[name] = c.buttonTime[i] != 0
	}
	dst.Buttons["s"] = c.startTime != 0
}

// clearInput releases every direction and button of dst in place, reusing its
// button map.
func clearInput(dst *input.TickInput) {
	dst.Up, dst.Down, dst.Left, dst.Right = false, false, false, false
	if dst.Buttons == nil {
		dst.Buttons = make(map[string]bool, 8)
	}
	for name := range dst.Buttons {
		dst.Buttons[name] = false
	}
}
