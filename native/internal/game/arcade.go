package game

import (
	"fmt"
	"slices"
	"strings"

	"github.com/openkakutou/game/native/internal/mode"
)

// continueItems are the entries of the continue screen: Yes fights the same
// opponent again, No ends the run.
var continueItems = []string{"Yes", "No"}

// ContinueItems are the entries the continue screen offers.
func (f *Flow) ContinueItems() []string { return continueItems }

// ContinueSeconds is the countdown the continue screen shows: the whole
// seconds left, rounded up.
func (f *Flow) ContinueSeconds() int {
	return (f.continueTicks + TickRateHz - 1) / TickRateHz
}

// isLadder is whether the mode being played fights a ladder of CPU opponents.
func (f *Flow) isLadder() bool {
	return f.Mode.Rules.Opponents.Kind == mode.OpponentsLadder
}

// ensureLadder draws the ladder of a ladder mode, once the player has picked
// (it is nil again whenever the flow goes back to the select or the mode is
// entered), and sets up its first fight. It reports false after showing the
// error screen of a mode that has nobody to fight.
func (f *Flow) ensureLadder() bool {
	if !f.isLadder() || f.ladder != nil {
		return true
	}
	f.ladder = f.drawLadder()
	if len(f.ladder) == 0 {
		f.fail(fmt.Sprintf("The %q mode has no opponent to fight: roster.characters.opponents leaves none of the installed characters.", f.Mode.Name))
		return false
	}
	f.ladderPos = 0
	f.setUpFight()
	return true
}

// drawLadder picks the fights of a ladder: Count characters of the opponent
// pool at random without repeats, then the final, when it is installed. The
// player's character is left out unless it is all the pool has; the final is
// left out of the draw as it is fought last anyway.
func (f *Flow) drawLadder() []int {
	opp := f.Mode.Rules.Opponents
	final := -1
	if opp.Final != "" {
		final = slices.IndexFunc(f.Chars, func(e Entry) bool { return strings.EqualFold(e.Name, opp.Final) })
	}
	pool := slices.DeleteFunc(f.OpponentPool(), func(i int) bool { return i == final })
	candidates := slices.DeleteFunc(slices.Clone(pool), func(i int) bool { return i == f.Picks[0] })
	if len(candidates) == 0 {
		candidates = pool
	}
	// Fisher-Yates, from the last cell down.
	for i := len(candidates) - 1; i > 0; i-- {
		j := f.Rand(i + 1)
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}
	ladder := candidates[:min(opp.Count, len(candidates))]
	if final >= 0 {
		ladder = append(ladder, final)
	}
	return ladder
}

// setUpFight makes the ladder's current opponent player 2's character and
// draws a visible stage for the fight.
func (f *Flow) setUpFight() {
	f.Picks[1] = f.ladder[f.ladderPos]
	f.StageNo = f.stageView[f.Rand(len(f.stageView))]
}

// nextOpponent follows a won ladder fight: the VS screen of the next opponent,
// or the victory screen after the last one.
func (f *Flow) nextOpponent() {
	if f.ladderPos+1 >= len(f.ladder) {
		f.enterScreen(mode.ScreenVictory, "The game mode ends on victory but has no victory screen.")
		return
	}
	f.ladderPos++
	f.setUpFight()
	i := slices.IndexFunc(f.Mode.Flow, func(s mode.Step) bool { return s.Screen == mode.ScreenVersus })
	if i < 0 {
		i = slices.IndexFunc(f.Mode.Flow, func(s mode.Step) bool { return s.Screen == mode.ScreenMatch })
	}
	f.enter(i, false)
}

func (f *Flow) navContinue(n Nav) {
	switch n {
	case NavLeft, NavRight:
		f.ContinueCursor = wrap(f.ContinueCursor, 1, len(continueItems))
	case NavConfirm:
		if f.ContinueCursor == 0 {
			f.startMatch()
			return
		}
		f.gameOver()
	}
}

// gameOver plays the game over sound and shows the main menu.
func (f *Flow) gameOver() {
	f.cueOn(SoundGameOver, "continue")
	f.toMenu()
}

// tickContinue counts the continue screen down one tick: a tick sound each
// second, the game over at 0.
func (f *Flow) tickContinue() {
	f.continueTicks--
	if f.continueTicks <= 0 {
		f.gameOver()
		return
	}
	if f.continueTicks%TickRateHz == 0 {
		f.cue(SoundTick)
	}
}

// tickVictory counts the victory screen down one tick, then shows the main menu.
func (f *Flow) tickVictory() {
	f.victoryTicks--
	if f.victoryTicks <= 0 {
		f.toMenu()
	}
}
