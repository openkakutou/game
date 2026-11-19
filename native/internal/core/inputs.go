package core

import (
	"errors"
	"fmt"

	"github.com/openkakutou/engine/v2/input"
)

// buttonNames are the buttons an input may hold (the MUGEN/Ikemen GO set).
// The map keys are these constants, so parsing keeps no reference to the
// request string.
var buttonNames = []string{"a", "b", "c", "x", "y", "z", "s", "d", "w", "m"}

// scanner reads the small, fixed JSON shape of a tick's inputs without
// allocating (encoding/json would allocate on every tick).
type scanner struct {
	s string
	i int
}

func (sc *scanner) skip() {
	for sc.i < len(sc.s) && (sc.s[sc.i] == ' ' || sc.s[sc.i] == '\n' || sc.s[sc.i] == '\t' || sc.s[sc.i] == '\r') {
		sc.i++
	}
}

func (sc *scanner) expect(c byte) error {
	sc.skip()
	if sc.i >= len(sc.s) || sc.s[sc.i] != c {
		return fmt.Errorf("expected %q at offset %d", c, sc.i)
	}
	sc.i++
	return nil
}

// nextKey moves to the next member of the object being read and returns its
// key, or ok=false at the object's end. first says no member was read yet.
func (sc *scanner) nextKey(first *bool) (key string, ok bool, err error) {
	sc.skip()
	if sc.i < len(sc.s) && sc.s[sc.i] == '}' {
		sc.i++
		return "", false, nil
	}
	if !*first {
		if err := sc.expect(','); err != nil {
			return "", false, err
		}
	}
	*first = false
	if err := sc.expect('"'); err != nil {
		return "", false, err
	}
	start := sc.i
	for sc.i < len(sc.s) && sc.s[sc.i] != '"' {
		if sc.s[sc.i] == '\\' {
			return "", false, errors.New("escapes in keys are not supported")
		}
		sc.i++
	}
	key = sc.s[start:min(sc.i, len(sc.s))]
	if err := sc.expect('"'); err != nil {
		return "", false, err
	}
	return key, true, sc.expect(':')
}

func (sc *scanner) boolean() (bool, error) {
	sc.skip()
	rest := sc.s[sc.i:]
	switch {
	case len(rest) >= 4 && rest[:4] == "true":
		sc.i += 4
		return true, nil
	case len(rest) >= 5 && rest[:5] == "false":
		sc.i += 5
		return false, nil
	}
	return false, fmt.Errorf("expected a boolean at offset %d", sc.i)
}

// parseInputs reads {"p1":{...},"p2":{...}} into dst, keys left out reading
// as not held. The buttons maps of dst are reused.
func parseInputs(s string, dst *[2]input.TickInput) error {
	for i := range dst {
		dst[i].Up, dst[i].Down, dst[i].Left, dst[i].Right = false, false, false, false
		clear(dst[i].Buttons)
	}
	sc := scanner{s: s}
	if err := sc.expect('{'); err != nil {
		return err
	}
	first := true
	for {
		key, more, err := sc.nextKey(&first)
		if err != nil {
			return err
		}
		if !more {
			break
		}
		var player *input.TickInput
		switch key {
		case "p1":
			player = &dst[0]
		case "p2":
			player = &dst[1]
		default:
			return fmt.Errorf("unknown player %q", key)
		}
		if err := sc.player(player); err != nil {
			return err
		}
	}
	sc.skip()
	if sc.i != len(sc.s) {
		return fmt.Errorf("unexpected data at offset %d", sc.i)
	}
	return nil
}

func (sc *scanner) player(p *input.TickInput) error {
	if err := sc.expect('{'); err != nil {
		return err
	}
	first := true
	for {
		key, more, err := sc.nextKey(&first)
		if err != nil {
			return err
		}
		if !more {
			return nil
		}
		if key == "buttons" {
			err = sc.buttons(p)
		} else {
			err = sc.direction(key, p)
		}
		if err != nil {
			return err
		}
	}
}

func (sc *scanner) direction(key string, p *input.TickInput) error {
	var target *bool
	switch key {
	case "up":
		target = &p.Up
	case "down":
		target = &p.Down
	case "left":
		target = &p.Left
	case "right":
		target = &p.Right
	default:
		return fmt.Errorf("unknown input %q", key)
	}
	v, err := sc.boolean()
	*target = v
	return err
}

func (sc *scanner) buttons(p *input.TickInput) error {
	if err := sc.expect('{'); err != nil {
		return err
	}
	first := true
	for {
		key, more, err := sc.nextKey(&first)
		if err != nil {
			return err
		}
		if !more {
			return nil
		}
		name, known := buttonName(key)
		if !known {
			return fmt.Errorf("unknown button %q", key)
		}
		v, err := sc.boolean()
		if err != nil {
			return err
		}
		p.Buttons[name] = v
	}
}

func buttonName(key string) (string, bool) {
	for _, name := range buttonNames {
		if name == key {
			return name, true
		}
	}
	return "", false
}
