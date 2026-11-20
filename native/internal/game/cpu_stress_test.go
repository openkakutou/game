package game

import (
	"os"
	"testing"

	"github.com/openkakutou/engine/v2/input"
)

// Two CPUs at level 8, jamming buttons with fixed seeds and playing their own
// AI, fight a round on the reference characters (kfm, kfm_zss, both common1)
// without the engine ever stopping the match.
func TestStress_CPUVersusCPUOnTheReferenceCharactersNeverStops(t *testing.T) {
	errs := map[string]int{}
	for seed := uint64(1); seed <= 6; seed++ {
		for _, defs := range [][2]string{{"chars/kfm_zss/kfm_zss.def", "chars/kfm_zss/kfm_zss.def"}, {"chars/kfm/kfm.def", "chars/kfm_zss/kfm_zss.def"}} {
			a, _ := LoadFighter(os.DirFS(reference), defs[0], "common")
			b, _ := LoadFighter(os.DirFS(reference), defs[1], "common")
			m, err := NewMatch(a, b, wideStage(), Setup{Rounds: 1, AILevel: [2]float64{8, 8}})
			if err != nil {
				t.Fatal(err)
			}
			cpus := [2]*CPU{NewCPU(8, SeededDraw(seed)), NewCPU(8, SeededDraw(seed+100))}
			var in [2]input.TickInput
			for tick := 0; tick < 3600 && !m.RoundEnded(); tick++ {
				cpus[0].Decide(&in[0])
				cpus[1].Decide(&in[1])
				if err := m.Step(in); err != nil {
					errs[err.Error()]++
					break
				}
			}
		}
	}
	for e, n := range errs {
		t.Errorf("%d× %s", n, e)
	}
}
