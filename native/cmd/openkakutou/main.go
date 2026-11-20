// Command openkakutou is the native (Windows/Mac/Linux) OpenKakutou game:
// raw go-gl + SDL2, importing character/stage/sff/engine as Go packages.
//
// Its command line is Ikemen GO's (see -h and docs/command-line.md): given
// two characters, it starts one match without menus (quick versus) and
// quits when the match ends.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/openkakutou/engine/v2/input"
	"github.com/openkakutou/game/native/internal/audio"
	"github.com/openkakutou/game/native/internal/game"
	"github.com/openkakutou/game/native/internal/gfx"
	"github.com/openkakutou/game/native/internal/host"
	"github.com/openkakutou/game/native/internal/mode"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "openkakutou:", err)
		os.Exit(1)
	}
}

// intFlag reads flag name as an integer of at least minimum, or fallback when
// it is absent.
func intFlag(flags map[string]string, name string, fallback, minimum int) (int, error) {
	raw, ok := flags[name]
	if !ok {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < minimum {
		return 0, fmt.Errorf("%s %q: must be a number of at least %d", name, raw, minimum)
	}
	return n, nil
}

func run() error {
	cl := game.ParseArgs(os.Args[1:])
	if cl.Help {
		fmt.Println(game.HelpText)
		return nil
	}
	frames, err := intFlag(cl.Flags, "-frames", 0, 0)
	if err != nil {
		return err
	}
	width, err := intFlag(cl.Flags, "-width", 960, 1)
	if err != nil {
		return err
	}
	height, err := intFlag(cl.Flags, "-height", 720, 1)
	if err != nil {
		return err
	}
	capture := cl.Flags["-capture"]

	root := cl.Flags["-content"]
	if root == "" {
		exe, err := os.Executable()
		if err == nil {
			exe, err = filepath.EvalSymlinks(exe)
		}
		if err != nil {
			return fmt.Errorf("locating the program: %w", err)
		}
		root = host.ContentRoot(exe)
	}

	host.MapPaths(cl, root)
	quick, isQuick, notices, err := game.QuickVersusFrom(cl)
	if err != nil {
		return err
	}
	for _, n := range notices {
		fmt.Fprintln(os.Stderr, "openkakutou:", n)
	}
	fsys := host.Open(root)

	// A missing folder is reported on the error screen, naming it.
	var chars, stages []game.Entry
	charsDir, stagesDir, folderErr := game.ContentFolders(fsys, root)
	if folderErr == nil {
		var charErr, stageErr error
		chars, charErr = game.DiscoverCharacters(fsys, charsDir)
		stages, stageErr = game.DiscoverStages(fsys, stagesDir)
		folderErr = errors.Join(charErr, stageErr)
	}
	versus, modeErr := mode.Versus() // the zero mode on error: NewFlow shows an error screen, replaced below
	flow := game.NewFlow(fsys, chars, stages, game.DefaultLoader(fsys, game.CommonFolder(fsys)), versus)
	hud, lifebarNotices := game.SelectLifebar(fsys)
	flow.Lifebar = hud
	for _, n := range lifebarNotices {
		fmt.Fprintln(os.Stderr, "openkakutou:", n)
	}
	switch {
	case isQuick:
		for _, n := range flow.StartQuickVersus(quick) {
			fmt.Fprintln(os.Stderr, "openkakutou:", n)
		}
	case modeErr != nil:
		flow.Screen = game.ScreenError
		flow.Message = fmt.Sprintf("Cannot read the game mode: %v", modeErr)
	case folderErr != nil:
		flow.Screen = game.ScreenError
		flow.Message = fmt.Sprintf("Cannot read the content: %v. Use -content to point at the content folder.", folderErr)
	}

	win, err := gfx.NewWindow("OpenKakutou", int32(width), int32(height))
	if err != nil {
		return err
	}
	defer win.Close()
	fmt.Println("renderer:", win.Renderer())
	renderer := gfx.NewRenderer(win)

	var aud *audio.Player // nil plays nothing (-nosound)
	if _, off := cl.Flags["-nosound"]; !off {
		aud = audio.New()
		_, aud.NoMusic = cl.Flags["-nomusic"]
	}
	defer aud.Close()
	fmt.Println("audio:", aud.Driver())

	pads := gfx.NewGamepads()
	defer pads.Close()
	var held [2]input.TickInput
	var acc float64
	last := time.Now()
	for n := 0; !flow.Quit && gfx.PollEvents(flow, pads); n++ {
		now := time.Now()
		ticks := gfx.Ticks(&acc, float64(now.Sub(last).Microseconds())/1000)
		last = now
		gfx.Held(&held, pads)
		for i := 0; i < ticks; i++ {
			flow.Tick(held)
		}
		aud.Update(flow)
		win.BeginFrame()
		renderer.Draw(flow)
		lastFrame := frames > 0 && n+1 >= frames
		if lastFrame && capture != "" {
			if err := win.SavePNG(capture); err != nil { // before the swap: the back buffer is still defined
				return err
			}
		}
		win.EndFrame()
		if lastFrame {
			break
		}
	}
	return nil
}
