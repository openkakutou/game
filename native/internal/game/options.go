package game

import (
	"slices"
	"strconv"
	"strings"

	"github.com/openkakutou/game/native/internal/lifebar"
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

const (
	appearanceRowScreenpack = iota
	appearanceRowLifebar
)

// volumeStep is how much one left/right changes a volume.
const volumeStep = 10

// WindowSizes are the window sizes the video screen offers, besides a custom
// size read from the file.
var WindowSizes = [][2]int{{640, 480}, {960, 720}, {1280, 960}, {1920, 1440}}

var optionsItems = []string{"Gameplay", "Audio", "Video", "Controls", "Appearance", "Back"}

var concernScreens = [concernCount]Screen{ScreenOptionsGameplay, ScreenOptionsAudio, ScreenOptionsVideo, ScreenOptionsControls, ScreenOptionsAppearance}

var concernTitles = [concernCount]string{"Gameplay", "Audio", "Video", "Controls", "Appearance"}

// OptionRow is one row of an options screen: its label, its values (one for
// a setting, a key then a button for a binding; "" for none) and whether the
// row is flagged as a warning (a binding shared with another action).
type OptionRow struct {
	Label   string
	Values  [2]string
	Flagged bool
}

// optionsRows are the concerns list's rows, labels only.
var optionsRows = func() []OptionRow {
	rows := make([]OptionRow, len(optionsItems))
	for i, label := range optionsItems {
		rows[i] = OptionRow{Label: label}
	}
	return rows
}()

// optionsState is the options screens' own state.
type optionsState struct {
	// cursor is the highlighted row of the screen shown.
	cursor int
	// fromPause is set when Options was opened from the pause menu: leaving
	// it goes back to the paused match.
	fromPause bool
	concern   Concern
	// rows are the labels of the concern screen, rebuilt when a value changes;
	// cells are the same rows split into label and values.
	rows  []string
	cells []OptionRow
	// gameplay, audio and video are the values when the concern screen was
	// entered, to save only what changed.
	gameplay   Gameplay
	audio      Audio
	video      Video
	controls   Controls
	appearance Appearance
	// previewPack and previewLifebar are the candidates the appearance screen
	// shows: the screenpack's name, and the lifebar loaded for the sample
	// (nil for the built-in bars).
	previewPack    string
	previewLifebar *HUDLifebar
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

// OptionsRows are the rows of the options screen shown, split into label and
// values for a screen that lays them out in columns.
func (f *Flow) OptionsRows() []OptionRow {
	if f.Screen == ScreenOptions {
		return optionsRows
	}
	return f.options.cells
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
	f.options.appearance = s.Appearance
	f.options.previewPack, f.options.previewLifebar = f.ScreenpackChoice(), f.Lifebar
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
	case ConcernAppearance:
		changed = s.Appearance != f.options.appearance
	}
	if !changed {
		return
	}
	s.Save(c)
	f.options.gameplay, f.options.audio, f.options.video, f.options.controls = s.Gameplay, s.Audio, s.Video, s.Controls
	f.options.appearance = s.Appearance
	if c == ConcernAppearance {
		// The lifebar the sample showed is the one the HUD uses now; the
		// host switches the screenpack when it sees AppearanceGen change.
		f.Lifebar = f.options.previewLifebar
		s.AppearanceGen++
	}
}

// ScreenpackChoice is the installed screenpack the settings select: the
// listed name matching the saved one without case, else "default".
func (f *Flow) ScreenpackChoice() string {
	return pickName(f.Packs, f.Settings.Appearance.Screenpack)
}

// PreviewPack is the screenpack the appearance screen is drawn with, its
// candidate; "" on any other screen.
func (f *Flow) PreviewPack() string {
	if f.Screen != ScreenOptionsAppearance {
		return ""
	}
	return f.options.previewPack
}

// PreviewLifebar is the lifebar to draw as a sample under the appearance
// screen while its Lifebar row is highlighted; nil when none is shown, or
// the candidate is the built-in bars.
func (f *Flow) PreviewLifebar() *HUDLifebar {
	if f.Screen != ScreenOptionsAppearance || f.options.cursor != appearanceRowLifebar {
		return nil
	}
	return f.options.previewLifebar
}

// pickName is the name of names matching saved without case, else the
// default's, else "default" itself.
func pickName(names []string, saved string) string {
	for _, want := range []string{saved, defaultName} {
		for _, n := range names {
			if strings.EqualFold(n, want) {
				return n
			}
		}
	}
	return defaultName
}

// nextName is the name after (before, when delta is -1) the one saved selects
// among names, wrapping around.
func nextName(names []string, saved string, delta int) string {
	current := pickName(names, saved)
	if !slices.Contains(names, current) {
		names = append(slices.Clone(names), current)
	}
	return cycle(names, current, delta)
}

func lifebarNames(entries []lifebar.Entry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	return names
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
	case ConcernAppearance:
		switch row {
		case appearanceRowScreenpack:
			s.Appearance.Screenpack = nextName(f.Packs, s.Appearance.Screenpack, delta)
			f.options.previewPack = f.ScreenpackChoice()
		case appearanceRowLifebar:
			s.Appearance.Lifebar = nextName(lifebarNames(f.Lifebars), s.Appearance.Lifebar, delta)
			hud, notices := SelectLifebar(f.FS, f.Lifebars, s.Appearance.Lifebar)
			s.NoteAppearance(notices)
			f.options.previewLifebar = hud
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

// addRow appends a row to the concern screen: the label, and its value
// when it has one (the flat text is the label, two spaces, the value).
func (o *optionsState) addRow(label, value string) {
	text := label
	if value != "" {
		text = label + "  " + value
	}
	o.rows = append(o.rows, text)
	o.cells = append(o.cells, OptionRow{Label: label, Values: [2]string{value, ""}})
}

// refreshRows rebuilds the labels of the concern screen from the settings.
func (f *Flow) refreshRows() {
	s, o := f.Settings, &f.options
	o.rows, o.cells = o.rows[:0], o.cells[:0]
	switch o.concern {
	case ConcernGameplay:
		limit := "No limit"
		if s.Gameplay.TimeLimitSeconds > 0 {
			limit = strconv.Itoa(s.Gameplay.TimeLimitSeconds)
		}
		o.addRow("Rounds", strconv.Itoa(s.Gameplay.Rounds))
		o.addRow("Time", limit)
	case ConcernAudio:
		o.addRow("Master", strconv.Itoa(s.Audio.Master))
		o.addRow("Music", strconv.Itoa(s.Audio.Music))
		o.addRow("Effects", strconv.Itoa(s.Audio.Effects))
	case ConcernControls:
		o.addRow("Player 1", "")
		o.addRow("Player 2", "")
		o.addRow("Reset to defaults", "")
	case ConcernAppearance:
		o.addRow("Screenpack", f.ScreenpackChoice())
		o.addRow("Lifebar", pickName(lifebarNames(f.Lifebars), s.Appearance.Lifebar))
	case ConcernVideo:
		o.addRow("Fullscreen", onOff(s.Video.Fullscreen))
		o.addRow("VSync", onOff(s.Video.VSync))
		o.addRow("Window", strconv.Itoa(s.Video.WindowWidth)+"x"+strconv.Itoa(s.Video.WindowHeight))
	}
	o.addRow("Back", "")
}
