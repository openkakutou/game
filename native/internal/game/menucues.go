package game

// MenuSound is the name of one of the screenpack's menu sounds (format.md,
// "Root file": move, confirm, cancel, error).
type MenuSound string

// The menu sounds.
const (
	SoundMove    MenuSound = "move"
	SoundConfirm MenuSound = "confirm"
	SoundCancel  MenuSound = "cancel"
	SoundError   MenuSound = "error"
)

// MenuCue is a menu sound the player should hear, and the screenpack screen
// it happened on ("" when none), whose sounds may override the pack's.
type MenuCue struct {
	Sound  MenuSound
	Screen string
}

// maxMenuCues bounds the undrained queue (a host without audio never drains it).
const maxMenuCues = 16

// cue queues a menu sound on the screen now shown.
func (f *Flow) cue(s MenuSound) { f.cueOn(s, f.PackScreen()) }

func (f *Flow) cueOn(s MenuSound, screen string) {
	if len(f.menuCues) < maxMenuCues {
		f.menuCues = append(f.menuCues, MenuCue{s, screen})
	}
}

// DrainMenuCues returns dst truncated to length 0 followed by every pending
// menu cue, in order, and empties the queue; like Match.DrainCues it does not
// allocate when nothing is pending.
func (f *Flow) DrainMenuCues(dst []MenuCue) []MenuCue {
	dst = append(dst[:0], f.menuCues...)
	f.menuCues = f.menuCues[:0]
	return dst
}

// PackScreen is the name of the screenpack screen the flow shows ("" for a
// match that is not paused), as format.md names them.
func (f *Flow) PackScreen() string {
	switch f.Screen {
	case ScreenTitle, ScreenMainMenu:
		return "title"
	case ScreenSelect:
		return "characterSelect"
	case ScreenStage:
		return "stageSelect"
	case ScreenSetup:
		return "matchSetup"
	case ScreenVersus:
		return "versus"
	case ScreenResult:
		return "results"
	case ScreenError:
		return "error"
	case ScreenOptions, ScreenOptionsGameplay, ScreenOptionsAudio, ScreenOptionsVideo,
		ScreenOptionsControls, ScreenOptionsBindings, ScreenOptionsAppearance:
		return "options"
	case ScreenMatch:
		if f.Paused {
			return "pause"
		}
	}
	return ""
}

// navState is everything a menu input can change: two equal states mean the
// input moved nothing.
type navState struct {
	screen                                                    Screen
	step, cursor, menuCursor, setupRow, roundIdx, timeIdx     int
	pauseCursor, resultCursor, optionsCursor, picking, pickAt int
	picks                                                     [2]int
	stageNo                                                   int
	player2CPU, paused, quit                                  bool
	gameplay                                                  Gameplay
	audio                                                     Audio
	video                                                     Video
	controls                                                  Controls
	bindDevice                                                Device
	bindPhase                                                 bindPhase
}

func (f *Flow) navState() navState {
	s := f.Settings
	return navState{
		screen: f.Screen, step: f.Step, cursor: f.Cursor, menuCursor: f.MenuCursor, setupRow: f.SetupRow,
		roundIdx: f.RoundIdx, timeIdx: f.TimeIdx, pauseCursor: f.PauseCursor, resultCursor: f.ResultCursor,
		optionsCursor: f.options.cursor, picking: f.Picking, pickAt: f.pickPos, picks: f.Picks, stageNo: f.StageNo,
		player2CPU: f.Player2CPU, paused: f.Paused, quit: f.Quit,
		gameplay: s.Gameplay, audio: s.Audio, video: s.Video, controls: s.Controls,
		bindDevice: f.options.bind.device, bindPhase: f.options.bind.phase,
	}
}

// Navigate applies a menu input to the current screen and queues its sound:
// move for an arrow that moved a cursor or a value, confirm and cancel for an
// input that changed something, nothing when it did not, and whatever a
// screen queued itself (an error for a refused action) instead.
func (f *Flow) Navigate(n Nav) {
	before, screen, queued := f.navState(), f.PackScreen(), len(f.menuCues)
	f.navigate(n)
	if len(f.menuCues) > queued || f.navState() == before {
		return
	}
	switch n {
	case NavUp, NavDown, NavLeft, NavRight:
		f.cueOn(SoundMove, screen)
	case NavConfirm:
		f.cueOn(SoundConfirm, screen)
	case NavCancel:
		f.cueOn(SoundCancel, screen)
	case NavPause:
		switch {
		case before.screen != ScreenMatch: // the title's start
			f.cueOn(SoundConfirm, screen)
		case before.paused: // start closes the pause menu like cancel; opening it is silent
			f.cueOn(SoundCancel, screen)
		}
	}
}
