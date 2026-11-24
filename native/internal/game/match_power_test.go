package game

import (
	"strconv"
	"strings"
	"testing"

	"github.com/openkakutou/engine/v2/evaluator"
)

func withPower(t *testing.T, f *Fighter, power int) *Fighter {
	t.Helper()
	c, err := evaluator.ParseConstants(strings.NewReader("[Data]\npower = " + strconv.Itoa(power) + "\n"))
	if err != nil {
		t.Fatal(err)
	}
	g := *f
	g.Program.Constants = c
	return &g
}

// char.go:4148 powerMax is the character's [Data] power (3000 unset); the
// power bars fill against it.
func TestMatch_EachFighterHasItsOwnPowerMax(t *testing.T) {
	a, b := loadPair(t)
	m, err := NewMatch(withPower(t, a, 5000), b, closeStage(), Setup{Rounds: 1})
	if err != nil {
		t.Fatal(err)
	}
	if m.PowerMax != [2]int{5000, 3000} {
		t.Errorf("PowerMax = %v, want [5000 3000]", m.PowerMax)
	}
}

// system.go:4426-4432: -maxpowermode starts each fighter at its powerMax,
// and -p<n>.power is clamped to it.
func TestMatch_StartPowerIsCappedAtTheFightersPowerMax(t *testing.T) {
	a, b := loadPair(t)
	m, err := NewMatch(withPower(t, a, 5000), withPower(t, b, 1000), closeStage(), Setup{Rounds: 1, MaxPowerMode: true})
	if err != nil {
		t.Fatal(err)
	}
	if m.State.Fighters[0].Power != 5000 || m.State.Fighters[1].Power != 1000 {
		t.Errorf("max power mode: power = %d, %d; want 5000, 1000", m.State.Fighters[0].Power, m.State.Fighters[1].Power)
	}
	m, err = NewMatch(withPower(t, a, 5000), withPower(t, b, 1000), closeStage(), Setup{Rounds: 1, Start: [2]StartValues{{Power: 4000}, {Power: 4000}}})
	if err != nil {
		t.Fatal(err)
	}
	if m.State.Fighters[0].Power != 4000 || m.State.Fighters[1].Power != 1000 {
		t.Errorf("-p<n>.power 4000: power = %d, %d; want 4000, 1000 (clamped)", m.State.Fighters[0].Power, m.State.Fighters[1].Power)
	}
}

// -p<n>.power takes any power: the character's powerMax caps it later.
func TestQuickVersusFrom_StartPowerAboveTheDefaultCapIsAccepted(t *testing.T) {
	q, _, _, err := QuickVersusFrom(ParseArgs([]string{"kfm", "kfm", "-p1.power", "4000"}))
	if err != nil || q.Start[0].Power != 4000 {
		t.Errorf("-p1.power 4000: power %d, err %v; want 4000, none", q.Start[0].Power, err)
	}
}
