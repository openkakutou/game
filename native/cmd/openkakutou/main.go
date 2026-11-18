// Command openkakutou is the native (Windows/Mac/Linux) Quick Versus game:
// raw go-gl + SDL2, importing character/stage/sff/engine as Go packages.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/openkakutou/engine/input"
	"github.com/openkakutou/game/native/internal/audio"
	"github.com/openkakutou/game/native/internal/game"
	"github.com/openkakutou/game/native/internal/gfx"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "openkakutou:", err)
		os.Exit(1)
	}
}

func run() error {
	content := flag.String("content", "", "folder with chars/ (one folder per character) and stages/ (stage .def files); default: the folder holding the program (on macOS, the one holding the .app)")
	frames := flag.Int("frames", 0, "exit after this many frames (0 = run until closed); for smoke tests")
	capture := flag.String("capture", "", "write the last frame to this .png before exiting (needs -frames)")
	width := flag.Int("width", 960, "window width")
	height := flag.Int("height", 720, "window height")
	flag.Parse()

	root := *content
	if root == "" {
		exe, err := os.Executable()
		if err == nil {
			exe, err = filepath.EvalSymlinks(exe)
		}
		if err != nil {
			return fmt.Errorf("locating the program: %w", err)
		}
		root = game.ContentRoot(exe)
	}

	// A missing folder is reported on the error screen, naming it.
	var chars, stages []game.Entry
	charsDir, stagesDir, folderErr := game.ContentFolders(root)
	if folderErr == nil {
		var charErr, stageErr error
		chars, charErr = game.DiscoverCharacters(charsDir)
		stages, stageErr = game.DiscoverStages(stagesDir)
		folderErr = errors.Join(charErr, stageErr)
	}
	flow := game.NewFlow(chars, stages, game.DefaultLoader(game.CommonFolder(root)))
	flow.ContentRoot = root
	if folderErr != nil {
		flow.Screen = game.ScreenError
		flow.Message = fmt.Sprintf("Cannot read the content: %v. Use -content to point at the content folder.", folderErr)
	}

	win, err := gfx.NewWindow("Quick Versus", int32(*width), int32(*height))
	if err != nil {
		return err
	}
	defer win.Close()
	fmt.Println("renderer:", win.Renderer())
	renderer := gfx.NewRenderer(win)

	aud := audio.New()
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
		lastFrame := *frames > 0 && n+1 >= *frames
		if lastFrame && *capture != "" {
			if err := win.SavePNG(*capture); err != nil { // before the swap: the back buffer is still defined
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
