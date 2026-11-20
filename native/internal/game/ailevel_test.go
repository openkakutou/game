package game

import (
	"testing"

	"github.com/openkakutou/engine/v2/input"
)

// cpuMatchFlow starts a Versus match with player 2 on the CPU.
func cpuMatchFlow(t *testing.T, difficulty int) *Flow {
	t.Helper()
	f := newTestFlow(t)
	f.Settings.Gameplay.Difficulty = difficulty
	f.Player2CPU = true
	f.Navigate(NavConfirm) // P1
	f.Navigate(NavConfirm) // P2
	f.Navigate(NavConfirm) // stage
	f.Navigate(NavConfirm) // setup: start
	f.Navigate(NavConfirm) // VS
	if f.Match == nil {
		t.Fatalf("screen = %v, want a match", f.Screen)
	}
	return f
}

func TestAILevel_TheCPUSidesStateMachineReadsTheDifficulty(t *testing.T) {
	f := cpuMatchFlow(t, 6)
	if got := f.Match.runtimes[1].Context.AILevel; got != 6 {
		t.Errorf("CPU side AILevel = %v, want 6", got)
	}
	if got := f.Match.runtimes[0].Context.AILevel; got != 0 {
		t.Errorf("human side AILevel = %v, want 0", got)
	}
	if f.cpus[1] == nil || f.cpus[1].Level != 6 {
		t.Errorf("CPU = %+v, want level 6", f.cpus[1])
	}
	f.Tick([2]input.TickInput{})
	if got := f.Match.runtimes[1].Context.AILevel; got != 6 {
		t.Errorf("after a tick AILevel = %v, want 6 kept", got)
	}
	if err := f.Match.NextRound(); err != nil {
		t.Fatal(err)
	}
	if got := f.Match.runtimes[1].Context.AILevel; got != 6 {
		t.Errorf("next round AILevel = %v, want 6", got)
	}
}

func TestAILevel_RunLevelWinsOverTheSetting(t *testing.T) {
	f := newTestFlow(t)
	f.Settings.Gameplay.Difficulty = 2
	f.RunAILevel = 8
	if f.AILevel() != 8 {
		t.Errorf("AILevel = %v, want -ailevel's 8", f.AILevel())
	}
}

func TestOptions_GameplayDifficultyCyclesOneToEight(t *testing.T) {
	f := newTitleFlow(t)
	f.Settings.Gameplay.Difficulty = 8
	f.options.concern, f.options.cursor = ConcernGameplay, gameplayRowDifficulty
	f.changeRow(1)
	if f.Settings.Gameplay.Difficulty != 1 {
		t.Errorf("8 then right = %d, want 1", f.Settings.Gameplay.Difficulty)
	}
	f.changeRow(-1)
	if f.Settings.Gameplay.Difficulty != 8 {
		t.Errorf("1 then left = %d, want 8", f.Settings.Gameplay.Difficulty)
	}
}

func TestQuickVersus_AILevelOfEachSide(t *testing.T) {
	q, ok, notices, err := QuickVersusFrom(ParseArgs([]string{"kfm", "kfm", "-p2.ai", "7", "-p1.ai", "0"}))
	if !ok || err != nil {
		t.Fatal(ok, err)
	}
	if s := q.Setup(); s.AILevel != [2]float64{0, 7} || s.CPU != [2]bool{false, true} {
		t.Errorf("setup = %+v, want player 2 on the CPU at level 7", s)
	}
	if len(notices) != 0 {
		t.Errorf("notices = %v, want none: levels are used now", notices)
	}
}

func TestRunOptions_AILevelIsClampedOneToEight(t *testing.T) {
	for raw, want := range map[string]int{"0": 1, "5": 5, "12": 8} {
		o, err := RunOptionsFrom(ParseArgs([]string{"-ailevel", raw}))
		if err != nil || o.AILevel != want {
			t.Errorf("-ailevel %s = %d (%v), want %d", raw, o.AILevel, err, want)
		}
	}
	if o, _ := RunOptionsFrom(ParseArgs(nil)); o.AILevel != 0 {
		t.Errorf("no -ailevel = %d, want 0", o.AILevel)
	}
}
