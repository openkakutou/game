package ui_test

import (
	"testing"
	"testing/fstest"

	"github.com/openkakutou/game/native/internal/ui"
	"github.com/openkakutou/screenpack"
)

const soundRoot = `{
  "formatVersion": 1, "name": "Test", "author": "Test", "version": "1.0.0",
  "fonts": { "body": "fonts/NotoSans-Regular.ttf" },
  "textStyles": { "default": { "font": "body", "size": 5, "color": "#ffffff" } },
  "sounds": { "move": "root-move.ogg", "confirm": "root-confirm.ogg", "cancel": "root-cancel.ogg", "error": "root-error.ogg" },
  "music": { "file": "root-music.ogg", "volume": 0.5, "loopStartMs": 1200 }
}`

func soundFiles() fstest.MapFS {
	files := fstest.MapFS{"screenpack.json": {Data: []byte(soundRoot)}}
	for _, name := range []string{"root-move", "root-confirm", "root-cancel", "root-error", "root-music", "screen-move", "menu-confirm", "pause-music"} {
		files[name+".ogg"] = &fstest.MapFile{Data: []byte("ogg")}
	}
	return files
}

func soundPack(t *testing.T, screens map[string]string) *ui.Pack {
	t.Helper()
	files := soundFiles()
	files["screenpack.json"] = &fstest.MapFile{Data: []byte(soundRoot)}
	game := screenGame(t, "title", screens["title"], files)
	for name, parts := range screens {
		if name != "title" {
			game["screenpacks/default/screens/"+name+".json"] = &fstest.MapFile{Data: []byte(`{"formatVersion": 1, ` + parts + `}`)}
		}
	}
	pack, notices := ui.LoadPack(game)
	if len(notices) > 0 {
		t.Fatalf("pack refused: %v", notices)
	}
	return pack
}

func TestPack_SoundsComeFromTheMenuThenTheScreenThenTheRootFile(t *testing.T) {
	pack := soundPack(t, map[string]string{
		"title": `"sounds": { "move": "screen-move.ogg" },
		          "menu": { "style": "default", "spacing": 5, "sounds": { "confirm": "menu-confirm.ogg" } }`,
	})
	cases := []struct {
		screen, name, want string
	}{
		{"title", "confirm", "menu-confirm.ogg"}, // the menu's own
		{"title", "move", "screen-move.ogg"},     // the screen's
		{"title", "cancel", "root-cancel.ogg"},   // the root file's
		{"", "error", "root-error.ogg"},          // no screen: the root file's
	}
	for _, c := range cases {
		got, ok := pack.Sound(c.screen, c.name)
		if !ok || got.Path != c.want || got.Source != screenpack.FromPack {
			t.Errorf("%s/%s = %+v ok=%v, want %s from the pack", c.screen, c.name, got, ok, c.want)
		}
	}
	if _, ok := pack.Sound("title", "tick"); ok {
		t.Errorf("a sound no file names must be absent")
	}
}

func TestPack_ASoundOfAPartTheScreenLeavesOutComesFromTheDefaultPack(t *testing.T) {
	// The pack's options screen has no menu: that part, and its sounds, are the default's.
	pack := soundPack(t, map[string]string{"title": ``, "options": `"sounds": { "move": "screen-move.ogg" }`})
	got, ok := pack.Sound("options", "move")
	if !ok || got.Path != "screen-move.ogg" || got.Source != screenpack.FromPack {
		t.Errorf("options move = %+v ok=%v, want the screen's own", got, ok)
	}
}

func TestPack_MusicIsTheScreensThenTheRootsAndNullIsSilence(t *testing.T) {
	pack := soundPack(t, map[string]string{
		"title":      ``, // leaves music out: the root file's
		"matchSetup": ``,
		"pause":      `"music": { "file": "pause-music.ogg" }`, // its own, full volume
		"options":    `"music": null`,                          // silence
	})
	root, ok := pack.Music("title")
	if !ok || root.Asset.Path != "root-music.ogg" || root.Volume != 0.5 || root.LoopStartMs != 1200 {
		t.Errorf("title music = %+v ok=%v, want the root music at 0.5 looping from 1200 ms", root, ok)
	}
	own, ok := pack.Music("pause")
	if !ok || own.Asset.Path != "pause-music.ogg" || own.Volume != 1 || own.LoopStartMs != 0 {
		t.Errorf("pause music = %+v ok=%v, want its own at full volume", own, ok)
	}
	if _, ok := pack.Music("options"); ok {
		t.Errorf("a screen with music null must be silent")
	}
	// Two screens naming the same file share its identity, so the player keeps it playing.
	if again, _ := pack.Music("matchSetup"); again.Asset != root.Asset {
		t.Errorf("a screen leaving music out has asset %+v, want the root's %+v", again.Asset, root.Asset)
	}
}

func TestPack_TheBuiltInPackPlaysItsThemeOnTheMenusAndIsSilentOnPause(t *testing.T) {
	pack, _ := ui.LoadPack(fstest.MapFS{})
	theme, ok := pack.Music("title")
	if !ok || theme.Asset.Path != "sounds/theme.ogg" || theme.Volume != 0.6 {
		t.Errorf("title music = %+v ok=%v, want sounds/theme.ogg at 0.6", theme, ok)
	}
	if _, ok := pack.Music("pause"); ok {
		t.Errorf("the built-in pause screen has music null")
	}
	move, ok := pack.Sound("title", "move")
	if !ok || move.Path != "sounds/move.ogg" {
		t.Errorf("move = %+v ok=%v, want sounds/move.ogg", move, ok)
	}
}

func TestScreens_AFadeInMultipliesTheWholeScreenByItsEasedProgress(t *testing.T) {
	parts := `"background": [{ "type": "rect", "size": [10, 10], "color": "#ff0000" }],
	  "transition": { "in": { "kind": "fade", "ms": 400, "easing": "linear" } }`
	l := layoutOf(t, gameWith(t, parts, nil))
	alphaAt := func(ms int) float32 {
		var list ui.DrawList
		l.Title(&list, view720, ui.TitleState{Phase: ui.PhaseMenu, ElapsedMs: ms})
		return list.Ops[0].Color.A
	}
	near(t, "at 0 ms", alphaAt(0), 0)
	near(t, "at 200 ms", alphaAt(200), 0.5)
	near(t, "at 400 ms", alphaAt(400), 1)
	near(t, "long after", alphaAt(9000), 1)
}

func TestScreens_AFadeInUsesTheDefaultEaseOutAndMultipliesTheElementsOwnOpacity(t *testing.T) {
	// easeOut at half the time: 1-(1-0.5)^2 = 0.75; the rect's own opacity 0.5 on top.
	parts := `"background": [{ "type": "rect", "size": [10, 10], "color": "#ff0000", "opacity": 0.5 }],
	  "transition": { "in": { "kind": "fade", "ms": 400 } }`
	l := layoutOf(t, gameWith(t, parts, nil))
	var list ui.DrawList
	l.Title(&list, view720, ui.TitleState{Phase: ui.PhaseMenu, ElapsedMs: 200})
	near(t, "alpha", list.Ops[0].Color.A, 0.375)
}

func TestScreens_AFadeInReachesTextAndImagesOfTheScreenToo(t *testing.T) {
	pack, _ := ui.LoadPack(fstest.MapFS{})
	l := ui.NewLayout(pack)
	var list ui.DrawList
	// The built-in title fades in over 250 ms, easeOut: at 125 ms, 1-(0.5)^2 = 0.75.
	l.Title(&list, view720, ui.TitleState{Phase: ui.PhaseMenu, Entries: []string{"Versus"}, ElapsedMs: 125})
	for i, op := range list.Ops {
		if op.Kind == ui.OpRect && op.Color.A > 0.76 {
			t.Errorf("op %d (%+v) is not faded", i, op)
		}
	}
	bg := imageOps(&list, "images/background.png")[0]
	near(t, "background alpha", bg.Color.A, 0.75)
	grid := imageOps(&list, "images/grid.png")[0]
	near(t, "grid alpha", grid.Color.A, 0.6*0.75)
	for _, op := range list.Ops {
		if op.Kind == ui.OpText && op.Color.A > 0.76 {
			t.Errorf("a text run is not faded: %+v", op)
		}
	}
}

func TestScreens_OtherTransitionKindsAreNotAppliedToTheWholeScreen(t *testing.T) {
	parts := `"background": [{ "type": "rect", "size": [10, 10], "color": "#ff0000" }],
	  "transition": { "in": { "kind": "slide", "from": "left", "ms": 400 } }`
	l := layoutOf(t, gameWith(t, parts, nil))
	var list ui.DrawList
	l.Title(&list, view720, ui.TitleState{Phase: ui.PhaseMenu, ElapsedMs: 0})
	near(t, "alpha", list.Ops[0].Color.A, 1)
}
