package gfx

import (
	"github.com/openkakutou/engine/v2/input"
	"github.com/openkakutou/game/native/internal/game"
	"github.com/veandco/go-sdl2/sdl"
)

// Held reads both players' held inputs into dst, reusing the button maps
// already in dst so a tick allocates nothing. A player with a controller uses
// it instead of the keyboard.
func Held(dst *[2]input.TickInput, pads *Gamepads, b *Bindings) {
	keys := sdl.GetKeyboardState()
	for i := range dst {
		if s, ok := pads.state(i, &b.pads[i]); ok {
			s.Apply(&dst[i])
			continue
		}
		k, in := &b.keys[i], &dst[i]
		in.Up, in.Down = keys[k[0]] != 0, keys[k[1]] != 0
		in.Left, in.Right = keys[k[2]] != 0, keys[k[3]] != 0
		if in.Buttons == nil {
			in.Buttons = make(map[string]bool, len(k)-game.KeyDirections)
		}
		for a := game.KeyDirections; a < len(k); a++ {
			in.Buttons[game.KeyboardActions[a]] = keys[k[a]] != 0
		}
	}
}

// PollEvents drains SDL events, turning key presses and controller changes
// into menu inputs for f. It reports false when the window was closed.
//
// While the controls screen waits for a key or button, the next one goes to
// f.Bind instead of the menus; Escape cancels the wait.
func PollEvents(f *game.Flow, pads *Gamepads, b *Bindings) bool {
	device, waiting := f.WaitingForBinding()
	padWait := waiting && device == game.DevicePad
	for ev := sdl.PollEvent(); ev != nil; ev = sdl.PollEvent() {
		switch e := ev.(type) {
		case *sdl.QuitEvent:
			return false
		case *sdl.KeyboardEvent:
			if e.Type != sdl.KEYDOWN {
				continue
			}
			if device, waiting := f.WaitingForBinding(); waiting {
				bindKey(f, device, e)
				continue
			}
			if n, ok := navFor(e.Keysym.Scancode); ok {
				if n == game.NavCancel && e.Keysym.Scancode == sdl.SCANCODE_ESCAPE && f.Screen == game.ScreenMatch {
					n = game.NavPause // Escape pauses; a match's other cancel inputs are fighter buttons
				}
				f.NavigateBy(n, pads.pads.KeyboardPlayer())
			}
		case *sdl.ControllerButtonEvent:
			if device, waiting := f.WaitingForBinding(); waiting && device == game.DevicePad && e.Type == sdl.CONTROLLERBUTTONDOWN {
				f.Bind(game.DevicePad, sdl.GameControllerGetStringForButton(sdl.GameControllerButton(e.Button)))
			}
		case *sdl.ControllerDeviceEvent:
			switch e.Type {
			case sdl.CONTROLLERDEVICEADDED:
				pads.add(int(e.Which))
			case sdl.CONTROLLERDEVICEREMOVED:
				pads.remove(e.Which)
			}
		}
	}
	// A button bound this frame is not also a menu press; during a keyboard
	// wait a pad's B still cancels it.
	pads.navigate(f, b, padWait)
	return true
}

// bindKey handles a key press while the controls screen waits for a binding:
// Escape cancels (it is never bindable, menus need it), a key binds when a key
// is awaited and is otherwise ignored; held keys repeating do not rebind.
func bindKey(f *game.Flow, device game.Device, e *sdl.KeyboardEvent) {
	switch {
	case e.Keysym.Scancode == sdl.SCANCODE_ESCAPE:
		f.Navigate(game.NavCancel)
	case device == game.DeviceKeyboard && e.Repeat == 0:
		f.Bind(game.DeviceKeyboard, sdl.GetScancodeName(e.Keysym.Scancode))
	}
}

func navFor(sc sdl.Scancode) (game.Nav, bool) {
	switch sc {
	case sdl.SCANCODE_UP, sdl.SCANCODE_W:
		return game.NavUp, true
	case sdl.SCANCODE_DOWN, sdl.SCANCODE_S:
		return game.NavDown, true
	case sdl.SCANCODE_LEFT, sdl.SCANCODE_A:
		return game.NavLeft, true
	case sdl.SCANCODE_RIGHT, sdl.SCANCODE_D:
		return game.NavRight, true
	case sdl.SCANCODE_RETURN, sdl.SCANCODE_SPACE, sdl.SCANCODE_KP_ENTER:
		return game.NavConfirm, true
	case sdl.SCANCODE_ESCAPE, sdl.SCANCODE_BACKSPACE:
		return game.NavCancel, true
	}
	return 0, false
}

// Ticks returns how many simulation ticks to run for elapsedMs of real time,
// carrying the remainder in *acc (a fixed 60Hz timestep, capped so a stall
// cannot trigger a long catch-up burst).
func Ticks(acc *float64, elapsedMs float64) int {
	const step = 1000.0 / game.TickRateHz
	*acc += elapsedMs
	if *acc > 5*step {
		*acc = 5 * step
	}
	n := 0
	for *acc >= step {
		*acc -= step
		n++
	}
	return n
}
