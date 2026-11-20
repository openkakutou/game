package game

import (
	"os"
	"testing"

	"github.com/openkakutou/engine/v2/input"
)

// statesVisited runs kfm_zss against an idle kfm for ticks with no input on
// either side, side 2 at level, and returns the states side 2 went through.
func statesVisited(t *testing.T, level float64, ticks int) map[int]bool {
	t.Helper()
	p1, err := LoadFighter(os.DirFS(reference), "chars/kfm/kfm.def", "common")
	if err != nil {
		t.Fatal(err)
	}
	p2, err := LoadFighter(os.DirFS(reference), "chars/kfm_zss/kfm_zss.def", "common")
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewMatch(p1, p2, closeStage(), Setup{Rounds: 1, AILevel: [2]float64{0, level}})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int]bool{}
	for range ticks {
		if err := m.Step([2]input.TickInput{}); err != nil {
			t.Fatal(err)
		}
		seen[m.runtimes[1].Context.StateNo] = true
		if m.RoundEnded() {
			break
		}
	}
	return seen
}

// kfm_zss's AI.zss acts on its own at a level (engine v2.27.0 runs it), and
// not without one.
func TestAILevel_KungFuMansOwnAIPlaysOnlyWithALevel(t *testing.T) {
	human := statesVisited(t, 0, 600)
	cpu := statesVisited(t, 8, 600)
	if len(cpu) <= len(human) {
		t.Fatalf("states with AI level 8 = %v, without = %v: kfm_zss's AI.zss must act on its own", cpu, human)
	}
}
