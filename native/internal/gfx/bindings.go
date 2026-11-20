package gfx

import (
	"fmt"

	"github.com/openkakutou/game/native/internal/game"
	"github.com/veandco/go-sdl2/sdl"
)

// Bindings are the players' controls as SDL scancodes and controller buttons,
// resolved from the settings' names once per change so a tick looks nothing up.
type Bindings struct {
	gen  int
	keys [2][len(game.KeyboardActions)]sdl.Scancode
	pads [2][len(game.PadActions)]sdl.GameControllerButton
}

// NewBindings returns bindings that resolve on the first Update.
func NewBindings() *Bindings { return &Bindings{gen: -1} }

// Update resolves the settings' controls when their generation changed. A
// name SDL does not know sends the settings back to the defaults, with a
// notice.
func (b *Bindings) Update(s *game.Settings) {
	if s.ControlsGen == b.gen {
		return
	}
	if err := b.resolve(s.Controls); err != nil {
		s.RejectControls(err)
		// The defaults are SDL's own names: this cannot fail again.
		_ = b.resolve(s.Controls)
	}
	b.gen = s.ControlsGen
}

func (b *Bindings) resolve(c game.Controls) error {
	for p := range c {
		for i, name := range c[p].Keyboard {
			sc := sdl.GetScancodeFromName(name)
			if sc == sdl.SCANCODE_UNKNOWN {
				return fmt.Errorf("player %d %s: unknown key %q", p+1, game.KeyboardActions[i], name)
			}
			b.keys[p][i] = sc
		}
		for i, name := range c[p].Pad {
			button := sdl.GameControllerGetButtonFromString(name)
			if button == sdl.CONTROLLER_BUTTON_INVALID {
				return fmt.Errorf("player %d pad %s: unknown button %q", p+1, game.PadActions[i], name)
			}
			b.pads[p][i] = button
		}
	}
	return nil
}
