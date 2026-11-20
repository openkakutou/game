package game

import (
	"os"
	"strings"
	"testing"

	"github.com/openkakutou/engine/v2/input"
	"github.com/openkakutou/engine/v2/match"
	"github.com/openkakutou/game/native/internal/mode"
)

// trainingFlow plays the built-in Training mode over the fixture content up
// to the match: both fighters picked, then the stage.
func trainingFlow(t *testing.T) *Flow {
	t.Helper()
	builtIns, err := mode.BuiltIn()
	if err != nil {
		t.Fatal(err)
	}
	fsys := os.DirFS(content)
	chars, _ := DiscoverCharacters(fsys, "chars")
	stages, _ := DiscoverStages(fsys, "stages")
	f := NewFlow(fsys, chars, stages, DefaultLoader(fsys), builtIns[1:2])
	inMode(t, f)
	f.Navigate(NavConfirm) // P1 picks fighter
	f.Navigate(NavConfirm) // P2 (the dummy) picks fighter
	f.Navigate(NavConfirm) // stage
	if f.Screen != ScreenMatch {
		t.Fatalf("screen = %v (%q), want the match right after the stage", f.Screen, f.Message)
	}
	return f
}

func pauseLabels(f *Flow) string {
	var labels []string
	for _, item := range f.PauseItems() {
		labels = append(labels, f.PauseLabel(item))
	}
	return strings.Join(labels, ",")
}

// chooseInPause opens the pause menu on the entry labelled label and confirms.
func chooseInPause(t *testing.T, f *Flow, item PauseItem) {
	t.Helper()
	if !f.Paused {
		f.Navigate(NavPause)
	}
	for i, it := range f.PauseItems() {
		if it == item {
			f.PauseCursor = i
			f.Navigate(NavConfirm)
			return
		}
	}
	t.Fatalf("pause menu has no %q entry: %s", item.Label(), pauseLabels(f))
}

func setHealth(m *Match, side int, health int) {
	m.State.Fighters[side].Health = health
	m.runtimes[side].Context.Health = health
}

func stepN(t *testing.T, m *Match, n int) {
	t.Helper()
	for range n {
		if err := m.Step([2]input.TickInput{}); err != nil {
			t.Fatal(err)
		}
	}
}

func trainingMatch(t *testing.T, training bool) *Match {
	t.Helper()
	a, b := loadPair(t)
	m, err := NewMatch(a, b, closeStage(), Setup{Rounds: 1, Training: training})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestPauseItems_VersusKeepsOnlyTheGamesEntries(t *testing.T) {
	f := pausedMatch(t)
	if got := pauseLabels(f); got != "Continue,Restart,Character Change,Move List,Options,Exit" {
		t.Errorf("versus items = %s", got)
	}
}

func TestPauseItems_TrainingAddsResetPositionsAndDummyAfterRestart(t *testing.T) {
	f := trainingFlow(t)
	if got := pauseLabels(f); got != "Continue,Restart,Reset Positions,Dummy: Stand,Character Change,Move List,Options,Exit" {
		t.Errorf("training items = %s", got)
	}
}

func TestPauseItems_TrainingMoveListIsReachableAndOpens(t *testing.T) {
	f := trainingFlow(t)
	chooseInPause(t, f, PauseMoveList)
	if !f.moveList.open {
		t.Error("Move List from the training pause menu did not open the move list")
	}
}

func TestDummy_ConfirmCyclesStandCrouchJumpGuardCPUThenStand(t *testing.T) {
	f := trainingFlow(t)
	want := []string{"Dummy: Crouch", "Dummy: Jump", "Dummy: Guard", "Dummy: CPU", "Dummy: Stand"}
	for _, label := range want {
		chooseInPause(t, f, PauseDummy)
		if !f.Paused {
			t.Fatal("choosing Dummy must keep the pause menu open")
		}
		if got := f.PauseLabel(PauseDummy); got != label {
			t.Fatalf("after confirm the entry reads %q, want %q", got, label)
		}
	}
}

func TestDummy_InputsPerSetting(t *testing.T) {
	left, right := match.Position{X: -50}, match.Position{X: 50}
	cases := []struct {
		name           string
		dummy          Dummy
		self, opponent match.Position
		up, down, l, r bool
	}{
		{"stand holds nothing", DummyStand, right, left, false, false, false, false},
		{"crouch holds down", DummyCrouch, right, left, false, true, false, false},
		{"jump holds up", DummyJump, right, left, true, false, false, false},
		{"guard on the right of the player holds right, away from them", DummyGuard, right, left, false, false, false, true},
		{"guard on the left of the player holds left, away from them", DummyGuard, left, right, false, false, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dst := input.TickInput{Up: true, Down: true, Left: true, Right: true, Buttons: map[string]bool{"a": true}}
			c.dummy.Decide(c.self, c.opponent, &dst)
			if dst.Up != c.up || dst.Down != c.down || dst.Left != c.l || dst.Right != c.r || dst.Buttons["a"] {
				t.Errorf("input = %+v, want up=%v down=%v left=%v right=%v and no button", dst, c.up, c.down, c.l, c.r)
			}
		})
	}
}

func TestDummy_DecideAllocatesNothing(t *testing.T) {
	dst := input.TickInput{Buttons: map[string]bool{"a": true}}
	allocs := testing.AllocsPerRun(100, func() {
		DummyGuard.Decide(match.Position{X: 50}, match.Position{X: -50}, &dst)
	})
	if allocs != 0 {
		t.Errorf("Decide allocates %v times per call, want 0", allocs)
	}
}

func TestFlowTick_TrainingDummyFollowsItsSetting(t *testing.T) {
	f := trainingFlow(t)
	f.Tick([2]input.TickInput{})
	if in := f.cpuIn[1]; in.Up || in.Down || in.Left || in.Right {
		t.Errorf("Stand: dummy input %+v, want none", in)
	}
	chooseInPause(t, f, PauseDummy) // Crouch
	f.Navigate(NavPause)
	f.Tick([2]input.TickInput{})
	if !f.cpuIn[1].Down {
		t.Errorf("Crouch: dummy input %+v, want down", f.cpuIn[1])
	}
	f.Dummy = DummyGuard
	f.Tick([2]input.TickInput{})
	if in := f.cpuIn[1]; !in.Right || in.Left || in.Down {
		t.Errorf("Guard with the dummy right of the player: %+v, want right only", in)
	}
	f.Match.State.Fighters[0].Position.X, f.Match.State.Fighters[1].Position.X = 50, -50
	f.Tick([2]input.TickInput{})
	if in := f.cpuIn[1]; !in.Left || in.Right {
		t.Errorf("Guard with the dummy left of the player: %+v, want left only", in)
	}
	f.Dummy = DummyCPU
	f.Match.State.Fighters[0].Position.X, f.Match.State.Fighters[1].Position.X = -150, 150
	f.Tick([2]input.TickInput{})
	if in := f.cpuIn[1]; !in.Left || in.Right {
		t.Errorf("CPU with the player far to the left: %+v, want the CPU walking left", in)
	}
}

func TestFlowTick_TrainingDummyNeverReadsPlayer2Controls(t *testing.T) {
	f := trainingFlow(t)
	f.Tick([2]input.TickInput{{}, {Up: true, Left: true}})
	if in := f.cpuIn[1]; in.Up || in.Left {
		t.Errorf("dummy input %+v, want the dummy setting (stand), not the controls", in)
	}
}

func TestTraining_FighterNotHitForOneSecondGetsItsLifeBack(t *testing.T) {
	m := trainingMatch(t, true)
	setHealth(m, 0, 600)
	stepN(t, m, 1) // the tick that sees the loss
	stepN(t, m, 59)
	if got := m.State.Fighters[0].Health; got != 600 {
		t.Fatalf("health after 59 quiet ticks = %d, want 600: refilled too early", got)
	}
	stepN(t, m, 1)
	if got := m.State.Fighters[0].Health; got != DefaultHealth {
		t.Fatalf("health after 60 quiet ticks = %d, want %d", got, DefaultHealth)
	}
	if got := m.runtimes[0].Context.Health; got != DefaultHealth {
		t.Errorf("runtime health = %d, want %d: the refill must reach the state machine's context", got, DefaultHealth)
	}
}

func TestTraining_AHitRestartsTheOneSecondWait(t *testing.T) {
	m := trainingMatch(t, true)
	setHealth(m, 1, 600)
	stepN(t, m, 31)
	setHealth(m, 1, 500) // hit again
	stepN(t, m, 60)      // the tick that sees it, then 59 quiet ones
	if got := m.State.Fighters[1].Health; got != 500 {
		t.Fatalf("health = %d, want 500: the second hit must restart the wait", got)
	}
	stepN(t, m, 1)
	if got := m.State.Fighters[1].Health; got != DefaultHealth {
		t.Errorf("health = %d, want %d after a full second without a hit", got, DefaultHealth)
	}
}

func TestTraining_HealthReachingZeroIsRefilledAndNoRoundEnds(t *testing.T) {
	cases := []struct {
		name string
		dead []int
	}{{"player", []int{0}}, {"dummy", []int{1}}, {"both", []int{0, 1}}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := trainingMatch(t, true)
			for _, side := range c.dead {
				setHealth(m, side, 0)
			}
			stepN(t, m, 1)
			if m.RoundEnded() || m.Over {
				t.Fatalf("round ended (%+v, over=%v), training never ends by KO", m.LastRound, m.Over)
			}
			for _, side := range c.dead {
				if got := m.State.Fighters[side].Health; got != DefaultHealth {
					t.Errorf("side %d health = %d, want %d at once", side, got, DefaultHealth)
				}
				if got := m.runtimes[side].Context.Health; got != DefaultHealth {
					t.Errorf("side %d runtime health = %d, want %d", side, got, DefaultHealth)
				}
			}
			if got := m.Progress; got.Wins != [2]int{} {
				t.Errorf("progress = %+v, want no round recorded", got)
			}
		})
	}
}

func TestTraining_OffKeepsTheLifeLostAndEndsTheRoundOnKO(t *testing.T) {
	m := trainingMatch(t, false)
	setHealth(m, 0, 600)
	stepN(t, m, 120)
	if got := m.State.Fighters[0].Health; got != 600 {
		t.Errorf("health = %d, want 600: only a training match refills", got)
	}
	setHealth(m, 1, 0)
	stepN(t, m, 1)
	if !m.RoundEnded() {
		t.Error("a KO must end the round outside training")
	}
}

func TestTraining_ResetPositionsRestoresStartPositionsLifeAndState(t *testing.T) {
	m := trainingMatch(t, true)
	start := [2]match.Position{m.State.Fighters[0].Position, m.State.Fighters[1].Position}
	m.State.Fighters[0].Position.X, m.State.Fighters[1].Position.X = 3, -4
	m.runtimes[0].Context.Position.X = 3
	setHealth(m, 0, 100)
	stepN(t, m, 5)
	if err := m.ResetPositions(); err != nil {
		t.Fatal(err)
	}
	for i := range start {
		if got := m.State.Fighters[i].Position; got != start[i] {
			t.Errorf("side %d position = %+v, want the start %+v", i, got, start[i])
		}
		if got := m.runtimes[i].Context.Position; got != start[i] {
			t.Errorf("side %d runtime position = %+v, want the start %+v", i, got, start[i])
		}
		if got := m.State.Fighters[i].Health; got != DefaultHealth {
			t.Errorf("side %d health = %d, want %d", i, got, DefaultHealth)
		}
	}
	if m.Ticks() != 0 {
		t.Errorf("ticks = %d, want 0", m.Ticks())
	}
	// the refill's wait starts over: a loss right after the reset still waits a second
	setHealth(m, 0, 600)
	stepN(t, m, 60)
	if got := m.State.Fighters[0].Health; got != 600 {
		t.Errorf("health = %d, want 600 (the wait restarts after a reset)", got)
	}
}

func TestPause_ResetPositionsKeepsTheDummySettingAndClosesTheMenu(t *testing.T) {
	f := trainingFlow(t)
	chooseInPause(t, f, PauseDummy) // Crouch
	chooseInPause(t, f, PauseDummy) // Jump
	f.Navigate(NavCancel)
	for range 10 {
		f.Tick([2]input.TickInput{})
	}
	f.Match.State.Fighters[0].Position.X = 1
	before := f.Match
	chooseInPause(t, f, PauseResetPositions)
	if f.Paused {
		t.Error("Reset Positions must close the pause menu")
	}
	if f.Dummy != DummyJump {
		t.Errorf("dummy = %v, want Jump kept", f.Dummy)
	}
	if f.Match != before || f.Match.Ticks() != 0 || f.Match.State.Fighters[0].Position.X >= 0 {
		t.Errorf("match reset = %v (same match %v), ticks %d, p1 x %v: want the same match back at its start", f.Match != before, f.Match == before, f.Match.Ticks(), f.Match.State.Fighters[0].Position.X)
	}
}

func TestTraining_HasNoTimerAndTheMatchKeepsRunning(t *testing.T) {
	f := trainingFlow(t)
	if got := f.Setup().TimeLimitSeconds; got != 0 {
		t.Errorf("time limit = %d, want 0", got)
	}
	if got := f.Match.State.RoundTimer; got != UnlimitedTimer {
		t.Errorf("round timer = %d at the start, want the unlimited %d", got, UnlimitedTimer)
	}
	for range 200 {
		f.Tick([2]input.TickInput{})
	}
	if f.Match.RoundEnded() || f.Screen != ScreenMatch {
		t.Fatalf("round ended or screen = %v: training never ends", f.Screen)
	}
}
