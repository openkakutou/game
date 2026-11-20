package game

import (
	"slices"
	"strconv"
)

// The options screens: ScreenOptions lists the concerns, each concern has a
// screen of rows changed with left/right and saved when it is left.

// Rows of the concern screens; the last row of each is Back.
const (
	gameplayRowRounds = iota
	gameplayRowTime
)

const (
	audioRowMaster = iota
	audioRowMusic
	audioRowEffects
)

const (
	videoRowFullscreen = iota
	videoRowVSync
	videoRowWindow
)

// volumeStep is how much one left/right changes a volume.
const volumeStep = 10

// WindowSizes are the window sizes the video screen offers, besides a custom
// size read from the file.
var WindowSizes = [][2]int{{640, 480}, {960, 720}, {1280, 960}, {1920, 1440}}

var optionsItems = []string{"Gameplay", "Audio", "Video", "Controls", "Back"}

var concernScreens = [concernCount]Screen{ScreenOptionsGameplay, ScreenOptionsAudio, ScreenOptionsVideo, ScreenOptionsControls}

var concernTitles = [concernCount]string{"Gameplay", "Audio", "Video", "Controls"}

// optionsState is the options screens' own state.
type optionsState struct {
	// cursor is the highlighted row of the screen shown.
	cursor int
	// fromPause is set when Options was opened from the pause menu: leaving
	// it goes back to the paused match.
	fromPause bool
	concern   Concern
	// rows are the labels of the concern screen, rebuilt when a value changes.
	rows []string
	// gameplay, audio and video are the values when the concern screen was
	// entered, to save only what changed.
	gameplay Gameplay
	audio    Audio
	video    Video
	controls Controls
	// bind is the state of the bindings screen.
	bind bindState
}

// OptionsTitle is the heading of the options screen shown.
func (f *Flow) OptionsTitle() string {
	switch f.Screen {
	case ScreenOptions:
		return "Options"
	case ScreenOptionsBindings:
		return bindTitles[f.options.bind.player]
	}
	return concernTitles[f.options.concern]
}

// OptionsItems are the rows of the options screen shown.
func (f *Flow) OptionsItems() []string {
	if f.Screen == ScreenOptions {
		return optionsItems
	}
	return f.options.rows
}

// OptionsCursor is the highlighted row of the options screen shown.
func (f *Flow) OptionsCursor() int { return f.options.cursor }

// inOptions is whether an options screen is shown.
func (f *Flow) inOptions() bool {
	return f.Screen == ScreenOptions || f.inConcern()
}

// inConcern is whether a screen of one concern is shown.
func (f *Flow) inConcern() bool {
	return slices.Contains(concernScreens[:], f.Screen) || f.Screen == ScreenOptionsBindings
}

func (f *Flow) openOptions(fromPause bool) {
	f.options.fromPause = fromPause
	f.Screen, f.options.cursor = ScreenOptions, 0
}

func (f *Flow) closeOptions() {
	if f.options.fromPause {
		f.Screen = ScreenMatch
		return
	}
	f.Screen = ScreenMainMenu
}

func (f *Flow) navOptions(n Nav) {
	switch n {
	case NavUp:
		f.options.cursor = wrap(f.options.cursor, -1, len(optionsItems))
	case NavDown:
		f.options.cursor = wrap(f.options.cursor, 1, len(optionsItems))
	case NavConfirm:
		if f.options.cursor < int(concernCount) {
			f.enterConcern(Concern(f.options.cursor))
			return
		}
		f.closeOptions()
	case NavCancel:
		f.closeOptions()
	}
}

func (f *Flow) enterConcern(c Concern) {
	s := f.Settings
	f.options.concern, f.options.cursor = c, 0
	f.options.gameplay, f.options.audio, f.options.video, f.options.controls = s.Gameplay, s.Audio, s.Video, s.Controls
	f.Screen = concernScreens[c]
	f.refreshRows()
}

// saveChanged saves the concern being edited when it changed since it was
// entered.
func (f *Flow) saveChanged() {
	c, s := f.options.concern, f.Settings
	changed := false
	switch c {
	case ConcernGameplay:
		changed = s.Gameplay != f.options.gameplay
	case ConcernAudio:
		changed = s.Audio != f.options.audio
	case ConcernVideo:
		changed = s.Video != f.options.video
	case ConcernControls:
		changed = s.Controls != f.options.controls
	}
	if changed {
		s.Save(c)
		f.options.gameplay, f.options.audio, f.options.video, f.options.controls = s.Gameplay, s.Audio, s.Video, s.Controls
	}
}

// leaveConcern goes back to the concerns list, saving the concern when it
// changed since it was entered.
func (f *Flow) leaveConcern() {
	f.saveChanged()
	f.Screen, f.options.cursor = ScreenOptions, int(f.options.concern)
}

// SaveOpenConcern saves the concern screen being shown, if it changed: the
// host calls it when the game exits, so a closed window loses no change.
func (f *Flow) SaveOpenConcern() {
	if f.inConcern() {
		f.saveChanged()
	}
}

func (f *Flow) navConcern(n Nav) {
	last := len(f.options.rows) - 1
	switch n {
	case NavUp:
		f.options.cursor = wrap(f.options.cursor, -1, len(f.options.rows))
	case NavDown:
		f.options.cursor = wrap(f.options.cursor, 1, len(f.options.rows))
	case NavLeft:
		f.changeRow(-1)
	case NavRight:
		f.changeRow(1)
	case NavConfirm:
		switch {
		case f.options.cursor == last:
			f.leaveConcern()
		case f.options.concern == ConcernControls:
			f.confirmControlsRow()
		}
	case NavCancel:
		f.leaveConcern()
	}
}

// changeRow moves the highlighted row's value one step (delta -1 or 1) and
// refreshes the labels.
func (f *Flow) changeRow(delta int) {
	s, row := f.Settings, f.options.cursor
	switch f.options.concern {
	case ConcernGameplay:
		switch row {
		case gameplayRowRounds:
			s.Gameplay.Rounds = cycle(withOption(RoundOptions, s.Gameplay.Rounds), s.Gameplay.Rounds, delta)
		case gameplayRowTime:
			s.Gameplay.TimeLimitSeconds = cycle(withOption(TimeOptions, s.Gameplay.TimeLimitSeconds), s.Gameplay.TimeLimitSeconds, delta)
		}
	case ConcernAudio:
		before := s.Audio
		switch row {
		case audioRowMaster:
			s.Audio.Master = stepVolume(s.Audio.Master, delta)
		case audioRowMusic:
			s.Audio.Music = stepVolume(s.Audio.Music, delta)
		case audioRowEffects:
			s.Audio.Effects = stepVolume(s.Audio.Effects, delta)
		}
		if s.Audio != before {
			s.AudioGen++
		}
	case ConcernControls:
		// The rows are actions, not values.
	case ConcernVideo:
		before := s.Video
		switch row {
		case videoRowFullscreen:
			s.Video.Fullscreen = !s.Video.Fullscreen
		case videoRowVSync:
			s.Video.VSync = !s.Video.VSync
		case videoRowWindow:
			size := cycle(windowSizes(s.Video), [2]int{s.Video.WindowWidth, s.Video.WindowHeight}, delta)
			s.Video.WindowWidth, s.Video.WindowHeight = size[0], size[1]
		}
		if s.Video != before {
			s.VideoGen++
		}
		if s.Video.WindowWidth != before.WindowWidth || s.Video.WindowHeight != before.WindowHeight {
			s.WindowGen++
		}
	}
	f.refreshRows()
}

// cycle is the element of options after current (before, when delta is -1),
// wrapping around; options contains current.
func cycle[T comparable](options []T, current T, delta int) T {
	return options[wrap(slices.Index(options, current), delta, len(options))]
}

func stepVolume(v, delta int) int {
	return min(100, max(0, v+delta*volumeStep))
}

// windowSizes is WindowSizes with v's size added in order when it lacks it.
func windowSizes(v Video) [][2]int {
	sizes := slices.Clone(WindowSizes)
	current := [2]int{v.WindowWidth, v.WindowHeight}
	if !slices.Contains(sizes, current) {
		sizes = append(sizes, current)
		slices.SortFunc(sizes, func(a, b [2]int) int {
			if a[0] != b[0] {
				return a[0] - b[0]
			}
			return a[1] - b[1]
		})
	}
	return sizes
}

func onOff(b bool) string {
	if b {
		return "On"
	}
	return "Off"
}

// refreshRows rebuilds the labels of the concern screen from the settings.
func (f *Flow) refreshRows() {
	s := f.Settings
	rows := f.options.rows[:0]
	switch f.options.concern {
	case ConcernGameplay:
		limit := "No limit"
		if s.Gameplay.TimeLimitSeconds > 0 {
			limit = strconv.Itoa(s.Gameplay.TimeLimitSeconds)
		}
		rows = append(rows, "Rounds  "+strconv.Itoa(s.Gameplay.Rounds), "Time  "+limit)
	case ConcernAudio:
		rows = append(rows,
			"Master  "+strconv.Itoa(s.Audio.Master),
			"Music  "+strconv.Itoa(s.Audio.Music),
			"Effects  "+strconv.Itoa(s.Audio.Effects))
	case ConcernControls:
		rows = append(rows, "Player 1", "Player 2", "Reset to defaults")
	case ConcernVideo:
		rows = append(rows,
			"Fullscreen  "+onOff(s.Video.Fullscreen),
			"VSync  "+onOff(s.Video.VSync),
			"Window  "+strconv.Itoa(s.Video.WindowWidth)+"x"+strconv.Itoa(s.Video.WindowHeight))
	}
	f.options.rows = append(rows, "Back")
}
