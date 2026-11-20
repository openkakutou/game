package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
)

// Persisted settings, one file per concern in config/ (decision 023).

// Gameplay is config/gameplay.json: the match setup's preselected values.
type Gameplay struct {
	Rounds           int `json:"rounds"`
	TimeLimitSeconds int `json:"timeLimitSeconds"`
}

// Audio is config/audio.json: volumes from 0 to 100.
type Audio struct {
	Master  int `json:"master"`
	Music   int `json:"music"`
	Effects int `json:"effects"`
}

// Video is one platform's section of config/video.json.
type Video struct {
	Fullscreen   bool `json:"fullscreen"`
	VSync        bool `json:"vsync"`
	WindowWidth  int  `json:"windowWidth"`
	WindowHeight int  `json:"windowHeight"`
}

// Smallest window the settings accept.
const (
	MinWindowWidth  = 320
	MinWindowHeight = 240
)

// The values of a concern whose file is missing or corrupt.
var (
	defaultGameplay = Gameplay{Rounds: 3, TimeLimitSeconds: 99}
	defaultAudio    = Audio{Master: 100, Music: 100, Effects: 100}
	defaultVideo    = Video{VSync: true, WindowWidth: 960, WindowHeight: 720}
)

func (g Gameplay) validate() error {
	if g.Rounds < 1 || g.Rounds%2 == 0 {
		return fmt.Errorf("rounds %d must be odd and at least 1", g.Rounds)
	}
	if g.TimeLimitSeconds < 0 {
		return fmt.Errorf("timeLimitSeconds %d must not be negative", g.TimeLimitSeconds)
	}
	return nil
}

func (a Audio) validate() error {
	for name, v := range map[string]int{"master": a.Master, "music": a.Music, "effects": a.Effects} {
		if v < 0 || v > 100 {
			return fmt.Errorf("%s %d must be between 0 and 100", name, v)
		}
	}
	return nil
}

func (v Video) validate() error {
	if v.WindowWidth < MinWindowWidth || v.WindowHeight < MinWindowHeight {
		return fmt.Errorf("window %dx%d must be at least %dx%d", v.WindowWidth, v.WindowHeight, MinWindowWidth, MinWindowHeight)
	}
	return nil
}

// Concern is one settings file.
type Concern int

// The concerns, in the order the Options screen lists them.
const (
	ConcernGameplay Concern = iota
	ConcernAudio
	ConcernVideo
	ConcernControls
	concernCount
)

var concernFiles = [concernCount]string{"gameplay.json", "audio.json", "video.json", "controls.json"}

// SettingsPlatform is the video.json section key of the running platform,
// from runtime.GOOS.
func SettingsPlatform(goos string) string {
	switch goos {
	case "windows", "android":
		return goos
	case "darwin":
		return "macos"
	}
	return "linux"
}

// WriteFunc stores a settings file: name is slash-separated, relative to the
// game folder. The host injects it; the core never touches the file system.
type WriteFunc func(name string, data []byte) error

type fileState int

const (
	fileMissing fileState = iota
	fileValid
	fileCorrupt
)

// Settings are the persisted settings of the running game. AudioGen, VideoGen
// and ControlsGen grow each time the audio, video or controls values change,
// so the main loop applies them without comparing values every frame.
type Settings struct {
	Gameplay                        Gameplay
	Audio                           Audio
	Video                           Video
	Controls                        Controls
	AudioGen, VideoGen, ControlsGen int
	WindowGen                       int

	platform string
	names    [concernCount]string                     // file names in the game folder
	raw      [concernCount]map[string]json.RawMessage // last read or written object, to keep unknown fields
	notice   [concernCount]string
	notices  []string
	write    WriteFunc
	warn     func(string)
}

// NewSettings returns the default settings, never persisted: the starting
// point of a Flow until the host loads the real ones.
func NewSettings() *Settings {
	return &Settings{Gameplay: defaultGameplay, Audio: defaultAudio, Video: defaultVideo, Controls: defaultControls}
}

// LoadSettings reads the settings files of fsys for platform (a
// SettingsPlatform key). A concern whose file is unreadable or invalid keeps
// its defaults, with a notice and a call of warn, and is left alone on disk
// until Save. A missing file means defaults, and is written with them through
// write now (first launch). write or warn may be nil.
func LoadSettings(fsys fs.FS, platform string, write WriteFunc, warn func(string)) *Settings {
	s := NewSettings()
	s.platform, s.write, s.warn = platform, write, warn
	dir := joinResolved(fsys, []string{"."}, "config")
	var states [concernCount]fileState
	for c := range concernCount {
		s.names[c] = joinResolved(fsys, []string{dir}, concernFiles[c])
		states[c] = s.load(fsys, c)
	}
	for c := range concernCount {
		if states[c] == fileMissing {
			s.Save(c)
		}
	}
	return s
}

// load reads concern c into s and reports the state of its file.
func (s *Settings) load(fsys fs.FS, c Concern) fileState {
	data, err := fs.ReadFile(fsys, s.names[c])
	if errors.Is(err, fs.ErrNotExist) {
		return fileMissing
	}
	if err == nil {
		err = s.decode(c, data)
	}
	if err != nil {
		s.setNotice(c, fmt.Sprintf("%s: defaults used", s.names[c]), fmt.Sprintf("%s ignored, defaults used: %v", s.names[c], err))
		return fileCorrupt
	}
	return fileValid
}

// decode applies the file's content to concern c; on an error s keeps its
// values.
func (s *Settings) decode(c Concern, data []byte) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	if obj == nil {
		return errors.New("not a JSON object")
	}
	// Kept even when a value is refused below: saving keeps its unknown keys
	// and other platforms' sections.
	s.raw[c] = obj
	switch c {
	case ConcernGameplay:
		g := defaultGameplay
		if err := json.Unmarshal(data, &g); err != nil {
			return err
		}
		if err := g.validate(); err != nil {
			return err
		}
		s.Gameplay = g
	case ConcernAudio:
		a := defaultAudio
		if err := json.Unmarshal(data, &a); err != nil {
			return err
		}
		if err := a.validate(); err != nil {
			return err
		}
		s.Audio = a
	case ConcernVideo:
		v := defaultVideo
		if section, ok := obj[s.platform]; ok {
			if err := json.Unmarshal(section, &v); err != nil {
				return fmt.Errorf("%s: %w", s.platform, err)
			}
		}
		if err := v.validate(); err != nil {
			return err
		}
		s.Video = v
	case ConcernControls:
		c := defaultControls
		if section, ok := obj[s.platform]; ok {
			if err := c.decode(section); err != nil {
				return fmt.Errorf("%s: %w", s.platform, err)
			}
		}
		s.Controls = c
	}
	return nil
}

// RejectControls replaces controls the host cannot use (a name SDL does not
// know) with the defaults, with the notice of an invalid file; the file is
// left as it is until Save.
func (s *Settings) RejectControls(err error) {
	s.Controls = defaultControls
	s.ControlsGen++
	name := s.names[ConcernControls]
	s.setNotice(ConcernControls, fmt.Sprintf("%s: defaults used", name), fmt.Sprintf("%s ignored, defaults used: %v", name, err))
}

// Notices are the problems the Options screen shows: concerns that fell back
// to defaults and files that could not be written.
func (s *Settings) Notices() []string { return s.notices }

// setNotice shows short on the Options screen and detail on stderr; short ""
// clears the concern's notice.
func (s *Settings) setNotice(c Concern, short, detail string) {
	s.notice[c] = short
	s.notices = s.notices[:0]
	for _, n := range s.notice {
		if n != "" {
			s.notices = append(s.notices, n)
		}
	}
	if detail != "" && s.warn != nil {
		s.warn(detail)
	}
}

// Save writes concern c's file with the current values, keeping the fields
// and platform sections it does not know. A failure is a notice; the values
// stay as they are. Without a writer nothing is persisted.
func (s *Settings) Save(c Concern) {
	if s.write == nil {
		return
	}
	obj, err := s.encode(c)
	var data []byte
	if err == nil {
		data, err = json.MarshalIndent(obj, "", "  ")
	}
	if err == nil {
		err = s.write(s.names[c], append(data, '\n'))
	}
	if err != nil {
		s.setNotice(c, fmt.Sprintf("%s: not saved", s.names[c]), fmt.Sprintf("cannot write %s: %v", s.names[c], err))
		return
	}
	s.raw[c] = obj
	s.setNotice(c, "", "")
}

// encode returns the object to write for concern c: what the file held, with
// the known keys overwritten.
func (s *Settings) encode(c Concern) (map[string]json.RawMessage, error) {
	obj := maps.Clone(s.raw[c])
	if obj == nil {
		obj = map[string]json.RawMessage{}
	}
	switch c {
	case ConcernGameplay:
		return obj, overlay(obj, s.Gameplay)
	case ConcernAudio:
		return obj, overlay(obj, s.Audio)
	case ConcernControls:
		// The section holds nothing but the known keys: decode refuses the others.
		data, err := json.Marshal(s.Controls.section())
		obj[s.platform] = data
		return obj, err
	}
	var section map[string]json.RawMessage
	if old, ok := obj[s.platform]; ok {
		// A section that is not an object is replaced.
		if err := json.Unmarshal(old, &section); err != nil {
			section = nil
		}
	}
	if section == nil {
		section = map[string]json.RawMessage{}
	}
	if err := overlay(section, s.Video); err != nil {
		return nil, err
	}
	data, err := json.Marshal(section)
	if err != nil {
		return nil, err
	}
	obj[s.platform] = data
	return obj, nil
}

// overlay sets the fields of v (its json keys) in obj.
func overlay(obj map[string]json.RawMessage, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	maps.Copy(obj, fields)
	return nil
}
