package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
)

// Rebindable controls (decision 024). The core keeps names only: SDL scancode
// names for keys ("W", "Up", "Return"), game controller button names for
// buttons ("a", "leftshoulder"); the SDL layer turns them into scancodes and
// buttons when they change.

// Device is a kind of input device with its own bindings.
type Device int

// The devices.
const (
	DeviceKeyboard Device = iota
	DevicePad
)

// The bindable actions, in the order of PlayerControls' arrays. The pad's
// directions stay on the d-pad and the left stick. The first PadButtons pad
// actions are the fighter's buttons.
var (
	KeyboardActions = [...]string{"up", "down", "left", "right", "a", "b", "c", "x", "y", "z"}
	PadActions      = [...]string{"a", "b", "c", "x", "y", "z", "start"}
)

// KeyDirections is how many of KeyboardActions come first as directions: the
// rest are the fighter's buttons, named like the pad's.
const KeyDirections = 4

// PlayerControls is one player's binding of each action, by the order of
// KeyboardActions and PadActions.
type PlayerControls struct {
	Keyboard [len(KeyboardActions)]string
	Pad      [len(PadActions)]string
}

// Controls is both players' bindings.
type Controls [2]PlayerControls

// defaultControls are the bindings of a missing or invalid controls.json: the
// web build's, P1 on WASD + F/G/H/R/T/Y, P2 on the arrows + J/K/L/I/O/P, and
// a pad's A/B/X/Y/LB/RB/Start for both.
var defaultControls = func() Controls {
	pad := [len(PadActions)]string{"a", "b", "x", "y", "leftshoulder", "rightshoulder", "start"}
	return Controls{
		{Keyboard: [len(KeyboardActions)]string{"W", "S", "A", "D", "F", "G", "H", "R", "T", "Y"}, Pad: pad},
		{Keyboard: [len(KeyboardActions)]string{"Up", "Down", "Left", "Right", "J", "K", "L", "I", "O", "P"}, Pad: pad},
	}
}()

// actions are the names of device's actions.
func actions(d Device) []string {
	if d == DeviceKeyboard {
		return KeyboardActions[:]
	}
	return PadActions[:]
}

// names are player p's bindings of device d.
func (c *Controls) names(p int, d Device) []string {
	if d == DeviceKeyboard {
		return c[p].Keyboard[:]
	}
	return c[p].Pad[:]
}

// The shape of a platform's section of controls.json.
type playerSection struct {
	Keyboard map[string]string `json:"keyboard"`
	Gamepad  map[string]string `json:"gamepad"`
}

type controlsSection struct {
	P1 playerSection `json:"p1"`
	P2 playerSection `json:"p2"`
}

// decode applies a platform's section to c, which holds the defaults: an
// action the section lacks keeps its default. An unknown player, device or
// action, or an empty name, is an error; the same name on several actions is
// not (the screen shows it).
func (c *Controls) decode(data []byte) error {
	var section controlsSection
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&section); err != nil {
		return err
	}
	for p, ps := range [2]playerSection{section.P1, section.P2} {
		if err := applyNames(c.names(p, DeviceKeyboard), KeyboardActions[:], ps.Keyboard); err != nil {
			return fmt.Errorf("p%d keyboard: %w", p+1, err)
		}
		if err := applyNames(c.names(p, DevicePad), PadActions[:], ps.Gamepad); err != nil {
			return fmt.Errorf("p%d gamepad: %w", p+1, err)
		}
	}
	return nil
}

func applyNames(dst, actions []string, names map[string]string) error {
	for action, name := range names {
		i := slices.Index(actions, action)
		if i < 0 {
			return fmt.Errorf("unknown action %q", action)
		}
		if name == "" {
			return fmt.Errorf("action %q has an empty name", action)
		}
		dst[i] = name
	}
	return nil
}

// section is c as a platform's section of controls.json.
func (c Controls) section() controlsSection {
	return controlsSection{P1: c.playerSection(0), P2: c.playerSection(1)}
}

func (c Controls) playerSection(p int) playerSection {
	ps := playerSection{Keyboard: map[string]string{}, Gamepad: map[string]string{}}
	for i, name := range c.names(p, DeviceKeyboard) {
		ps.Keyboard[KeyboardActions[i]] = name
	}
	for i, name := range c.names(p, DevicePad) {
		ps.Gamepad[PadActions[i]] = name
	}
	return ps
}
