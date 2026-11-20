package game

// PauseItem is one entry of the in-match pause menu.
type PauseItem int

// Pause menu entries, named as Ikemen GO's default [Pause Menu]
// (resources/defaultMotif.ini: menu.itemname.back = Continue,
// characterchange = Character Change, exit = Exit). Restart and Options are
// OpenKakutou's.
const (
	PauseContinue PauseItem = iota
	PauseRestart
	PauseCharacterChange
	PauseOptions
	PauseExit
)

// Label is the entry's text.
func (p PauseItem) Label() string {
	switch p {
	case PauseContinue:
		return "Continue"
	case PauseRestart:
		return "Restart"
	case PauseCharacterChange:
		return "Character Change"
	case PauseOptions:
		return "Options"
	}
	return "Exit"
}

var (
	pauseItems      = []PauseItem{PauseContinue, PauseRestart, PauseCharacterChange, PauseOptions, PauseExit}
	quickPauseItems = []PauseItem{PauseContinue, PauseRestart, PauseOptions, PauseExit}
)

// PauseItems are the entries the pause menu offers: a quick versus match has
// no character select to go back to.
func (f *Flow) PauseItems() []PauseItem {
	if f.quick != nil {
		return quickPauseItems
	}
	return pauseItems
}

// navPause handles a menu input while the match is paused: Start or cancel
// closes the menu, as Ikemen GO's menu.cancel.key.
func (f *Flow) navPause(n Nav) {
	items := f.PauseItems()
	switch n {
	case NavUp:
		f.PauseCursor = wrap(f.PauseCursor, -1, len(items))
	case NavDown:
		f.PauseCursor = wrap(f.PauseCursor, 1, len(items))
	case NavCancel, NavPause:
		f.Paused = false
	case NavConfirm:
		f.choosePause(items[f.PauseCursor])
	}
}

func (f *Flow) choosePause(item PauseItem) {
	if item == PauseOptions {
		f.openOptions(true) // the match stays paused underneath
		return
	}
	f.Paused = false
	switch item {
	case PauseRestart:
		f.start(f.lastChars, f.lastStage)
	case PauseCharacterChange:
		f.toSelect()
	case PauseExit:
		if f.quick != nil {
			f.Quit = true
			return
		}
		f.toMenu()
	}
}
