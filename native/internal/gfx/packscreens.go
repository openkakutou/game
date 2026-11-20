package gfx

import (
	"fmt"
	"time"

	"github.com/openkakutou/engine/v2/match"
	"github.com/openkakutou/game/native/internal/game"
	"github.com/openkakutou/game/native/internal/ui"
)

// packScreen says which screenpack screen the flow is showing, if any.
type packScreen uint8

const (
	packNone packScreen = iota
	packTitle
	packOptions
	packSelect
	packStage
	packPause
	packSetup
	packVersus
	packResult
	packError
)

// packScreenOf is the screenpack screen drawing f's screen: none for the
// screens still drawn by the renderer itself. Pause is drawn over the match.
func packScreenOf(f *game.Flow) packScreen {
	switch f.Screen {
	case game.ScreenTitle, game.ScreenMainMenu:
		return packTitle
	case game.ScreenOptions, game.ScreenOptionsGameplay, game.ScreenOptionsAudio, game.ScreenOptionsVideo,
		game.ScreenOptionsControls, game.ScreenOptionsBindings:
		return packOptions
	case game.ScreenSelect:
		return packSelect
	case game.ScreenStage:
		return packStage
	case game.ScreenSetup:
		return packSetup
	case game.ScreenVersus:
		return packVersus
	case game.ScreenResult:
		return packResult
	case game.ScreenError:
		return packError
	case game.ScreenMatch:
		if f.Paused {
			return packPause
		}
	}
	return packNone
}

// drawPack draws a screenpack screen over the whole window, in pixels.
func (r *Renderer) drawPack(f *game.Flow, kind packScreen) {
	width, height := r.win.BeginPixels()
	view := ui.View{W: width, H: height}
	r.list.Reset()
	ms := int(time.Since(r.packSince).Milliseconds())
	switch kind {
	case packTitle:
		r.layoutTitle(f, view, ms)
	case packOptions:
		r.layoutOptions(f, view, ms)
	case packSelect:
		r.layoutSelect(f, view, ms)
	case packStage:
		r.layoutStage(f, view, ms)
	case packPause:
		r.layoutPause(f, view, ms)
	case packSetup:
		r.layoutSetup(f, view, ms)
	case packVersus:
		r.layoutVersus(f, view, ms)
	case packResult:
		r.layoutResult(f, view, ms)
	case packError:
		r.layoutError(f, view, ms)
	}
	r.painter.draw(&r.list)
	r.win.EndPixels()
}

// layoutTitle is the "press start" phase on the game's title, the menu phase
// on its main menu.
func (r *Renderer) layoutTitle(f *game.Flow, view ui.View, ms int) {
	state := ui.TitleState{Phase: ui.PhasePressStart, ElapsedMs: ms}
	if f.Screen == game.ScreenMainMenu {
		state.Phase, state.Entries, state.Cursor = ui.PhaseMenu, f.MainMenuItems(), f.MenuCursor
	}
	r.layout.Title(&r.list, view, state)
}

func (r *Renderer) layoutOptions(f *game.Flow, view ui.View, ms int) {
	r.rows = r.rows[:0]
	for _, row := range f.OptionsRows() {
		r.rows = append(r.rows, ui.Row{Label: row.Label, Values: row.Values, Flagged: row.Flagged})
	}
	_, waiting := f.WaitingForBinding()
	r.layout.Options(&r.list, view, ui.OptionsState{
		Title: f.OptionsTitle(), Rows: r.rows, Cursor: f.OptionsCursor(),
		Controls: f.Screen == game.ScreenOptionsBindings, Column: f.OptionsColumn(), Waiting: waiting,
		Prompt: f.OptionsPrompt(), Notices: f.Settings.Notices(), ElapsedMs: ms,
	})
}

// layoutPause lists the pause menu, titled PAUSE as Ikemen GO's default
// [Pause Menu]; the match stays drawn underneath.
func (r *Renderer) layoutPause(f *game.Flow, view ui.View, ms int) {
	r.labels = r.labels[:0]
	for _, item := range f.PauseItems() {
		r.labels = append(r.labels, item.Label())
	}
	r.layout.Pause(&r.list, view, ui.PauseState{Title: "PAUSE", Entries: r.labels, Cursor: f.PauseCursor, ElapsedMs: ms})
}

// setupRows are the setup screen's rows, built again only when the choices
// change (formatting them every frame would allocate).
type setupRows struct {
	built  bool
	choice game.Setup
	hasP2  bool
	rows   []ui.Row
}

func (c *setupRows) update(f *game.Flow) {
	choice, hasP2 := f.Setup(), f.HasPlayer2Row()
	if c.built && choice == c.choice && hasP2 == c.hasP2 {
		return
	}
	c.built, c.choice, c.hasP2 = true, choice, hasP2
	limit := "No limit"
	if choice.TimeLimitSeconds > 0 {
		limit = fmt.Sprintf("%d seconds", choice.TimeLimitSeconds)
	}
	c.rows = append(c.rows[:0],
		ui.Row{Label: "Rounds", Values: [2]string{fmt.Sprintf("best of %d", choice.Rounds)}},
		ui.Row{Label: "Time", Values: [2]string{limit}})
	if hasP2 {
		who := "Human"
		if choice.CPU[1] {
			who = "CPU"
		}
		c.rows = append(c.rows, ui.Row{Label: "Player 2", Values: [2]string{who}})
	}
	c.rows = append(c.rows, ui.Row{Label: "Start match"})
}

func (r *Renderer) layoutSetup(f *game.Flow, view ui.View, ms int) {
	r.setup.update(f)
	cursor := f.SetupRow
	if !r.setup.hasP2 && cursor > game.SetupRowPlayer2 {
		cursor-- // the player 2 row is not listed
	}
	r.layout.MatchSetup(&r.list, view, ui.MatchSetupState{Title: "Match setup", Rows: r.setup.rows, Cursor: cursor, ElapsedMs: ms})
}

// selectTitles name the side picking, indexed by side.
var selectTitles = [2]string{"Player 1: select your fighter", "Player 2: select your fighter"}

// layoutSelect draws the character select: the sides' cursors, the grid
// scrolled to the picking side's.
func (r *Renderer) layoutSelect(f *game.Flow, view ui.View, ms int) {
	var sides [2]ui.SideSelect
	for i, c := range f.SelectCursors() {
		sides[i] = ui.SideSelect{Shown: c.Shown, Cursor: c.Cursor}
	}
	r.layout.CharacterSelect(&r.list, view, ui.SelectState{
		Title: selectTitles[f.Picking], Roster: f.Chars, Sides: sides, Focus: f.Cursor,
		Portraits: r.portraits, ElapsedMs: ms,
	})
}

func (r *Renderer) layoutStage(f *game.Flow, view ui.View, ms int) {
	r.layout.StageSelect(&r.list, view, ui.StageSelectState{Title: "Select a stage", Roster: f.Stages, Cursor: f.Cursor, ElapsedMs: ms})
}

// charName is the name of the character side picked, "" before any pick.
func charName(f *game.Flow, side int) string {
	pick := f.Picks[side]
	if pick >= len(f.Chars) {
		return ""
	}
	if f.Chars[pick].Display == "" {
		return f.Chars[pick].Name
	}
	return f.Chars[pick].Display
}

func (r *Renderer) layoutVersus(f *game.Flow, view ui.View, ms int) {
	stage := ""
	if f.StageNo < len(f.Stages) {
		stage = f.Stages[f.StageNo].Display
	}
	r.layout.Versus(&r.list, view, ui.VersusState{
		P1: charName(f, 0), P2: charName(f, 1), Stage: stage,
		P1Char: f.Picks[0], P2Char: f.Picks[1], Portraits: r.portraits, ElapsedMs: ms,
	})
}

// resultTexts are the result screen's texts, built again only when the
// match's outcome changes.
type resultTexts struct {
	built          bool
	winner         match.Side
	wins           [2]int
	message, score string
}

func (c *resultTexts) update(f *game.Flow) {
	wins := f.Match.Progress.Wins
	if c.built && c.winner == f.ResultWinner && c.wins == wins {
		return
	}
	c.built, c.winner, c.wins = true, f.ResultWinner, wins
	c.message = fmt.Sprintf("Player %d wins", int(f.ResultWinner)+1)
	c.score = fmt.Sprintf("%d - %d", wins[0], wins[1])
}

func (r *Renderer) layoutResult(f *game.Flow, view ui.View, ms int) {
	r.result.update(f)
	r.layout.Results(&r.list, view, ui.ResultsState{
		Message: r.result.message, Score: r.result.score, Winner: charName(f, int(f.ResultWinner)),
		WinnerChar: f.Picks[f.ResultWinner], Portraits: r.portraits,
		Entries: f.ResultItems(), Cursor: f.ResultCursor, ElapsedMs: ms,
	})
}

const errorTitle = "Something went wrong"

// layoutError says what went wrong and what Enter does: quit, or back to the
// main menu for an error a mode caused.
func (r *Renderer) layoutError(f *game.Flow, view ui.View, ms int) {
	hint := "Press Enter to quit"
	if f.ErrorReturnsToMenu() {
		hint = "Press Enter to return to the main menu"
	}
	r.layout.Error(&r.list, view, ui.ErrorState{Title: errorTitle, Message: f.Message, Hint: hint, ElapsedMs: ms})
}
