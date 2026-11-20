package game

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/openkakutou/engine/v2/input"
	"github.com/openkakutou/engine/v2/match"
	"github.com/openkakutou/game/native/internal/mode"
	"github.com/openkakutou/stage"
)

const (
	stepSelectP1 = `{"screen": "characterSelect", "params": {"sides": ["p1"]}}`
	stepContinue = `{"screen": "continue"}`
	stepVictory  = `{"screen": "victory"}`
)

// arcadeChars and arcadeStages are the installed content of the arcade tests,
// in the default (folder name) order; the fighters all load the fixture's.
var (
	arcadeChars  = []Entry{{Name: "a", DefPath: "a"}, {Name: "b", DefPath: "b"}, {Name: "c", DefPath: "c"}, {Name: "d", DefPath: "d"}, {Name: "e", DefPath: "e"}, {Name: "boss", DefPath: "boss"}}
	arcadeStages = []Entry{{Name: "s0", DefPath: "s0"}, {Name: "s1", DefPath: "s1"}, {Name: "s2", DefPath: "s2"}}
)

// ladderSrc is an Arcade-like mode: a ladder with the given opponents rule
// body (after "kind": "ladder"), the roster rule ("" for none) and outcomes,
// over the given flow steps.
func ladderSrc(opponents, roster, onWin, onLose string, flow ...string) string {
	if roster != "" {
		roster = `"roster": ` + roster + `,`
	}
	return fmt.Sprintf(`{"formatVersion": 1, "id": "arcade", "name": "Arcade", "flow": [%s],
	"rules": {"sides": [{"control": "human", "characters": 1}, {"control": "cpu", "characters": 1}],
	%s "rounds": 1, "timeLimitSeconds": 0, "opponents": {"kind": "ladder", %s}, "onWin": %q, "onLose": %q}}`,
		strings.Join(flow, ","), roster, opponents, onWin, onLose)
}

// pool2 leaves b and c as the opponents of a player who picked a.
const pool2 = `{"characters": {"opponents": {"include": ["b", "c"]}}}`

var arcadeFlowSteps = []string{stepSelectP1, stepVS, stepMatch, stepContinue, stepVictory}

// scripted is a random source answering the given values in order; asking
// for more, or for a value its bound does not allow, fails the test.
func scripted(t *testing.T, values ...int) func(n int) int {
	t.Helper()
	next := 0
	return func(n int) int {
		if next >= len(values) {
			t.Fatalf("Rand(%d): asked %d times, only %d values scripted", n, next+1, len(values))
		}
		v := values[next]
		next++
		if v < 0 || v >= n {
			t.Fatalf("Rand(%d) call %d: scripted value %d is out of range", n, next, v)
		}
		return v
	}
}

// arcadeFlow is a flow inside the mode src over arcadeChars and arcadeStages,
// whose loader gives every character the fixture's fighter and records the
// .def names it is asked for.
func arcadeFlow(t *testing.T, src string, rand func(int) int) (*Flow, *[]string) {
	t.Helper()
	m, err := mode.Parse("arcade.jsonc", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	var loaded []string
	l := Loader{
		Fighter: func(def string) (*Fighter, error) {
			loaded = append(loaded, def)
			return LoadFighter(os.DirFS(content), "chars/fighter/fighter.def")
		},
		Stage: func(def string) (*stage.Stage, error) { return closeStage(), nil },
	}
	f := NewFlow(nil, slices.Clone(arcadeChars), slices.Clone(arcadeStages), l, []mode.Mode{m})
	f.Rand = rand
	return inMode(t, f), &loaded
}

// pick moves the character select cursor right steps cells and confirms.
func pick(f *Flow, steps int) {
	for range steps {
		f.Navigate(NavRight)
	}
	f.Navigate(NavConfirm)
}

func TestArcade_Ladder_IsDrawnAfterThePickFromTheOpponentPool(t *testing.T) {
	cases := []struct {
		name              string
		opponents, roster string
		pickSteps         int
		rand              []int
		wantLadder        []int
		wantStage         int
	}{
		{
			// Candidates a c d e (the player is b, the boss is the final): Fisher-Yates
			// with draws 1, 2, 1 turns [0 2 3 4] into [0 4 3 2]; the first 3, then the boss.
			name:      "no repeats, the player's character left out, the final last",
			opponents: `"count": 3, "final": "boss"`, pickSteps: 1, rand: []int{1, 2, 1, 2},
			wantLadder: []int{0, 4, 3, 5}, wantStage: 2,
		},
		{
			// Draws of 0 move the last cell to the front each time: [0 2 3 4] becomes [2 3 4 0].
			name:      "a draw of zero always rotates the candidates",
			opponents: `"count": 3, "final": "boss"`, pickSteps: 1, rand: []int{0, 0, 0, 0},
			wantLadder: []int{2, 3, 4, 5}, wantStage: 0,
		},
		{
			name:      "the player's own character is fought when nothing else is left",
			opponents: `"count": 3`, roster: `{"characters": {"opponents": {"include": ["a"]}}}`, pickSteps: 0, rand: []int{1},
			wantLadder: []int{0}, wantStage: 1,
		},
		{
			name:      "fewer candidates than the count shorten the ladder",
			opponents: `"count": 6`, roster: `{"characters": {"opponents": {"include": ["a", "b", "c"]}}}`, pickSteps: 0, rand: []int{0, 2},
			wantLadder: []int{2, 1}, wantStage: 2,
		},
		{
			name:      "a final that is not installed is skipped",
			opponents: `"count": 2, "final": "ghost"`, roster: `{"characters": {"opponents": {"include": ["a", "b", "c"]}}}`, pickSteps: 0, rand: []int{1, 0},
			wantLadder: []int{1, 2}, wantStage: 0,
		},
		{
			name:      "the final is fought even when the opponent pool leaves it out",
			opponents: `"count": 3, "final": "boss"`, roster: `{"characters": {"opponents": {"include": ["a", "b"]}}}`, pickSteps: 0, rand: []int{1},
			wantLadder: []int{1, 5}, wantStage: 1,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, _ := arcadeFlow(t, ladderSrc(c.opponents, c.roster, "nextOpponent", "continue", arcadeFlowSteps...), scripted(t, c.rand...))
			pick(f, c.pickSteps)
			if f.Screen != ScreenVersus {
				t.Fatalf("screen = %v (%q), want the VS screen", f.Screen, f.Message)
			}
			if !slices.Equal(f.ladder, c.wantLadder) {
				t.Errorf("ladder = %v, want %v", f.ladder, c.wantLadder)
			}
			if f.Picks[1] != c.wantLadder[0] || f.StageNo != c.wantStage {
				t.Errorf("first fight: opponent %d on stage %d, want %d on %d", f.Picks[1], f.StageNo, c.wantLadder[0], c.wantStage)
			}
		})
	}
}

func TestArcade_ALadderWithNobodyToFight_ShowsAnErrorNamingTheMode(t *testing.T) {
	src := ladderSrc(`"count": 3`, `{"characters": {"opponents": {"include": []}}}`, "nextOpponent", "continue", arcadeFlowSteps...)
	f, _ := arcadeFlow(t, src, scripted(t))
	pick(f, 0)
	if f.Screen != ScreenError || !strings.Contains(f.Message, "Arcade") || !strings.Contains(f.Message, "opponent") {
		t.Errorf("screen = %v, message = %q, want an error naming the mode and the missing opponents", f.Screen, f.Message)
	}
}

func TestArcade_CancellingTheVersusScreenAndPickingAgainDrawsAFreshLadder(t *testing.T) {
	// Candidates b c: a draw of 0 gives [c], a draw of 1 gives [b].
	f, _ := arcadeFlow(t, ladderSrc(`"count": 1`, pool2, "nextOpponent", "continue", arcadeFlowSteps...), scripted(t, 0, 0, 1, 0))
	pick(f, 0)
	if !slices.Equal(f.ladder, []int{2}) {
		t.Fatalf("ladder = %v, want [2]", f.ladder)
	}
	f.Navigate(NavCancel)
	if f.Screen != ScreenSelect {
		t.Fatalf("screen = %v, want the character select", f.Screen)
	}
	pick(f, 0)
	if !slices.Equal(f.ladder, []int{1}) {
		t.Errorf("ladder = %v, want a fresh draw: [1]", f.ladder)
	}
}

// versusOpponentOnStage is the opponent and stage roster indexes the VS screen
// shows.
func versusOpponentOnStage(t *testing.T, f *Flow) (opponent, stageNo int) {
	t.Helper()
	if f.Screen != ScreenVersus {
		t.Fatalf("screen = %v (%q), want the VS screen", f.Screen, f.Message)
	}
	return f.Picks[1], f.StageNo
}

func TestArcade_WinningEveryFight_RunsTheLadderThenVictoryThenTheMainMenu(t *testing.T) {
	// Two opponents, b and c, against a; the stages visible are s1 and s2.
	roster := `{"characters": {"opponents": {"include": ["a", "b", "c"]}}, "stages": {"select": {"exclude": ["s0"]}}}`
	src := ladderSrc(`"count": 2`, roster, "nextOpponent", "continue", arcadeFlowSteps...)
	// Candidates b c; Rand(2)=0 makes [c b]; stage Rand(2)=1 picks s2; next fight stage Rand(2)=0 picks s1.
	f, loaded := arcadeFlow(t, src, scripted(t, 0, 1, 0))
	f.VictoryTicks = 300
	pick(f, 0)
	if opponent, stageNo := versusOpponentOnStage(t, f); opponent != 2 || stageNo != 2 {
		t.Fatalf("first fight: opponent %d on stage %d, want c (2) on s2 (2)", opponent, stageNo)
	}
	f.Navigate(NavConfirm)
	finishMatch(t, f, match.SideP1)
	if opponent, stageNo := versusOpponentOnStage(t, f); opponent != 1 || stageNo != 1 {
		t.Fatalf("second fight: opponent %d on stage %d, want b (1) on s1 (1)", opponent, stageNo)
	}
	f.Navigate(NavConfirm)
	if want := []string{"a", "c", "a", "b"}; !slices.Equal(*loaded, want) {
		t.Errorf("loaded %v, want the player then each opponent: %v", *loaded, want)
	}
	finishMatch(t, f, match.SideP1)
	if f.Screen != ScreenVictory {
		t.Fatalf("screen = %v (%q), want the victory screen after the last opponent", f.Screen, f.Message)
	}
	for range 299 {
		f.Tick([2]input.TickInput{})
	}
	if f.Screen != ScreenVictory {
		t.Fatalf("screen = %v after 299 ticks, want still the victory screen", f.Screen)
	}
	f.Tick([2]input.TickInput{})
	if f.Screen != ScreenMainMenu {
		t.Errorf("screen = %v after 300 ticks, want the main menu", f.Screen)
	}
}

func TestArcade_ConfirmOnTheVictoryScreen_SkipsToTheMainMenu(t *testing.T) {
	f, _ := arcadeFlow(t, ladderSrc(`"count": 1`, pool2, "nextOpponent", "continue", arcadeFlowSteps...), scripted(t, 0, 0))
	f.VictoryTicks = 300
	pick(f, 0)
	f.Navigate(NavConfirm)
	finishMatch(t, f, match.SideP1)
	if f.Screen != ScreenVictory {
		t.Fatalf("screen = %v, want the victory screen", f.Screen)
	}
	f.Navigate(NavConfirm)
	if f.Screen != ScreenMainMenu {
		t.Errorf("screen = %v, want the main menu", f.Screen)
	}
}

// lostFirstFight plays an arcade flow of one opponent until the player has
// lost it, on the continue screen.
func lostFirstFight(t *testing.T) (*Flow, *[]string) {
	t.Helper()
	// Candidates b c: a draw of 1 keeps [b c]; the first fight is on s2, the next on s0.
	src := ladderSrc(`"count": 2`, pool2, "nextOpponent", "continue", arcadeFlowSteps...)
	f, loaded := arcadeFlow(t, src, scripted(t, 1, 2, 0))
	pick(f, 0)
	f.Navigate(NavConfirm)
	finishMatch(t, f, match.SideP2)
	if f.Screen != ScreenContinue {
		t.Fatalf("screen = %v (%q), want the continue screen after a loss", f.Screen, f.Message)
	}
	return f, loaded
}

func drainCues(f *Flow) []MenuCue { return f.DrainMenuCues(nil) }

func TestArcade_ContinueScreen_StartsAt10SecondsOnYes(t *testing.T) {
	f, _ := lostFirstFight(t)
	if f.ContinueSeconds() != 10 || f.ContinueCursor != 0 || !slices.Equal(f.ContinueItems(), []string{"Yes", "No"}) {
		t.Errorf("seconds = %d, cursor = %d, items = %v; want 10, Yes (0), [Yes No]", f.ContinueSeconds(), f.ContinueCursor, f.ContinueItems())
	}
	if f.PackScreen() != "continue" {
		t.Errorf("pack screen = %q, want continue", f.PackScreen())
	}
}

func TestArcade_ContinueScreen_LeftAndRightMoveBetweenYesAndNoWithTheMoveSound(t *testing.T) {
	f, _ := lostFirstFight(t)
	drainCues(f)
	f.Navigate(NavRight)
	if f.ContinueCursor != 1 {
		t.Fatalf("cursor = %d after right, want No (1)", f.ContinueCursor)
	}
	f.Navigate(NavLeft)
	if f.ContinueCursor != 0 {
		t.Fatalf("cursor = %d after left, want Yes (0)", f.ContinueCursor)
	}
	want := []MenuCue{{SoundMove, "continue"}, {SoundMove, "continue"}}
	if got := drainCues(f); !slices.Equal(got, want) {
		t.Errorf("cues = %v, want %v", got, want)
	}
	f.Navigate(NavCancel)
	if f.Screen != ScreenContinue {
		t.Errorf("screen = %v after cancel, want the continue screen: there is no leaving it but No or the countdown", f.Screen)
	}
}

func TestArcade_ContinueYes_FightsTheSameOpponentAgainOnTheSameStage(t *testing.T) {
	f, loaded := lostFirstFight(t)
	opponent, stageNo := f.Picks[1], f.StageNo
	*loaded = nil
	f.Navigate(NavConfirm)
	if f.Screen != ScreenMatch || f.Match == nil || f.Match.Progress.RoundsPlayed != 0 {
		t.Fatalf("screen = %v (%q), want a fresh match", f.Screen, f.Message)
	}
	if f.Picks[1] != opponent || f.StageNo != stageNo || !slices.Equal(*loaded, []string{"a", arcadeChars[opponent].DefPath}) {
		t.Errorf("opponent %d on stage %d, loaded %v; want %d on %d, loaded a and %s again", f.Picks[1], f.StageNo, *loaded, opponent, stageNo, arcadeChars[opponent].DefPath)
	}
	finishMatch(t, f, match.SideP1)
	if opponent2, _ := versusOpponentOnStage(t, f); opponent2 == opponent {
		t.Errorf("after winning the retry the ladder stayed on opponent %d, want the next one", opponent2)
	}
}

func TestArcade_ContinueNo_PlaysGameOverThenShowsTheMainMenu(t *testing.T) {
	f, _ := lostFirstFight(t)
	drainCues(f)
	f.Navigate(NavRight)
	f.Navigate(NavConfirm)
	if f.Screen != ScreenMainMenu {
		t.Fatalf("screen = %v, want the main menu", f.Screen)
	}
	want := []MenuCue{{SoundMove, "continue"}, {SoundGameOver, "continue"}}
	if got := drainCues(f); !slices.Equal(got, want) {
		t.Errorf("cues = %v, want the move then the game over sound (and no confirm): %v", got, want)
	}
}

func TestArcade_ContinueCountdown_TicksEachSecondThenGameOverAfter600Ticks(t *testing.T) {
	f, _ := lostFirstFight(t)
	drainCues(f)
	var tickedAt []int
	var seconds []int
	for i := 1; i <= 599; i++ {
		f.Tick([2]input.TickInput{})
		if f.Screen != ScreenContinue {
			t.Fatalf("screen = %v at tick %d, want the continue screen until tick 600", f.Screen, i)
		}
		for _, c := range drainCues(f) {
			if c != (MenuCue{SoundTick, "continue"}) {
				t.Fatalf("cue %v at tick %d, want only tick cues", c, i)
			}
			tickedAt = append(tickedAt, i)
			seconds = append(seconds, f.ContinueSeconds())
		}
	}
	if want := []int{60, 120, 180, 240, 300, 360, 420, 480, 540}; !slices.Equal(tickedAt, want) {
		t.Errorf("tick sounds at ticks %v, want one each second: %v", tickedAt, want)
	}
	if want := []int{9, 8, 7, 6, 5, 4, 3, 2, 1}; !slices.Equal(seconds, want) {
		t.Errorf("seconds shown at each tick sound = %v, want %v", seconds, want)
	}
	if f.ContinueSeconds() != 1 {
		t.Errorf("seconds = %d at tick 599, want 1", f.ContinueSeconds())
	}
	f.Tick([2]input.TickInput{})
	if f.Screen != ScreenMainMenu {
		t.Fatalf("screen = %v at tick 600, want the main menu", f.Screen)
	}
	if got, want := drainCues(f), []MenuCue{{SoundGameOver, "continue"}}; !slices.Equal(got, want) {
		t.Errorf("cues = %v, want %v", got, want)
	}
}

func TestArcade_GameOverOutcome_GoesStraightToTheMainMenu(t *testing.T) {
	src := ladderSrc(`"count": 2`, pool2, "nextOpponent", "gameOver", arcadeFlowSteps...)
	f, _ := arcadeFlow(t, src, scripted(t, 0, 0))
	pick(f, 0)
	f.Navigate(NavConfirm)
	finishMatch(t, f, match.SideP2)
	if f.Screen != ScreenMainMenu {
		t.Errorf("screen = %v (%q), want the main menu without a continue screen", f.Screen, f.Message)
	}
}

func TestArcade_AnOutcomeWhoseScreenTheFlowLacks_ShowsAnErrorNamingIt(t *testing.T) {
	cases := []struct {
		name, onWin, onLose string
		winner              match.Side
		flow                []string
		want                string
	}{
		{"continue without a continue screen", "nextOpponent", "continue", match.SideP2, []string{stepSelectP1, stepVS, stepMatch, stepVictory}, "continue"},
		{"the last opponent without a victory screen", "nextOpponent", "continue", match.SideP1, []string{stepSelectP1, stepVS, stepMatch, stepContinue}, "victory"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, _ := arcadeFlow(t, ladderSrc(`"count": 1`, pool2, c.onWin, c.onLose, c.flow...), scripted(t, 0, 0))
			pick(f, 0)
			f.Navigate(NavConfirm)
			finishMatch(t, f, c.winner)
			if f.Screen != ScreenError || !strings.Contains(f.Message, c.want) {
				t.Errorf("screen = %v, message = %q, want an error naming %s", f.Screen, f.Message, c.want)
			}
		})
	}
}

func TestArcade_BuiltInMode_FightsTheFixtureRivalThenVictory(t *testing.T) {
	builtIns, err := mode.BuiltIn()
	if err != nil {
		t.Fatal(err)
	}
	fsys := os.DirFS(content)
	chars, _ := DiscoverCharacters(fsys, "chars")
	stages, _ := DiscoverStages(fsys, "stages")
	f := NewFlow(fsys, chars, stages, DefaultLoader(fsys), builtIns[:1])
	f.Rand = scripted(t, 0)
	f.VictoryTicks = 60
	inMode(t, f)
	pick(f, 0) // fighter: the only other character is the rival
	if f.Screen != ScreenVersus || f.Picks != [2]int{0, 1} {
		t.Fatalf("screen = %v, picks = %v, want the VS screen: fighter against rival", f.Screen, f.Picks)
	}
	f.Navigate(NavConfirm)
	if f.Screen != ScreenMatch || !f.Setup().CPU[1] || f.Setup().CPU[0] {
		t.Fatalf("screen = %v (%q), cpu = %v, want a match with the CPU on side 2", f.Screen, f.Message, f.Setup().CPU)
	}
	finishMatch(t, f, match.SideP1)
	if f.Screen != ScreenVictory {
		t.Errorf("screen = %v, want the victory screen: the ladder had one candidate", f.Screen)
	}
}

func TestRosterNotices_ANamedFinalThatIsNotInstalledIsReported(t *testing.T) {
	m, err := mode.Parse("arcade.jsonc", []byte(ladderSrc(`"count": 2, "final": "ghost"`, "", "nextOpponent", "continue", arcadeFlowSteps...)))
	if err != nil {
		t.Fatal(err)
	}
	got := RosterNotices([]mode.Mode{m}, arcadeChars, arcadeStages)
	want := []string{`mode "Arcade": rules.opponents.final names "ghost", which is not an installed character`}
	if !slices.Equal(got, want) {
		t.Errorf("notices = %q, want %q", got, want)
	}
	installed, err := mode.Parse("arcade.jsonc", []byte(ladderSrc(`"count": 2, "final": "BOSS"`, "", "nextOpponent", "continue", arcadeFlowSteps...)))
	if err != nil {
		t.Fatal(err)
	}
	if got := RosterNotices([]mode.Mode{installed}, arcadeChars, arcadeStages); len(got) != 0 {
		t.Errorf("notices = %q, want none for an installed final (ignoring case)", got)
	}
}
