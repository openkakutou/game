package game

import (
	"fmt"
	"io/fs"
	"path"
	"slices"

	"github.com/openkakutou/engine/v2/input"
	"github.com/openkakutou/engine/v2/match"
	"github.com/openkakutou/game/native/internal/mode"
	"github.com/openkakutou/stage"
)

// Screen identifies which screen the game is showing.
type Screen int

// The game's screens, in flow order. ScreenError is shown when content cannot
// be loaded or the mode asks for something not supported yet.
const (
	ScreenTitle    Screen = iota
	ScreenMainMenu        // the installed modes, then Quit
	ScreenSelect          // the sides of a characterSelect step pick a character
	ScreenStage           // pick a stage
	ScreenSetup           // rounds and time limit
	ScreenVersus          // both picked characters, before the match
	ScreenMatch
	ScreenResult
	ScreenOptions         // the settings concerns: Gameplay, Audio, Video, Controls, Back
	ScreenOptionsGameplay // the rows of one concern, in Concern order
	ScreenOptionsAudio
	ScreenOptionsVideo
	ScreenOptionsControls // Player 1, Player 2, Reset to defaults, Back
	ScreenOptionsBindings // one player's actions with their key and button
	ScreenError
)

// Nav is a menu input, already decoded from keyboard or gamepad.
type Nav int

// Menu inputs.
const (
	NavUp Nav = iota
	NavDown
	NavLeft
	NavRight
	NavConfirm
	NavCancel
	// NavPause is Escape or a controller's start button in a match: it opens
	// and closes the pause menu, and does nothing elsewhere.
	NavPause
)

// Options offered on the setup screen.
var (
	RoundOptions = []int{1, 3, 5}
	TimeOptions  = []int{0, 30, 60, 99} // seconds; 0 = no limit
)

// VersusTicks is how long the VS screen shows before the match starts:
// 2 seconds.
const VersusTicks = 2 * TickRateHz

// RoundEndTicks is how long a finished round stays on screen before the next
// round starts (or the result screen appears): 2 seconds.
const RoundEndTicks = 2 * TickRateHz

// Rows of the setup screen.
const (
	SetupRowRounds = iota
	SetupRowTime
	SetupRowPlayer2
	SetupRowStart
	setupRowCount
)

// Loader loads content, by the name of its .def in the content file system;
// replaceable so flow logic is testable.
type Loader struct {
	Fighter func(defName string) (*Fighter, error)
	Stage   func(defName string) (*stage.Stage, error)
}

// DefaultLoader reads the content of fsys with the character and stage
// libraries. stateDirs are folders of fsys searched for state files
// (stcommon) a character folder lacks.
func DefaultLoader(fsys fs.FS, stateDirs ...string) Loader {
	return Loader{
		Fighter: func(defName string) (*Fighter, error) { return LoadFighter(fsys, defName, stateDirs...) },
		Stage:   func(defName string) (*stage.Stage, error) { return LoadStage(fsys, defName) },
	}
}

// Flow is the screen state machine: title, main menu, then the steps of the
// chosen mode's flow.
type Flow struct {
	Chars, Stages []Entry
	Loader        Loader
	// Settings are the persisted settings; the host replaces the defaults
	// with LoadSettings' result.
	Settings *Settings
	options  optionsState
	// Modes are the installed modes, in main menu order; Mode is the one being
	// played.
	Modes []mode.Mode
	Mode  mode.Mode
	// MenuCursor is the highlighted main menu item: a mode's index in Modes,
	// then len(Modes) for Options and len(Modes)+1 for Quit.
	MenuCursor int
	menuLabels []string // mode names, then "Options" and "Quit"
	// Step is the index in Mode.Flow of the step being shown.
	Step int

	Screen Screen
	// Picking is which player is choosing a character (0 or 1); pickPos is
	// its position in the characterSelect step's sides.
	Picking int
	pickPos int
	Cursor  int
	Picks   [2]int // chosen roster indexes
	StageNo int

	// RoundIdx and TimeIdx index roundOpts and timeOpts: RoundOptions and
	// TimeOptions plus the rules' own value when they lack it.
	RoundIdx, TimeIdx int
	roundOpts         []int
	timeOpts          []int
	SetupRow          int // a SetupRow* constant
	Player2CPU        bool

	Fighters [2]*Fighter
	Stage    *stage.Stage
	// StageDefPath is the name of the chosen stage's .def, to find its sprite sheet.
	StageDefPath string
	// FS is the content: the game folder, where a stage looks for files not
	// found beside its .def.
	FS fs.FS
	// Lifebar is the HUD's lifebar; nil keeps the built-in bars.
	Lifebar *HUDLifebar
	Match   *Match
	cpus    [2]*CPU // drive the sides the match setup gives to the CPU
	cpuIn   [2]input.TickInput
	// quick is the setup of a quick versus match started from the command
	// line: no menus before it, and the game quits when it ends.
	quick *Setup
	// musicFrom/musicPath memoize Music's resolved path (see Music).
	musicFrom [2]string
	musicPath string
	// versusTicks counts down while the VS screen shows.
	versusTicks int
	// RoundOverTicks counts down while a finished round is shown.
	RoundOverTicks int
	// ResultWinner is the match winner shown on the result screen.
	ResultWinner match.Side
	// Message is the error text shown on ScreenError.
	Message string
	// Paused is set while the pause menu is open over the match; the
	// simulation does not advance.
	Paused bool
	// PauseCursor is the highlighted PauseItems entry.
	PauseCursor int
	// lastChars/lastStage are what the running match was started with, for
	// Restart.
	lastChars [2]Entry
	lastStage Entry
	// Quit is set when the player asks to leave the game.
	Quit bool
}

// NewFlow starts on the title screen, over the content fsys, offering modes
// in the main menu. An empty roster or stage list goes straight to an error
// screen instead of an unusable menu.
func NewFlow(fsys fs.FS, chars, stages []Entry, l Loader, modes []mode.Mode) *Flow {
	f := &Flow{FS: fsys, Chars: chars, Stages: stages, Loader: l, Modes: modes, Settings: NewSettings()}
	for _, m := range modes {
		f.menuLabels = append(f.menuLabels, m.Name)
	}
	f.menuLabels = append(f.menuLabels, "Options", "Quit")
	switch {
	case len(chars) == 0:
		f.fail("No characters found. Put character folders in the characters folder.")
	case len(stages) == 0:
		f.fail("No stages found. Put stage .def files in the stages folder.")
	default:
		f.Screen = ScreenTitle
	}
	return f
}

// MainMenuItems are the main menu's labels: each mode's name, then Options and Quit.
func (f *Flow) MainMenuItems() []string { return f.menuLabels }

// enterMode starts the mode Modes[i] afresh, at its first step.
func (f *Flow) enterMode(i int) {
	m := f.Modes[i]
	f.Mode, f.MenuCursor = m, i
	rounds, timeLimit := m.Rules.Rounds, m.Rules.TimeLimitSeconds
	if hasMatchSetup(m) {
		rounds, timeLimit = f.Settings.Gameplay.Rounds, f.Settings.Gameplay.TimeLimitSeconds
	}
	f.roundOpts = withOption(RoundOptions, rounds)
	f.timeOpts = withOption(TimeOptions, timeLimit)
	f.RoundIdx = slices.Index(f.roundOpts, rounds)
	f.TimeIdx = slices.Index(f.timeOpts, timeLimit)
	f.Picks, f.StageNo, f.Player2CPU = [2]int{}, 0, false
	f.Match, f.Paused = nil, false
	if len(m.Flow) == 0 {
		f.fail("The game mode has no screens.")
		return
	}
	f.enter(0, false)
}

// hasMatchSetup is whether m's flow has a matchSetup step, whose preselected
// values are the gameplay settings; a mode without one keeps its rules.
func hasMatchSetup(m mode.Mode) bool {
	return slices.ContainsFunc(m.Flow, func(s mode.Step) bool { return s.Screen == mode.ScreenMatchSetup })
}

// ErrorReturnsToMenu is true when the error screen came from a mode the
// player chose from the main menu: leaving it goes back there. Errors before
// any mode (content missing) and in quick versus quit.
func (f *Flow) ErrorReturnsToMenu() bool {
	return f.quick == nil && f.Mode.ID != ""
}

// toMenu shows the main menu, its cursor on the mode being played.
func (f *Flow) toMenu() {
	f.Match, f.Paused = nil, false
	f.Screen = ScreenMainMenu
}

func (f *Flow) navTitle(n Nav) {
	switch n {
	case NavConfirm, NavPause:
		f.toMenu()
	case NavCancel:
		f.Quit = true
	}
}

func (f *Flow) navMainMenu(n Nav) {
	switch n {
	case NavUp:
		f.MenuCursor = wrap(f.MenuCursor, -1, len(f.menuLabels))
	case NavDown:
		f.MenuCursor = wrap(f.MenuCursor, 1, len(f.menuLabels))
	case NavConfirm:
		switch f.MenuCursor {
		case len(f.Modes):
			f.openOptions(false)
		case len(f.Modes) + 1:
			f.Quit = true
		default:
			f.enterMode(f.MenuCursor)
		}
	case NavCancel:
		f.Screen = ScreenTitle
	}
}

// withOption returns options, with value added in order when it lacks it.
func withOption(options []int, value int) []int {
	if slices.Contains(options, value) {
		return options
	}
	out := append(slices.Clone(options), value)
	slices.Sort(out)
	return out
}

func (f *Flow) fail(msg string) {
	f.Screen, f.Message = ScreenError, msg
}

// HasPlayer2Row is whether the setup screen lets the player choose who plays
// side 2.
// shortcut: side 1 is human unless its control is cpu, a humanOrCpu side 1
// gets no row; upgrade when a mode needs it.
func (f *Flow) HasPlayer2Row() bool {
	return f.Mode.Rules.Sides[1].Control == mode.ControlHumanOrCPU
}

// Setup returns the currently chosen match setup: the setup screen's choices,
// preselected to the rules (which apply as written when the flow has no
// matchSetup step). A cpu side is always CPU-driven, a human side never.
func (f *Flow) Setup() Setup {
	if f.quick != nil {
		return *f.quick
	}
	sides := f.Mode.Rules.Sides
	cpu := [2]bool{sides[0].Control == mode.ControlCPU, sides[1].Control == mode.ControlCPU || f.Player2CPU}
	return Setup{Rounds: f.roundOpts[f.RoundIdx], TimeLimitSeconds: f.timeOpts[f.TimeIdx], CPU: cpu}
}

func wrap(i, delta, n int) int { return ((i+delta)%n + n) % n }

// Navigate applies a menu input to the current screen.
func (f *Flow) Navigate(n Nav) {
	switch f.Screen {
	case ScreenTitle:
		f.navTitle(n)
	case ScreenMainMenu:
		f.navMainMenu(n)
	case ScreenSelect:
		f.navSelect(n)
	case ScreenStage:
		f.navStage(n)
	case ScreenSetup:
		f.navSetup(n)
	case ScreenVersus:
		f.navVersus(n)
	case ScreenResult:
		f.navResult(n)
	case ScreenOptions:
		f.navOptions(n)
	case ScreenOptionsGameplay, ScreenOptionsAudio, ScreenOptionsVideo, ScreenOptionsControls:
		f.navConcern(n)
	case ScreenOptionsBindings:
		f.navBindings(n)
	case ScreenError:
		if n != NavCancel && n != NavConfirm {
			return
		}
		if f.ErrorReturnsToMenu() {
			f.toMenu()
			return
		}
		f.Quit = true
	case ScreenMatch:
		f.navMatch(n)
	}
}

// navMatch opens the pause menu on NavPause (Escape, the bound start button),
// then hands it the inputs. Cancel (a pad's B, Backspace) is a fighter's
// button there.
func (f *Flow) navMatch(n Nav) {
	if f.Paused {
		f.navPause(n)
		return
	}
	if n == NavPause {
		f.Paused, f.PauseCursor = true, 0
	}
}

// sideIndex is the player index of a characterSelect side name ("p1", "p2").
func sideIndex(side string) int {
	if side == "p2" {
		return 1
	}
	return 0
}

// enter shows step i. returning is set when the player comes back to it with
// cancel, so a characterSelect starts on its last picker instead of its first.
func (f *Flow) enter(i int, returning bool) {
	f.Step = i
	step := f.Mode.Flow[i]
	switch step.Screen {
	case mode.ScreenCharacterSelect:
		if len(step.Sides) == 0 {
			f.fail("The characterSelect screen lists no sides.")
			return
		}
		pos := 0
		if returning {
			pos = len(step.Sides) - 1
		}
		f.Screen = ScreenSelect
		f.setPicker(pos)
		f.Cursor = f.Picks[f.Picking]
	case mode.ScreenStageSelect:
		f.Screen, f.Cursor = ScreenStage, f.StageNo
	case mode.ScreenMatchSetup:
		f.Screen, f.SetupRow = ScreenSetup, SetupRowStart
	case mode.ScreenVersus:
		f.Screen, f.versusTicks = ScreenVersus, VersusTicks
	case mode.ScreenMatch:
		f.startMatch()
	case mode.ScreenResults:
		f.Screen = ScreenResult
	default:
		f.fail(fmt.Sprintf("The %s screen is not supported yet.", step.Screen))
	}
}

// setPicker makes the pos-th side of the current characterSelect step pick.
// A side no characterSelect step lists keeps index 0, the first roster entry.
func (f *Flow) setPicker(pos int) {
	f.pickPos = pos
	f.Picking = sideIndex(f.Mode.Flow[f.Step].Sides[pos])
}

// next shows the step after the current one.
func (f *Flow) next() {
	if f.Step+1 >= len(f.Mode.Flow) {
		f.fail("The game mode has no screen after this one.")
		return
	}
	f.enter(f.Step+1, false)
}

// back shows the previous interactive step, never the VS screen nor the
// match; cancelling the first one shows the main menu.
func (f *Flow) back() {
	for i := f.Step - 1; i >= 0; i-- {
		switch f.Mode.Flow[i].Screen {
		case mode.ScreenCharacterSelect, mode.ScreenStageSelect, mode.ScreenMatchSetup:
			f.enter(i, true)
			return
		}
	}
	f.toMenu()
}

func (f *Flow) navSelect(n Nav) {
	sides := f.Mode.Flow[f.Step].Sides
	switch n {
	case NavUp, NavLeft:
		f.Cursor = wrap(f.Cursor, -1, len(f.Chars))
	case NavDown, NavRight:
		f.Cursor = wrap(f.Cursor, 1, len(f.Chars))
	case NavConfirm:
		f.Picks[f.Picking] = f.Cursor
		if f.pickPos < len(sides)-1 {
			f.setPicker(f.pickPos + 1)
			return
		}
		f.next()
	case NavCancel:
		if f.pickPos > 0 {
			f.setPicker(f.pickPos - 1)
			f.Cursor = f.Picks[f.Picking]
			return
		}
		f.back()
	}
}

func (f *Flow) navStage(n Nav) {
	switch n {
	case NavUp, NavLeft:
		f.Cursor = wrap(f.Cursor, -1, len(f.Stages))
	case NavDown, NavRight:
		f.Cursor = wrap(f.Cursor, 1, len(f.Stages))
	case NavConfirm:
		f.StageNo = f.Cursor
		f.next()
	case NavCancel:
		f.back()
	}
}

// moveRow moves the setup cursor, skipping the player 2 row when it is not
// offered.
func (f *Flow) moveRow(delta int) {
	f.SetupRow = wrap(f.SetupRow, delta, setupRowCount)
	if f.SetupRow == SetupRowPlayer2 && !f.HasPlayer2Row() {
		f.SetupRow = wrap(f.SetupRow, delta, setupRowCount)
	}
}

func (f *Flow) navSetup(n Nav) {
	switch n {
	case NavUp:
		f.moveRow(-1)
	case NavDown:
		f.moveRow(1)
	case NavLeft, NavRight:
		d := 1
		if n == NavLeft {
			d = -1
		}
		switch f.SetupRow {
		case SetupRowRounds:
			f.RoundIdx = wrap(f.RoundIdx, d, len(f.roundOpts))
		case SetupRowTime:
			f.TimeIdx = wrap(f.TimeIdx, d, len(f.timeOpts))
		case SetupRowPlayer2:
			f.Player2CPU = !f.Player2CPU
		}
	case NavConfirm:
		if f.SetupRow == SetupRowStart {
			f.next()
		} else {
			f.moveRow(1)
		}
	case NavCancel:
		f.back()
	}
}

func (f *Flow) navVersus(n Nav) {
	switch n {
	case NavConfirm:
		f.next()
	case NavCancel:
		f.back()
	}
}

func (f *Flow) navResult(n Nav) {
	switch n {
	case NavConfirm: // rematch, same characters and stage
		f.startMatch()
	case NavCancel:
		f.toSelect()
	}
}

// toSelect goes back to the flow's first characterSelect step, P1 picking.
// shortcut: a flow without one shows the main menu, upgrade when a mode needs
// another way to change characters.
func (f *Flow) toSelect() {
	i := slices.IndexFunc(f.Mode.Flow, func(s mode.Step) bool { return s.Screen == mode.ScreenCharacterSelect })
	if i < 0 {
		f.toMenu()
		return
	}
	f.Match, f.Paused = nil, false
	f.enter(i, false)
}

// finishMatch follows the rules' outcome of the match that just ended:
// onWin when player 1 won, onLose otherwise (a match always has a winner;
// player 2 winning is player 1 losing). Only results is supported yet.
func (f *Flow) finishMatch() {
	f.ResultWinner = f.Match.Winner
	outcome := f.Mode.Rules.OnLose
	if f.Match.Winner == match.SideP1 {
		outcome = f.Mode.Rules.OnWin
	}
	if outcome != mode.OutcomeResults {
		f.fail(fmt.Sprintf("%s is not supported yet.", outcome))
		return
	}
	i := slices.IndexFunc(f.Mode.Flow, func(s mode.Step) bool { return s.Screen == mode.ScreenResults })
	if i < 0 {
		f.fail("The game mode ends on results but has no results screen.")
		return
	}
	f.enter(i, false)
}

// startMatch loads both characters and the stage; any failure lands on the
// error screen with a message naming what could not be loaded.
func (f *Flow) startMatch() {
	f.start([2]Entry{f.Chars[f.Picks[0]], f.Chars[f.Picks[1]]}, f.Stages[f.StageNo])
}

// start loads chars and the stage se, then starts the match.
func (f *Flow) start(chars [2]Entry, se Entry) {
	f.lastChars, f.lastStage, f.Paused = chars, se, false
	for i, e := range chars {
		fighter, err := f.Loader.Fighter(e.DefPath)
		if err != nil {
			f.fail(fmt.Sprintf("Could not load character %q: %v", e.Name, err))
			return
		}
		f.Fighters[i] = fighter
	}
	st, err := f.Loader.Stage(se.DefPath)
	if err != nil {
		f.fail(fmt.Sprintf("Could not load stage %q: %v", se.Name, err))
		return
	}
	f.Stage, f.StageDefPath = st, se.DefPath
	m, err := NewMatch(f.Fighters[0], f.Fighters[1], st, f.Setup())
	if err != nil {
		f.fail(fmt.Sprintf("Could not start the match: %v", err))
		return
	}
	f.Match, f.Screen, f.RoundOverTicks = m, ScreenMatch, 0
	for i, cpu := range f.Setup().CPU {
		f.cpus[i] = nil
		if cpu {
			f.cpus[i] = NewCPU(nil)
		}
	}
}

// StartQuickVersus starts q's match straight away, as Ikemen GO's quick VS
// does (main.lua:1023-1256): characters and stage are found in f.FS
// (ResolveCharDef, ResolveStageDef), and the game quits when the match ends
// or the player leaves it. A character that cannot be found is an error
// screen naming it; a stage that cannot be found falls back to the default
// stage with a notice, as Ikemen GO does (main.lua:1150-1155). The default is
// stages/stage0.def (Ikemen GO's Debug.StartStage) or else the first stage
// found.
func (f *Flow) StartQuickVersus(q QuickVersus) (notices []string) {
	setup := q.Setup()
	if err := setup.Validate(); err != nil {
		f.fail(fmt.Sprintf("Cannot start the match: %v", err))
		return nil
	}
	f.quick = &setup
	var chars [2]Entry
	for i, decl := range []string{q.P1, q.P2} {
		def, err := ResolveCharDef(f.FS, decl)
		if err != nil {
			f.fail(fmt.Sprintf("Cannot start the match: %v", err))
			return nil
		}
		chars[i] = Entry{Name: decl, DefPath: def}
	}
	stageDef := ""
	if q.Stage != "" {
		def, err := ResolveStageDef(f.FS, q.Stage)
		if err != nil {
			notices = append(notices, fmt.Sprintf("stage %q not found, using the default stage", q.Stage))
		}
		stageDef = def
	}
	if stageDef == "" {
		stageDef, _ = ResolveStageDef(f.FS, "stage0")
	}
	if stageDef == "" && len(f.Stages) > 0 {
		stageDef = f.Stages[0].DefPath
	}
	if stageDef == "" {
		f.fail("Cannot start the match: no stage found in the content")
		return notices
	}
	f.start(chars, Entry{Name: path.Base(stageDef), DefPath: stageDef})
	return notices
}

// Tick advances the match screen by one simulation tick (60 per second) with
// both players' held inputs, and counts down the VS screen. It does nothing on
// other screens.
func (f *Flow) Tick(in [2]input.TickInput) {
	if f.Screen == ScreenVersus {
		f.versusTicks--
		if f.versusTicks <= 0 {
			f.next()
		}
		return
	}
	if f.Screen != ScreenMatch || f.Match == nil || f.Paused {
		return
	}
	if f.Match.RoundEnded() {
		f.RoundOverTicks--
		if f.RoundOverTicks > 0 {
			return
		}
		if f.Match.Over && f.quick != nil {
			f.Quit = true
			return
		}
		if f.Match.Over {
			f.finishMatch()
			return
		}
		if err := f.Match.NextRound(); err != nil {
			f.fail(err.Error())
		}
		return
	}
	for i, cpu := range f.cpus {
		if cpu == nil {
			continue
		}
		cpu.Decide(f.Match.State.Fighters[i].Position, f.Match.State.Fighters[1-i].Position, &f.cpuIn[i])
		in[i] = f.cpuIn[i]
	}
	if err := f.Match.Step(in); err != nil {
		f.fail(fmt.Sprintf("The match stopped: %v", err))
		return
	}
	if f.Match.RoundEnded() {
		f.RoundOverTicks = RoundEndTicks
	}
}
