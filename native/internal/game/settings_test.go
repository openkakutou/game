package game

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// recorder is the writer a host injects: it keeps what was written.
type recorder struct {
	files map[string][]byte
	names []string
	err   error
}

func (r *recorder) write(name string, data []byte) error {
	if r.err != nil {
		return r.err
	}
	if r.files == nil {
		r.files = map[string][]byte{}
	}
	r.files[name] = data
	r.names = append(r.names, name)
	return nil
}

func decodeJSON(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("written file is not a JSON object: %v\n%s", err, data)
	}
	return out
}

func configFS(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for name, body := range files {
		fsys["config/"+name] = &fstest.MapFile{Data: []byte(body)}
	}
	return fsys
}

func TestSettings_MissingFilesGiveDefaultsAndAreWrittenOnFirstLaunch(t *testing.T) {
	rec := &recorder{}
	s := LoadSettings(fstest.MapFS{}, "linux", rec.write, nil)
	if s.Gameplay != (Gameplay{Rounds: 3, TimeLimitSeconds: 99}) {
		t.Errorf("gameplay = %+v", s.Gameplay)
	}
	if s.Audio != (Audio{Master: 100, Music: 100, Effects: 100}) {
		t.Errorf("audio = %+v", s.Audio)
	}
	if s.Video != (Video{Fullscreen: false, VSync: true, WindowWidth: 960, WindowHeight: 720}) {
		t.Errorf("video = %+v", s.Video)
	}
	if len(s.Notices()) != 0 {
		t.Errorf("a missing file is silent, got notices %q", s.Notices())
	}
	got := slices.Clone(rec.names)
	slices.Sort(got)
	if want := []string{"config/appearance.json", "config/audio.json", "config/controls.json", "config/gameplay.json", "config/video.json"}; !slices.Equal(got, want) {
		t.Fatalf("written = %q, want %q", got, want)
	}
	if want := map[string]any{"rounds": 3.0, "timeLimitSeconds": 99.0}; !reflect.DeepEqual(decodeJSON(t, rec.files["config/gameplay.json"]), want) {
		t.Errorf("gameplay.json = %s", rec.files["config/gameplay.json"])
	}
	wantVideo := map[string]any{"linux": map[string]any{"fullscreen": false, "vsync": true, "windowWidth": 960.0, "windowHeight": 720.0}}
	if !reflect.DeepEqual(decodeJSON(t, rec.files["config/video.json"]), wantVideo) {
		t.Errorf("video.json = %s", rec.files["config/video.json"])
	}
}

func TestSettings_OnlyMissingFilesAreWrittenOnFirstLaunch(t *testing.T) {
	rec := &recorder{}
	fsys := configFS(map[string]string{"audio.json": `{"master": 50, "music": 60, "effects": 70}`})
	s := LoadSettings(fsys, "linux", rec.write, nil)
	if s.Audio != (Audio{Master: 50, Music: 60, Effects: 70}) {
		t.Errorf("audio = %+v", s.Audio)
	}
	got := slices.Clone(rec.names)
	slices.Sort(got)
	if want := []string{"config/appearance.json", "config/controls.json", "config/gameplay.json", "config/video.json"}; !slices.Equal(got, want) {
		t.Errorf("written = %q, want %q", got, want)
	}
}

func TestSettings_ACorruptConcernFallsBackAloneAndIsNotRewritten(t *testing.T) {
	rec := &recorder{}
	var warned []string
	fsys := configFS(map[string]string{
		"gameplay.json": `{"rounds": 5,`,
		"audio.json":    `{"master": 50, "music": 60, "effects": 70}`,
		"video.json":    `{}`,
		"controls.json": `{}`,
	})
	s := LoadSettings(fsys, "linux", rec.write, func(m string) { warned = append(warned, m) })
	if s.Gameplay != (Gameplay{Rounds: 3, TimeLimitSeconds: 99}) {
		t.Errorf("gameplay = %+v, want the defaults", s.Gameplay)
	}
	if s.Audio != (Audio{Master: 50, Music: 60, Effects: 70}) {
		t.Errorf("audio = %+v, want its own values", s.Audio)
	}
	if n := s.Notices(); len(n) != 1 || !strings.Contains(n[0], "config/gameplay.json") {
		t.Errorf("notices = %q, want one naming config/gameplay.json", n)
	}
	if len(warned) != 1 || !strings.Contains(warned[0], "config/gameplay.json") {
		t.Errorf("stderr warnings = %q", warned)
	}
	if !slices.Equal(rec.names, []string{"config/appearance.json"}) {
		t.Errorf("startup wrote %q, want only the missing appearance.json: a corrupt file must be left alone", rec.names)
	}
	s.Gameplay.Rounds = 1
	s.Save(ConcernGameplay)
	if got := decodeJSON(t, rec.files["config/gameplay.json"]); got["rounds"] != 1.0 || got["timeLimitSeconds"] != 99.0 {
		t.Errorf("saved gameplay.json = %v", got)
	}
	if len(s.Notices()) != 0 {
		t.Errorf("notices = %q after the concern was saved", s.Notices())
	}
}

func TestSettings_InvalidValuesMakeTheConcernCorrupt(t *testing.T) {
	cases := []struct{ file, body string }{
		{"gameplay.json", `{"rounds": 2, "timeLimitSeconds": 99}`},
		{"gameplay.json", `{"rounds": 0, "timeLimitSeconds": 99}`},
		{"gameplay.json", `{"rounds": 3, "timeLimitSeconds": -1}`},
		{"audio.json", `{"master": 120, "music": 100, "effects": 100}`},
		{"audio.json", `{"master": 100, "music": -5, "effects": 100}`},
		{"video.json", `{"linux": {"windowWidth": 300, "windowHeight": 720}}`},
		{"video.json", `{"linux": {"windowWidth": 960, "windowHeight": 200}}`},
		{"video.json", `{"linux": {"fullscreen": "yes"}}`},
		{"video.json", `[1]`},
	}
	for _, c := range cases {
		s := LoadSettings(configFS(map[string]string{c.file: c.body}), "linux", nil, nil)
		if n := s.Notices(); len(n) != 1 || !strings.Contains(n[0], "config/"+c.file) {
			t.Errorf("%s %s: notices = %q", c.file, c.body, n)
		}
		if s.Gameplay != (Gameplay{3, 99}) || s.Audio != (Audio{100, 100, 100}) || s.Video != (Video{false, true, 960, 720}) {
			t.Errorf("%s %s: values = %+v %+v %+v, want defaults", c.file, c.body, s.Gameplay, s.Audio, s.Video)
		}
	}
}

const videoFile = `{
	"theme": "dark",
	"windows": {"fullscreen": true, "vsync": false, "windowWidth": 1280, "windowHeight": 960},
	"linux": {"fullscreen": false, "vsync": true, "windowWidth": 800, "windowHeight": 600, "extra": 1}
}`

func TestSettings_VideoReadsOnlyTheRunningPlatformsSection(t *testing.T) {
	fsys := configFS(map[string]string{"video.json": videoFile})
	if v := LoadSettings(fsys, "linux", nil, nil).Video; v != (Video{false, true, 800, 600}) {
		t.Errorf("linux video = %+v", v)
	}
	if v := LoadSettings(fsys, "windows", nil, nil).Video; v != (Video{true, false, 1280, 960}) {
		t.Errorf("windows video = %+v", v)
	}
	s := LoadSettings(fsys, "macos", nil, nil)
	if s.Video != (Video{false, true, 960, 720}) || len(s.Notices()) != 0 {
		t.Errorf("macos video = %+v notices %q, want silent defaults for a platform without a section", s.Video, s.Notices())
	}
}

func TestSettings_SavingVideoKeepsOtherSectionsAndUnknownFields(t *testing.T) {
	rec := &recorder{}
	s := LoadSettings(configFS(map[string]string{"video.json": videoFile}), "linux", rec.write, nil)
	s.Video.WindowWidth = 1024
	s.Save(ConcernVideo)
	got := decodeJSON(t, rec.files["config/video.json"])
	if got["theme"] != "dark" {
		t.Errorf("theme = %v, want it kept", got["theme"])
	}
	wantWindows := map[string]any{"fullscreen": true, "vsync": false, "windowWidth": 1280.0, "windowHeight": 960.0}
	if !reflect.DeepEqual(got["windows"], wantWindows) {
		t.Errorf("windows section = %v, want it untouched", got["windows"])
	}
	wantLinux := map[string]any{"fullscreen": false, "vsync": true, "windowWidth": 1024.0, "windowHeight": 600.0, "extra": 1.0}
	if !reflect.DeepEqual(got["linux"], wantLinux) {
		t.Errorf("linux section = %v, want %v", got["linux"], wantLinux)
	}
}

func TestSettings_SavingGameplayKeepsUnknownFields(t *testing.T) {
	rec := &recorder{}
	fsys := configFS(map[string]string{"gameplay.json": `{"rounds": 5, "timeLimitSeconds": 30, "note": "hi"}`})
	s := LoadSettings(fsys, "linux", rec.write, nil)
	if s.Gameplay != (Gameplay{5, 30}) {
		t.Fatalf("gameplay = %+v", s.Gameplay)
	}
	s.Gameplay.Rounds = 1
	s.Save(ConcernGameplay)
	want := map[string]any{"rounds": 1.0, "timeLimitSeconds": 30.0, "note": "hi"}
	if got := decodeJSON(t, rec.files["config/gameplay.json"]); !reflect.DeepEqual(got, want) {
		t.Errorf("gameplay.json = %v, want %v", got, want)
	}
}

func TestSettings_AWriteFailureIsANoticeAndAWarningNotACrash(t *testing.T) {
	rec := &recorder{}
	var warned []string
	s := LoadSettings(configFS(map[string]string{"audio.json": `{}`}), "linux", rec.write, func(m string) { warned = append(warned, m) })
	rec.err = errors.New("disk full")
	s.Audio.Music = 40
	s.Save(ConcernAudio)
	if n := s.Notices(); len(n) != 1 || !strings.Contains(n[0], "config/audio.json") {
		t.Errorf("notices = %q, want one naming config/audio.json", n)
	}
	if len(warned) != 1 || !strings.Contains(warned[0], "disk full") {
		t.Errorf("warnings = %q, want the error on stderr", warned)
	}
	if s.Audio.Music != 40 {
		t.Errorf("music = %d, the running value must stay", s.Audio.Music)
	}
	rec.err = nil
	s.Save(ConcernAudio)
	if len(s.Notices()) != 0 {
		t.Errorf("notices = %q after a successful save", s.Notices())
	}
}

func TestSettingsPlatform_MapsGOOSToTheVideoSectionKey(t *testing.T) {
	for goos, want := range map[string]string{"windows": "windows", "darwin": "macos", "linux": "linux", "android": "android", "freebsd": "linux"} {
		if got := SettingsPlatform(goos); got != want {
			t.Errorf("SettingsPlatform(%q) = %q, want %q", goos, got, want)
		}
	}
}

func TestSettings_AnInvalidSectionOfThisPlatformKeepsTheRestOfTheFileOnSave(t *testing.T) {
	rec := &recorder{}
	body := `{"theme": "dark", "windows": {"fullscreen": true, "vsync": false, "windowWidth": 1280, "windowHeight": 960}, "linux": {"windowWidth": 100}}`
	s := LoadSettings(configFS(map[string]string{"video.json": body}), "linux", rec.write, nil)
	if len(s.Notices()) != 1 || s.Video != (Video{false, true, 960, 720}) {
		t.Fatalf("notices = %q video = %+v", s.Notices(), s.Video)
	}
	s.Video.VSync = false
	s.Save(ConcernVideo)
	got := decodeJSON(t, rec.files["config/video.json"])
	wantWindows := map[string]any{"fullscreen": true, "vsync": false, "windowWidth": 1280.0, "windowHeight": 960.0}
	if got["theme"] != "dark" || !reflect.DeepEqual(got["windows"], wantWindows) {
		t.Errorf("video.json = %v, want the windows section and theme kept", got)
	}
	want := map[string]any{"fullscreen": false, "vsync": false, "windowWidth": 960.0, "windowHeight": 720.0}
	if !reflect.DeepEqual(got["linux"], want) {
		t.Errorf("linux section = %v, want %v", got["linux"], want)
	}
}

func TestSettings_AnInvalidValueKeepsUnknownKeysOfTheObjectOnSave(t *testing.T) {
	rec := &recorder{}
	s := LoadSettings(configFS(map[string]string{"gameplay.json": `{"rounds": 2, "note": "hi"}`}), "linux", rec.write, nil)
	s.Save(ConcernGameplay)
	want := map[string]any{"rounds": 3.0, "timeLimitSeconds": 99.0, "note": "hi"}
	if got := decodeJSON(t, rec.files["config/gameplay.json"]); !reflect.DeepEqual(got, want) {
		t.Errorf("gameplay.json = %v, want %v", got, want)
	}
}

func TestSettings_WindowGenGrowsOnlyWhenTheWindowSizeChanges(t *testing.T) {
	f, _ := optionsFlow(t)
	navAll(f, NavDown, NavConfirm, NavDown, NavDown, NavConfirm) // Options > Video
	gen := f.Settings.WindowGen
	navAll(f, NavRight, NavDown, NavLeft) // fullscreen, vsync
	if f.Settings.WindowGen != gen {
		t.Errorf("WindowGen = %d, want %d: no size changed", f.Settings.WindowGen, gen)
	}
	navAll(f, NavDown, NavRight)
	if f.Settings.WindowGen == gen {
		t.Error("WindowGen must grow when the window size changes")
	}
}

func TestOptions_SaveOpenConcernSavesAChangedConcernOnce(t *testing.T) {
	f, rec := optionsFlow(t)
	toAudio(t, f)
	navAll(f, NavDown, NavLeft)
	f.SaveOpenConcern()
	if len(rec.names) != 1 || rec.names[0] != "config/audio.json" {
		t.Fatalf("written = %q", rec.names)
	}
	if got := decodeJSON(t, rec.files["config/audio.json"]); got["music"] != 90.0 {
		t.Errorf("audio.json = %v", got)
	}
}

func TestOptions_SaveOpenConcernWritesNothingWhenUnchangedOrNotOnAConcernScreen(t *testing.T) {
	f, rec := optionsFlow(t)
	f.SaveOpenConcern() // main menu
	toAudio(t, f)
	f.SaveOpenConcern() // unchanged
	if len(rec.names) != 0 {
		t.Errorf("written = %q, want nothing", rec.names)
	}
}

func TestSettings_AppearanceDefaultsToTheDefaultPackAndLifebarAndIsWrittenOnFirstLaunch(t *testing.T) {
	rec := &recorder{}
	s := LoadSettings(fstest.MapFS{}, "linux", rec.write, nil)
	if s.Appearance != (Appearance{Screenpack: "default", Lifebar: "default"}) {
		t.Errorf("appearance = %+v", s.Appearance)
	}
	want := map[string]any{"screenpack": "default", "lifebar": "default"}
	if !reflect.DeepEqual(decodeJSON(t, rec.files["config/appearance.json"]), want) {
		t.Errorf("appearance.json = %s", rec.files["config/appearance.json"])
	}
}

func TestSettings_AppearanceReadsBothNamesAndAMissingOrEmptyOneIsDefault(t *testing.T) {
	cases := []struct {
		body string
		want Appearance
	}{
		{`{"screenpack": "alt", "lifebar": "VHD"}`, Appearance{"alt", "VHD"}},
		{`{"lifebar": "VHD"}`, Appearance{"default", "VHD"}},
		{`{"screenpack": "", "lifebar": "x"}`, Appearance{"default", "x"}},
	}
	for _, c := range cases {
		s := LoadSettings(configFS(map[string]string{"appearance.json": c.body}), "linux", nil, nil)
		if s.Appearance != c.want || len(s.Notices()) != 0 {
			t.Errorf("%s: appearance = %+v notices = %q, want %+v and none", c.body, s.Appearance, s.Notices(), c.want)
		}
	}
}

func TestSettings_ACorruptAppearanceFallsBackAloneWithANoticeAndIsNotRewritten(t *testing.T) {
	rec := &recorder{}
	var warned []string
	fsys := configFS(map[string]string{
		"appearance.json": `{"screenpack": 5}`,
		"audio.json":      `{"master": 50, "music": 60, "effects": 70}`,
	})
	s := LoadSettings(fsys, "linux", rec.write, func(m string) { warned = append(warned, m) })
	if s.Appearance != (Appearance{"default", "default"}) {
		t.Errorf("appearance = %+v, want the defaults", s.Appearance)
	}
	if s.Audio != (Audio{50, 60, 70}) {
		t.Errorf("audio = %+v, want its own values", s.Audio)
	}
	if n := s.Notices(); len(n) != 1 || !strings.Contains(n[0], "config/appearance.json") {
		t.Errorf("notices = %q, want one naming config/appearance.json", n)
	}
	if len(warned) != 1 || slices.Contains(rec.names, "config/appearance.json") {
		t.Errorf("warnings = %q written = %q, want one warning and no rewrite", warned, rec.names)
	}
}

func TestSettings_SavingAppearanceKeepsUnknownKeys(t *testing.T) {
	rec := &recorder{}
	fsys := configFS(map[string]string{"appearance.json": `{"screenpack": "alt", "theme": {"x": 1}}`})
	s := LoadSettings(fsys, "linux", rec.write, nil)
	rec.names = nil
	s.Appearance.Lifebar = "VHD"
	s.Save(ConcernAppearance)
	want := map[string]any{"screenpack": "alt", "lifebar": "VHD", "theme": map[string]any{"x": 1.0}}
	if got := decodeJSON(t, rec.files["config/appearance.json"]); !reflect.DeepEqual(got, want) {
		t.Errorf("saved appearance.json = %v, want %v", got, want)
	}
}

func TestSettings_AppearanceNoticeIsRebuiltFromTheScreenpackAndLifebarPartsAndClearedWhenBothAreEmpty(t *testing.T) {
	var warned []string
	s := LoadSettings(fstest.MapFS{}, "linux", nil, func(m string) { warned = append(warned, m) })
	s.NoteScreenpack([]string{"screenpack \"gone\" not found, using \"default\"\ndetail"})
	s.NoteLifebar([]string{"lifebar \"x\" not found: \"default\" used"})
	want := "screenpack \"gone\" not found, using \"default\"; lifebar \"x\" not found: \"default\" used"
	if n := s.Notices(); len(n) != 1 || n[0] != want {
		t.Errorf("notices = %q, want [%s]", n, want)
	}
	if len(warned) != 2 {
		t.Errorf("warnings = %q, want one per fallback", warned)
	}
	s.NoteLifebar(nil)
	if n := s.Notices(); len(n) != 1 || n[0] != "screenpack \"gone\" not found, using \"default\"" {
		t.Errorf("after the lifebar notice went: %q, want the screenpack one alone", n)
	}
	s.NoteScreenpack(nil)
	if len(s.Notices()) != 0 {
		t.Errorf("notices = %q, want none", s.Notices())
	}
}
