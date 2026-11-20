package audio

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/openkakutou/game/native/internal/game"
	"github.com/openkakutou/game/native/internal/mode"
	"github.com/openkakutou/game/native/internal/ui"
	"github.com/openkakutou/screenpack"
	"github.com/veandco/go-sdl2/mix"
)

// titleFlow is a flow over the fixture content, on the title screen.
func titleFlow(t *testing.T) *game.Flow {
	t.Helper()
	fsys := os.DirFS(content)
	chars, _ := game.DiscoverCharacters(fsys, "chars")
	stages, _ := game.DiscoverStages(fsys, "stages")
	builtins, err := mode.BuiltIn()
	if err != nil {
		t.Fatal(err)
	}
	versus := builtins[2] // versus.jsonc, third in file-name order (arcade, training, versus)
	return game.NewFlow(fsys, chars, stages, game.DefaultLoader(fsys), []mode.Mode{versus})
}

// menuPlayer is a Player with the built-in screenpack, recording what it logs.
func menuPlayer(t *testing.T, pack *ui.Pack) (p *Player, warns, infos *[]string) {
	t.Helper()
	p, warns = newPlayer(t)
	infos = &[]string{}
	p.infof = func(format string, a ...any) { *infos = append(*infos, fmt.Sprintf(format, a...)) }
	p.SetPack(pack)
	p.SetVolumes(game.Audio{Master: 100, Music: 100, Effects: 100})
	return p, warns, infos
}

func builtInPack() *ui.Pack {
	pack, _ := ui.LoadPack(fstest.MapFS{}, "default")
	return pack
}

// packWith is the built-in screenpack copied into the game folder, with the
// files of edit changed (name -> new content from the old one).
func packWith(t *testing.T, edit map[string]func([]byte) []byte) *ui.Pack {
	t.Helper()
	game := fstest.MapFS{}
	root := screenpack.DefaultFS()
	err := fs.WalkDir(root, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(root, name)
		if err != nil {
			return err
		}
		if f, ok := edit[name]; ok {
			data = f(data)
		}
		game["screenpacks/default/"+name] = &fstest.MapFile{Data: data}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	pack, notices := ui.LoadPack(game, "default")
	if len(notices) > 0 {
		t.Fatalf("pack refused: %v", notices)
	}
	return pack
}

func TestUpdate_TheTitlePlaysTheBuiltInThemeAndEveryMenuScreenNamingItKeepsItPlaying(t *testing.T) {
	p, warns, infos := menuPlayer(t, builtInPack())
	f := titleFlow(t)
	p.Update(f)
	if p.music == nil || mix.PlayingMusic() != true {
		t.Fatalf("no music on the title: music=%v playing=%v warnings=%v", p.music, mix.PlayingMusic(), *warns)
	}
	started := p.music
	if len(*infos) != 1 || !strings.HasSuffix((*infos)[0], "sounds/theme.ogg") {
		t.Errorf("infos = %q, want one saying sounds/theme.ogg loaded", *infos)
	}
	f.Navigate(game.NavConfirm) // main menu
	f.Navigate(game.NavConfirm) // character select
	p.Update(f)
	themes := 0
	for _, i := range *infos {
		if strings.HasSuffix(i, "sounds/theme.ogg") {
			themes++
		}
	}
	if p.music != started || themes != 1 {
		t.Errorf("the theme was reloaded across screens naming the same file (infos %q)", *infos)
	}
	if len(*warns) != 0 {
		t.Errorf("warnings: %v", *warns)
	}
}

func TestUpdate_TheMenuMusicVolumeIsTheMasterTimesMusicTimesThePacksVolume(t *testing.T) {
	p, _, _ := menuPlayer(t, builtInPack())
	p.SetVolumes(game.Audio{Master: 100, Music: 50, Effects: 100})
	p.Update(titleFlow(t))
	// 128 x 100% x 50% = 64, times the built-in theme's volume 0.6 = 38.4.
	if got := mix.VolumeMusic(-1); got != 38 {
		t.Errorf("menu music volume = %d, want 38", got)
	}
}

func TestUpdate_AMatchStopsTheMenuMusicForTheStageMusicAndTheNextMenuResumesIt(t *testing.T) {
	p, _, _ := menuPlayer(t, builtInPack())
	title := titleFlow(t)
	p.Update(title)
	menuTrack := p.music
	if !p.menuPlaying {
		t.Fatal("the menu music is not playing")
	}
	m := matchFlow(t)
	p.Update(m)
	if p.menuPlaying || p.owner == nil || p.music == nil || p.music == menuTrack {
		t.Errorf("in a match: menuPlaying=%v owner=%v music=%v, want the stage music alone", p.menuPlaying, p.owner, p.music)
	}
	if got := mix.VolumeMusic(-1); got != 128 {
		t.Errorf("stage music volume = %d, want the full 128 (the menu's own volume must not stick)", got)
	}
	m.Match, m.Screen = nil, game.ScreenMainMenu
	p.Update(m)
	if !p.menuPlaying || p.owner != nil || !mix.PlayingMusic() {
		t.Errorf("back on a menu: menuPlaying=%v owner=%v playing=%v, want the theme again", p.menuPlaying, p.owner, mix.PlayingMusic())
	}
}

func TestUpdate_AScreenWithMusicNullIsSilentAndNoMusicKeepsTheMenusSilent(t *testing.T) {
	silent := packWith(t, map[string]func([]byte) []byte{
		"screens/title.json": func(b []byte) []byte {
			return []byte(strings.Replace(string(b), `"formatVersion": 1,`, `"formatVersion": 1, "music": null,`, 1))
		},
	})
	p, warns, _ := menuPlayer(t, silent)
	f := titleFlow(t)
	p.Update(f)
	if mix.PlayingMusic() || len(*warns) != 0 {
		t.Errorf("music null: playing=%v warnings=%v, want silence", mix.PlayingMusic(), *warns)
	}
	f.Navigate(game.NavConfirm)
	f.Navigate(game.NavConfirm) // the select leaves music out: the root theme
	p.Update(f)
	if !mix.PlayingMusic() {
		t.Errorf("the select screen should play the root theme")
	}
	f.Navigate(game.NavCancel) // back to the main menu: the title screen again, silent
	p.Update(f)
	if mix.PlayingMusic() {
		t.Errorf("going back to the silent title must stop the theme")
	}

	quiet, _, _ := menuPlayer(t, builtInPack())
	quiet.NoMusic = true
	quiet.Update(titleFlow(t))
	if mix.PlayingMusic() {
		t.Errorf("-nomusic must keep the menu music off")
	}
}

func TestUpdate_AMenuCuePlaysThePacksSoundOnceLoadedAndNotReloaded(t *testing.T) {
	p, warns, infos := menuPlayer(t, builtInPack())
	f := titleFlow(t)
	p.Update(f) // the theme
	*infos = nil
	f.Navigate(game.NavConfirm) // title -> main menu: confirm
	p.Update(f)
	if mix.Playing(-1) != 1 {
		t.Fatalf("playing channels = %d, want the confirm sound on one (warnings %v)", mix.Playing(-1), *warns)
	}
	if len(*infos) != 1 || !strings.HasSuffix((*infos)[0], "sounds/confirm.ogg") {
		t.Errorf("infos = %q, want sounds/confirm.ogg loaded", *infos)
	}
	f.Navigate(game.NavDown) // move
	p.Update(f)
	f.Navigate(game.NavUp)
	f.Navigate(game.NavDown)
	p.Update(f)
	loads := 0
	for _, i := range *infos {
		if strings.Contains(i, "move.ogg") {
			loads++
		}
	}
	if loads != 1 {
		t.Errorf("move.ogg loaded %d times, want once (infos %q)", loads, *infos)
	}
	if len(*warns) != 0 {
		t.Errorf("warnings: %v", *warns)
	}
}

func TestUpdate_ASoundFileThatCannotBeDecodedIsWarnedOnceAndSkipped(t *testing.T) {
	broken := packWith(t, map[string]func([]byte) []byte{
		"sounds/confirm.ogg": func([]byte) []byte { return []byte("not an ogg file") },
	})
	p, warns, _ := menuPlayer(t, broken)
	f := titleFlow(t)
	f.Navigate(game.NavConfirm)
	p.Update(f)
	f.Navigate(game.NavCancel)
	f.Navigate(game.NavConfirm)
	p.Update(f)
	if len(*warns) != 1 || !strings.Contains((*warns)[0], "sounds/confirm.ogg") {
		t.Errorf("warnings = %q, want exactly one naming sounds/confirm.ogg", *warns)
	}
}

func TestUpdate_WithoutAPackTheMenuCuesAreDrainedAndNothingPlays(t *testing.T) {
	p, _ := newPlayer(t)
	f := titleFlow(t)
	f.Navigate(game.NavConfirm)
	p.Update(f)
	if got := f.DrainMenuCues(nil); len(got) != 0 {
		t.Errorf("cues left: %+v", got)
	}
	if mix.Playing(-1) != 0 || mix.PlayingMusic() {
		t.Errorf("something plays without a pack")
	}
}

func TestUpdate_OnAMenuWithNothingNewDoesNotAllocate(t *testing.T) {
	p, _, _ := menuPlayer(t, builtInPack())
	f := titleFlow(t)
	p.Update(f)
	if allocs := testing.AllocsPerRun(100, func() { p.Update(f) }); allocs != 0 {
		t.Errorf("%v allocations per frame, want 0", allocs)
	}
}

func TestUpdate_AMenuSoundThatStartsOrLeavesAMatchIsStillPlayingAfterTheOwnerChange(t *testing.T) {
	p, warns, _ := menuPlayer(t, builtInPack())
	f := matchFlow(t)
	p.Update(f)
	tickUntil(t, f, func() bool { return f.Screen == game.ScreenResult })
	p.Update(f)
	mix.HaltChannel(-1)         // the sounds of the walk to the match are over
	f.Navigate(game.NavConfirm) // Rematch: a new match takes the music over
	p.Update(f)
	if p.owner != f.Match || mix.Playing(-1) != 1 {
		t.Errorf("rematch: owner ok=%v playing=%d, want the confirm sound on one channel (warnings %v)", p.owner == f.Match, mix.Playing(-1), *warns)
	}

	mix.HaltChannel(-1)
	f.Navigate(game.NavPause)
	p.Update(f)
	f.Navigate(game.NavDown)
	f.Navigate(game.NavDown)
	f.Navigate(game.NavConfirm) // Character Change: leaves the match
	p.Update(f)
	if p.owner != nil || mix.Playing(-1) < 1 {
		t.Errorf("leaving the match: owner=%v playing=%d, want the confirm sound still playing", p.owner, mix.Playing(-1))
	}
}

func TestUpdate_AMenuMusicThatCannotBeLoadedIsWarnedOnceNotEveryFrame(t *testing.T) {
	broken := packWith(t, map[string]func([]byte) []byte{
		"sounds/theme.ogg": func([]byte) []byte { return []byte("not an ogg file") },
	})
	p, warns, infos := menuPlayer(t, broken)
	f := titleFlow(t)
	for range 60 {
		p.Update(f)
	}
	if len(*warns) != 1 || !strings.Contains((*warns)[0], "sounds/theme.ogg") {
		t.Fatalf("warnings = %q, want exactly one naming sounds/theme.ogg", *warns)
	}
	if mix.PlayingMusic() || len(*infos) != 0 {
		t.Errorf("a broken theme must stay silent and unannounced (infos %q)", *infos)
	}
	// After a match the same screen may try the file once more.
	p.Update(matchFlow(t))
	p.Update(f)
	p.Update(f)
	if len(*warns) != 2 {
		t.Errorf("warnings = %d after returning from a match, want one more attempt (2 in all)", len(*warns))
	}
}

func TestSetPack_AnotherPackReleasesTheOldMenuSoundsAndMusicAndTheSamePackKeepsThem(t *testing.T) {
	p, _, infos := menuPlayer(t, builtInPack())
	f := titleFlow(t)
	p.Update(f)
	f.Navigate(game.NavConfirm) // queues the confirm sound
	p.Update(f)
	if len(p.menuChunks) == 0 || !p.menuPlaying {
		t.Fatalf("setup: menu chunks %d, music playing %v", len(p.menuChunks), p.menuPlaying)
	}
	p.SetPack(p.pack)
	if len(p.menuChunks) == 0 || !p.menuPlaying {
		t.Error("setting the pack already in use released its sounds or music")
	}
	p.SetPack(builtInPack())
	if len(p.menuChunks) != 0 || p.menuPlaying || p.music != nil {
		t.Errorf("after another pack: chunks %d, playing %v, music %v, want all released", len(p.menuChunks), p.menuPlaying, p.music)
	}
	p.Update(f)
	themes := 0
	for _, i := range *infos {
		if strings.HasSuffix(i, "sounds/theme.ogg") {
			themes++
		}
	}
	if themes != 2 || !p.menuPlaying {
		t.Errorf("the new pack's music did not start: theme loads %d, playing %v", themes, p.menuPlaying)
	}
}

func TestUpdate_AMenuMusicWithLoopStartRestartsFromItWhenItEnds(t *testing.T) {
	pack := packWith(t, map[string]func([]byte) []byte{
		"screenpack.json": func(b []byte) []byte {
			return []byte(strings.Replace(string(b), `"music": { "file": "sounds/theme.ogg", "volume": 0.6 }`,
				`"music": { "file": "sounds/tick.ogg", "volume": 0.6, "loopStartMs": 10 }`, 1))
		},
	})
	p, warns, _ := menuPlayer(t, pack)
	f := titleFlow(t)
	p.Update(f)
	if !p.menuPlaying {
		t.Fatalf("the short theme is not playing, warnings %v", *warns)
	}
	for i := 0; i < 100 && p.loopRestarts < 2; i++ {
		time.Sleep(20 * time.Millisecond)
		p.Update(f)
	}
	if p.loopRestarts < 2 {
		t.Fatalf("restarts = %d after 2 s of the short tick sound, want at least 2", p.loopRestarts)
	}
	if !mix.PlayingMusic() {
		t.Error("the music must keep playing after a restart")
	}
}

func TestUpdate_AMenuMusicWithoutLoopStartLoopsByItself(t *testing.T) {
	pack := packWith(t, map[string]func([]byte) []byte{
		"screenpack.json": func(b []byte) []byte {
			return []byte(strings.Replace(string(b), `"sounds/theme.ogg"`, `"sounds/tick.ogg"`, 1))
		},
	})
	p, _, _ := menuPlayer(t, pack)
	f := titleFlow(t)
	for range 20 {
		p.Update(f)
		time.Sleep(20 * time.Millisecond)
	}
	if p.loopRestarts != 0 || !mix.PlayingMusic() {
		t.Fatalf("restarts = %d, playing %v: without loopStartMs the mixer loops the whole track", p.loopRestarts, mix.PlayingMusic())
	}
}
