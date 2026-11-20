package game

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/openkakutou/engine/v2/input"
	"github.com/openkakutou/engine/v2/match"
	"github.com/openkakutou/game/native/internal/mode"
)

// modeSrc builds a mode file with the given flow steps (JSON) and rules.
func modeSrc(flow, p1, p2 string, rounds, timeLimit int, onWin, onLose string) string {
	return fmt.Sprintf(`{"formatVersion": 1, "id": "t", "name": "T", "flow": [%s],
	"rules": {"sides": [{"control": %q, "characters": 1}, {"control": %q, "characters": 1}],
	"rounds": %d, "timeLimitSeconds": %d, "opponents": {"kind": "chosen"}, "onWin": %q, "onLose": %q}}`,
		flow, p1, p2, rounds, timeLimit, onWin, onLose)
}

const (
	stepSelect = `{"screen": "characterSelect", "params": {"sides": ["p1", "p2"]}}`
	stepStage  = `{"screen": "stageSelect"}`
	stepSetup  = `{"screen": "matchSetup"}`
	stepVS     = `{"screen": "versus"}`
	stepMatch  = `{"screen": "match"}`
	stepResult = `{"screen": "results"}`
)

func flowFromSrc(t *testing.T, src string) *Flow {
	t.Helper()
	m, err := mode.Parse("t.jsonc", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	fsys := os.DirFS(content)
	chars, _ := DiscoverCharacters(fsys, "chars")
	stages, _ := DiscoverStages(fsys, "stages")
	return inMode(t, NewFlow(fsys, chars, stages, DefaultLoader(fsys), []mode.Mode{m}))
}

func steps(s ...string) string { return strings.Join(s, ",") }

// finishMatch swaps in a close-quarters stage and ticks until the match
// leaves the match screen, side p attacking and the other side idle.
func finishMatch(t *testing.T, f *Flow, p match.Side) {
	t.Helper()
	f.Match, _ = NewMatch(f.Fighters[0], f.Fighters[1], closeStage(), f.Setup())
	for i := 0; i < 20000 && f.Screen == ScreenMatch; i++ {
		var in [2]input.TickInput
		in[p] = tapA(i)
		f.Tick(in)
	}
	if f.Screen == ScreenMatch {
		t.Fatal("the match never ended")
	}
}

func TestVersusMode_WalksSelectStageSetupVersusMatchResults(t *testing.T) {
	f := newTestFlow(t)
	f.Navigate(NavConfirm) // P1 picks fighter
	f.Navigate(NavRight)
	f.Navigate(NavConfirm) // P2 picks rival
	if f.Screen != ScreenStage || f.Picks != [2]int{0, 1} {
		t.Fatalf("screen=%v picks=%v, want the stage screen with picks 0,1", f.Screen, f.Picks)
	}
	f.Navigate(NavConfirm)
	if f.Screen != ScreenSetup {
		t.Fatalf("screen=%v, want setup", f.Screen)
	}
	if s := f.Setup(); s.Rounds != 3 || s.TimeLimitSeconds != 99 || s.CPU != [2]bool{false, false} {
		t.Fatalf("setup preselects %+v, want the rules: 3 rounds, 99 seconds, no CPU", s)
	}
	f.Navigate(NavConfirm) // Start
	if f.Screen != ScreenVersus || f.Match != nil {
		t.Fatalf("screen=%v match=%v, want the VS screen before the match", f.Screen, f.Match)
	}
	f.Navigate(NavConfirm)
	if f.Screen != ScreenMatch || f.Match == nil {
		t.Fatalf("screen=%v msg=%q, want the match", f.Screen, f.Message)
	}
	finishMatch(t, f, match.SideP1)
	if f.Screen != ScreenResult || f.ResultWinner != match.SideP1 {
		t.Fatalf("screen=%v winner=%v, want results won by P1", f.Screen, f.ResultWinner)
	}
	f.Navigate(NavConfirm) // rematch: same characters and stage, straight to the match
	if f.Screen != ScreenMatch || f.Match.Progress.RoundsPlayed != 0 {
		t.Fatalf("rematch: screen=%v", f.Screen)
	}
	if f.lastChars[0].Name != "fighter" || f.lastChars[1].Name != "rival" {
		t.Errorf("rematch characters = %v", f.lastChars)
	}
	finishMatch(t, f, match.SideP1)
	f.Navigate(NavCancel) // back to the first character select, P1 picking
	if f.Screen != ScreenSelect || f.Picking != 0 || f.Cursor != 0 {
		t.Errorf("screen=%v picking=%d cursor=%d, want P1 picking", f.Screen, f.Picking, f.Cursor)
	}
}

func TestVersusScreen_AdvancesAfter120TicksAndOnConfirm(t *testing.T) {
	f := toVersus(t, newTestFlow(t))
	for range 119 {
		f.Tick([2]input.TickInput{})
	}
	if f.Screen != ScreenVersus {
		t.Fatalf("screen=%v after 119 ticks, want still the VS screen", f.Screen)
	}
	f.Tick([2]input.TickInput{})
	if f.Screen != ScreenMatch {
		t.Errorf("screen=%v after 120 ticks, want the match", f.Screen)
	}
	g := toVersus(t, newTestFlow(t))
	g.Navigate(NavConfirm)
	if g.Screen != ScreenMatch {
		t.Errorf("confirm on the VS screen: screen=%v", g.Screen)
	}
}

func TestVersusScreen_CancelGoesBackToMatchSetup(t *testing.T) {
	f := toVersus(t, newTestFlow(t))
	f.Navigate(NavCancel)
	if f.Screen != ScreenSetup || f.SetupRow != SetupRowStart {
		t.Errorf("screen=%v row=%d, want the setup screen on Start", f.Screen, f.SetupRow)
	}
}

// toVersus confirms through the first screens of the Versus flow.
func toVersus(t *testing.T, f *Flow) *Flow {
	t.Helper()
	for range 4 {
		f.Navigate(NavConfirm)
	}
	if f.Screen != ScreenVersus {
		t.Fatalf("screen=%v, want the VS screen", f.Screen)
	}
	return f
}

func TestCancelOnStageSelect_ReturnsToTheLastPickerOfTheSelect(t *testing.T) {
	f := newTestFlow(t)
	f.Navigate(NavConfirm)
	f.Navigate(NavRight)
	f.Navigate(NavConfirm)
	f.Navigate(NavCancel)
	if f.Screen != ScreenSelect || f.Picking != 1 || f.Cursor != 1 {
		t.Errorf("screen=%v picking=%d cursor=%d, want P2 picking on its pick", f.Screen, f.Picking, f.Cursor)
	}
}

func TestCharacterSelect_OnlyP1SidePicksAndP2KeepsFirstEntry(t *testing.T) {
	f := flowFromSrc(t, modeSrc(steps(`{"screen": "characterSelect", "params": {"sides": ["p1"]}}`, stepStage, stepMatch), "human", "cpu", 3, 99, "results", "results"))
	f.Navigate(NavRight)
	f.Navigate(NavConfirm)
	if f.Screen != ScreenStage || f.Picks != [2]int{1, 0} {
		t.Fatalf("screen=%v picks=%v, want the stage screen, P1 on 1 and P2 left on the first entry", f.Screen, f.Picks)
	}
	f.Navigate(NavCancel)
	if f.Screen != ScreenSelect || f.Picking != 0 || f.Cursor != 1 {
		t.Errorf("screen=%v picking=%d cursor=%d, want P1 picking", f.Screen, f.Picking, f.Cursor)
	}
	f.Navigate(NavCancel)
	if f.Screen != ScreenMainMenu {
		t.Error("cancel on the first step should return to the main menu")
	}
}

func TestCharacterSelect_OnlyP2SidePicksIntoItsOwnSlot(t *testing.T) {
	f := flowFromSrc(t, modeSrc(steps(`{"screen": "characterSelect", "params": {"sides": ["p2"]}}`, stepStage, stepMatch), "human", "human", 3, 99, "results", "results"))
	if f.Picking != 1 {
		t.Fatalf("picking=%d, want player 2", f.Picking)
	}
	f.Navigate(NavRight)
	f.Navigate(NavConfirm)
	if f.Screen != ScreenStage || f.Picks != [2]int{0, 1} {
		t.Errorf("screen=%v picks=%v, want P2 on 1 and P1 on the first entry", f.Screen, f.Picks)
	}
}

func TestFlowWithoutMatchSetup_UsesTheRules(t *testing.T) {
	f := flowFromSrc(t, modeSrc(steps(stepSelect, stepStage, stepMatch), "human", "cpu", 5, 45, "results", "results"))
	f.Navigate(NavConfirm)
	f.Navigate(NavConfirm)
	f.Navigate(NavConfirm) // stage
	if f.Screen != ScreenMatch {
		t.Fatalf("screen=%v msg=%q, want the match straight after the stage", f.Screen, f.Message)
	}
	if s := f.Match.Setup; s.Rounds != 5 || s.TimeLimitSeconds != 45 || s.CPU != [2]bool{false, true} {
		t.Errorf("setup = %+v, want 5 rounds, 45 seconds, P2 CPU", s)
	}
	if f.cpus[0] != nil || f.cpus[1] == nil {
		t.Errorf("cpus = %v, want only P2 driven", f.cpus)
	}
}

func TestFlowWithoutMatchSetup_HumanOrCpuSideStaysHuman(t *testing.T) {
	f := flowFromSrc(t, modeSrc(steps(stepSelect, stepStage, stepMatch), "human", "humanOrCpu", 3, 99, "results", "results"))
	for range 3 {
		f.Navigate(NavConfirm)
	}
	if f.Screen != ScreenMatch || f.cpus[1] != nil {
		t.Errorf("screen=%v cpu=%v, want a human P2", f.Screen, f.cpus[1])
	}
}

func TestMatchSetup_Player2RowOnlyWhenControlIsHumanOrCpu(t *testing.T) {
	cases := []struct {
		control string
		up      int // row reached by Up from Start
		down    int // row reached by Down from Start
	}{
		{"humanOrCpu", SetupRowPlayer2, SetupRowRounds},
		{"human", SetupRowTime, SetupRowRounds},
		{"cpu", SetupRowTime, SetupRowRounds},
	}
	for _, c := range cases {
		f := flowFromSrc(t, modeSrc(steps(stepSelect, stepStage, stepSetup, stepMatch), "human", c.control, 3, 99, "results", "results"))
		for range 3 {
			f.Navigate(NavConfirm)
		}
		f.Navigate(NavUp)
		if f.SetupRow != c.up {
			t.Errorf("%s: Up from Start reached row %d, want %d", c.control, f.SetupRow, c.up)
		}
		f.SetupRow = SetupRowStart
		f.Navigate(NavDown)
		if f.SetupRow != c.down {
			t.Errorf("%s: Down from Start reached row %d, want %d", c.control, f.SetupRow, c.down)
		}
	}
}

func TestMatchSetup_ControlCpuSideIsCPUDrivenWithoutARowAndHumanNever(t *testing.T) {
	f := flowFromSrc(t, modeSrc(steps(stepSelect, stepStage, stepSetup, stepMatch), "human", "cpu", 3, 99, "results", "results"))
	for range 3 {
		f.Navigate(NavConfirm)
	}
	f.Navigate(NavConfirm) // Start
	if f.Screen != ScreenMatch || f.cpus[0] != nil || f.cpus[1] == nil {
		t.Fatalf("screen=%v cpus=%v, want P2 alone driven by the CPU", f.Screen, f.cpus)
	}
	h := flowFromSrc(t, modeSrc(steps(stepSelect, stepStage, stepSetup, stepMatch), "human", "human", 3, 99, "results", "results"))
	for range 4 {
		h.Navigate(NavConfirm)
	}
	if h.cpus[1] != nil || h.Setup().CPU != [2]bool{false, false} {
		t.Errorf("a human side must never be CPU-driven: %+v", h.Setup())
	}
}

func TestMatchSetup_RulesTimeMissingFromOptionsIsOfferedSortedAndPreselected(t *testing.T) {
	f := flowFromSrc(t, modeSrc(steps(stepSelect, stepStage, stepSetup, stepMatch), "human", "human", 3, 45, "results", "results"))
	f.Settings.Gameplay = Gameplay{Rounds: 3, TimeLimitSeconds: 45} // the setup preselects the gameplay settings
	f.enterMode(0)
	for range 3 {
		f.Navigate(NavConfirm)
	}
	if got := f.Setup().TimeLimitSeconds; got != 45 {
		t.Fatalf("preselected time = %d, want 45", got)
	}
	f.SetupRow = SetupRowTime
	f.Navigate(NavLeft)
	if got := f.Setup().TimeLimitSeconds; got != 30 {
		t.Errorf("Left from 45 = %d, want 30", got)
	}
	f.Navigate(NavRight)
	f.Navigate(NavRight)
	if got := f.Setup().TimeLimitSeconds; got != 60 {
		t.Errorf("Right from 45 = %d, want 60", got)
	}
}

func TestMatchEnd_ResultsOutcomeWithoutAResultsStepIsAnError(t *testing.T) {
	f := flowFromSrc(t, modeSrc(steps(stepSelect, stepStage, stepMatch), "human", "human", 1, 99, "results", "results"))
	for range 3 {
		f.Navigate(NavConfirm)
	}
	finishMatch(t, f, match.SideP1)
	if f.Screen != ScreenError || !strings.Contains(f.Message, "results") {
		t.Errorf("screen=%v message=%q", f.Screen, f.Message)
	}
}

func TestErrorInsideAMode_ConfirmReturnsToTheMainMenuInsteadOfQuitting(t *testing.T) {
	f := flowFromSrc(t, modeSrc(steps(stepSelect, stepStage, stepMatch), "human", "human", 1, 99, "results", "results"))
	for range 3 {
		f.Navigate(NavConfirm)
	}
	finishMatch(t, f, match.SideP1)
	if !f.ErrorReturnsToMenu() {
		t.Fatal("an error inside a mode should offer the main menu")
	}
	f.Navigate(NavConfirm)
	if f.Quit || f.Screen != ScreenMainMenu || f.Match != nil {
		t.Errorf("quit=%v screen=%v match=%v, want the main menu without a match", f.Quit, f.Screen, f.Match)
	}
}

func TestErrorBeforeAnyMode_StillQuits(t *testing.T) {
	f := NewFlow(nil, nil, nil, Loader{}, nil)
	if f.Screen != ScreenError || f.ErrorReturnsToMenu() {
		t.Fatalf("screen=%v returnsToMenu=%v, want a quitting error screen", f.Screen, f.ErrorReturnsToMenu())
	}
	f.Navigate(NavCancel)
	if !f.Quit {
		t.Error("cancel on a startup error should quit")
	}
}

func TestWatchMode_BothSidesCPUDrivenWithoutAPlayer2RowAndPreselectedFromTheRules(t *testing.T) {
	builtIns, err := mode.BuiltIn()
	if err != nil {
		t.Fatal(err)
	}
	fsys := os.DirFS(content)
	chars, _ := DiscoverCharacters(fsys, "chars")
	stages, _ := DiscoverStages(fsys, "stages")
	f := inMode(t, NewFlow(fsys, chars, stages, DefaultLoader(fsys), builtIns[2:]))
	for range 3 {
		f.Navigate(NavConfirm) // P1 pick, P2 pick, stage
	}
	if f.Screen != ScreenSetup || f.HasPlayer2Row() {
		t.Fatalf("screen=%v player2Row=%v, want setup without a Player 2 row", f.Screen, f.HasPlayer2Row())
	}
	if s := f.Setup(); s.Rounds != 3 || s.TimeLimitSeconds != 99 || s.CPU != [2]bool{true, true} {
		t.Fatalf("setup preselects %+v, want 3 rounds, 99 seconds, both CPU", s)
	}
	f.Navigate(NavUp)
	if f.SetupRow != SetupRowTime {
		t.Errorf("Up from Start reached row %d, want the time row", f.SetupRow)
	}
	f.SetupRow = SetupRowStart
	f.Navigate(NavConfirm) // Start
	f.Navigate(NavConfirm) // skip the VS screen
	if f.Screen != ScreenMatch || f.cpus[0] == nil || f.cpus[1] == nil {
		t.Fatalf("screen=%v cpus=%v, want both fighters CPU-driven", f.Screen, f.cpus)
	}
}
