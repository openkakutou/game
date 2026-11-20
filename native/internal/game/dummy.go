package game

import (
	"github.com/openkakutou/engine/v2/input"
	"github.com/openkakutou/engine/v2/match"
)

// Dummy is what the training dummy (side 2) does, chosen from the pause menu.
type Dummy int

// The dummy settings, in the order the pause menu cycles them. DummyCPU leaves
// the dummy to the CPU opponent.
const (
	DummyStand Dummy = iota
	DummyCrouch
	DummyJump
	DummyGuard
	DummyCPU
	dummyCount
)

// dummyLabels are the pause entry's texts, one per setting, so drawing the
// menu allocates nothing.
var dummyLabels = [dummyCount]string{"Dummy: Stand", "Dummy: Crouch", "Dummy: Jump", "Dummy: Guard", "Dummy: CPU"}

// next is the setting after d, the first one after the last.
func (d Dummy) next() Dummy { return (d + 1) % dummyCount }

// Decide writes the dummy's input for this tick into dst in place, so a tick
// allocates nothing: Stand and CPU hold nothing, Crouch holds down, Jump holds
// up, Guard holds back, away from the opponent. Guard only blocks once the
// engine guards (backlog item 023 of engine).
func (d Dummy) Decide(self, opponent match.Position, dst *input.TickInput) {
	clearInput(dst)
	switch d {
	case DummyCrouch:
		dst.Down = true
	case DummyJump:
		dst.Up = true
	case DummyGuard:
		if self.X < opponent.X {
			dst.Left = true
		} else {
			dst.Right = true
		}
	}
}
