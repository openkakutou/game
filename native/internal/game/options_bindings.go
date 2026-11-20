package game

import "slices"

// The controls screens (decision 024): Options > Controls lists Player 1,
// Player 2, Reset to defaults and Back; a player's screen lists every action
// with its key and its button. Left and right pick the column, confirm waits
// for the next key or button, which the host hands to Bind; the cancel input
// (Esc) ends the wait. Menus keep their own fixed keys, so a bad binding
// never locks a player out.

// bindRows are the actions of a player's screen, in the order of its rows:
// the keyboard's, then the pad-only ones.
var bindRows = func() []string {
	rows := slices.Clone(KeyboardActions[:])
	for _, a := range PadActions {
		if !slices.Contains(rows, a) {
			rows = append(rows, a)
		}
	}
	return rows
}()

var bindTitles = [2]string{"Player 1 controls", "Player 2 controls"}

// Rows of the controls menu before Back.
const (
	controlsRowPlayer1 = iota
	controlsRowPlayer2
	controlsRowReset
)

type bindPhase int

const (
	bindIdle bindPhase = iota
	bindWaiting
	bindConflict // the awaited name is held by another action: swap or keep
)

// bindSlot is one binding: a player's action of a device.
type bindSlot struct {
	player, action int
}

// bindState is the state of the bindings screen.
type bindState struct {
	player int
	device Device // the highlighted column
	phase  bindPhase
	// name and holder are the conflict: name is held by holder.
	name   string
	holder bindSlot
	// prompt is the line shown under the rows, "" when there is none.
	prompt string
	// flagged marks the rows whose key or button is shared with another
	// action, in red.
	flagged []bool
}

// OptionsPrompt is the question or instruction shown with the options screen.
func (f *Flow) OptionsPrompt() string { return f.options.bind.prompt }

// OptionsFlagged marks the rows of the options screen shown to draw in red;
// nil when no row is.
func (f *Flow) OptionsFlagged() []bool {
	if f.Screen != ScreenOptionsBindings {
		return nil
	}
	return f.options.bind.flagged
}

// OptionsColumn is the highlighted column of the bindings screen: the
// keyboard's (0) or the pad's (1).
func (f *Flow) OptionsColumn() int { return int(f.options.bind.device) }

// WaitingForBinding reports whether the bindings screen waits for the next
// key or button of the device, which the host then gives to Bind (the cancel
// input still goes to Navigate).
func (f *Flow) WaitingForBinding() (Device, bool) {
	b := f.options.bind
	return b.device, f.Screen == ScreenOptionsBindings && b.phase == bindWaiting
}

// confirmControlsRow runs the row of the controls menu under the cursor.
func (f *Flow) confirmControlsRow() {
	switch f.options.cursor {
	case controlsRowPlayer1, controlsRowPlayer2:
		f.options.bind = bindState{player: f.options.cursor}
		f.Screen, f.options.cursor = ScreenOptionsBindings, 0
		f.refreshBindingRows()
	case controlsRowReset:
		if f.Settings.Controls != defaultControls {
			f.Settings.Controls = defaultControls
			f.Settings.ControlsGen++
		}
	}
}

func (f *Flow) navBindings(n Nav) {
	b := &f.options.bind
	switch b.phase {
	case bindWaiting:
		if n == NavCancel {
			f.endBinding()
		}
		return
	case bindConflict:
		switch n {
		case NavConfirm:
			f.swapBinding()
		case NavCancel:
			f.endBinding()
		}
		return
	}
	switch n {
	case NavUp:
		f.options.cursor = wrap(f.options.cursor, -1, len(bindRows)+1)
	case NavDown:
		f.options.cursor = wrap(f.options.cursor, 1, len(bindRows)+1)
	case NavLeft:
		b.device = DeviceKeyboard
		f.refreshBindingRows()
	case NavRight:
		b.device = DevicePad
		f.refreshBindingRows()
	case NavConfirm:
		if f.options.cursor == len(bindRows) {
			f.leaveBindings()
			return
		}
		if f.rowSlot().action >= 0 {
			b.phase = bindWaiting
			b.prompt = bindPrompts[b.device]
		}
	case NavCancel:
		f.leaveBindings()
	}
}

var bindPrompts = [2]string{DeviceKeyboard: "Press a key...", DevicePad: "Press a button..."}

func (f *Flow) leaveBindings() {
	f.options.cursor = f.options.bind.player
	f.Screen = ScreenOptionsControls
}

// endBinding goes back to browsing the rows.
func (f *Flow) endBinding() {
	f.options.bind.phase, f.options.bind.prompt = bindIdle, ""
}

// rowSlot is the binding under the cursor in the highlighted column; its
// action is -1 when the device has no such action (a pad has no Up).
func (f *Flow) rowSlot() bindSlot {
	b := f.options.bind
	return bindSlot{b.player, slices.Index(actions(b.device), bindRows[f.options.cursor])}
}

// Bind gives the awaited key or button name to the action under the cursor.
// A name held by another action of either player is a conflict, answered by
// confirm (swap the two) or cancel (keep both); the same name as the action's
// own changes nothing. An empty name, a name of another device, or one given
// without a wait, is ignored.
func (f *Flow) Bind(d Device, name string) {
	b := &f.options.bind
	if dev, waiting := f.WaitingForBinding(); !waiting || dev != d || name == "" {
		return
	}
	target := f.rowSlot()
	if *f.slotName(target) == name {
		f.endBinding()
		return
	}
	holder, held := f.holderOf(d, name, target)
	if !held {
		*f.slotName(target) = name
		f.Settings.ControlsGen++
		f.endBinding()
		f.refreshBindingRows()
		return
	}
	b.phase, b.name, b.holder = bindConflict, name, holder
	b.prompt = name + " is Player " + string(rune('1'+holder.player)) + " " + actionLabel(actions(d)[holder.action]) + ": Enter swaps, Esc keeps"
}

// swapBinding gives the awaited name to the action under the cursor and the
// action's old name to the action that held it.
func (f *Flow) swapBinding() {
	b := f.options.bind
	target := f.rowSlot()
	*f.slotName(b.holder) = *f.slotName(target)
	*f.slotName(target) = b.name
	f.Settings.ControlsGen++
	f.endBinding()
	f.refreshBindingRows()
}

// slotName is the binding slot's name, in the settings.
func (f *Flow) slotName(s bindSlot) *string {
	return &f.Settings.Controls.names(s.player, f.options.bind.device)[s.action]
}

// holderOf finds a binding of device d that holds name, other than slot. A
// key is shared by both players, a pad belongs to one: its buttons only
// conflict with the same player's.
func (f *Flow) holderOf(d Device, name string, slot bindSlot) (bindSlot, bool) {
	for p := range f.Settings.Controls {
		if d == DevicePad && p != slot.player {
			continue
		}
		for i, n := range f.Settings.Controls.names(p, d) {
			if n == name && (bindSlot{p, i}) != slot {
				return bindSlot{p, i}, true
			}
		}
	}
	return bindSlot{}, false
}

// actionLabel is an action's name as shown: Up, A, Start.
func actionLabel(action string) string {
	return string(action[0]-'a'+'A') + action[1:]
}

// refreshBindingRows rebuilds the labels of the player's screen: each row
// shows the action, its key and its button, the highlighted column in
// angle brackets (the bitmap font has no square ones); the cells hold the
// action, key and button apart.
func (f *Flow) refreshBindingRows() {
	b := &f.options.bind
	c := f.Settings.Controls
	rows, cells := f.options.rows[:0], f.options.cells[:0]
	b.flagged = b.flagged[:0]
	for _, action := range bindRows {
		key, button := "-", "-"
		shared := false
		if i := slices.Index(KeyboardActions[:], action); i >= 0 {
			key = c[b.player].Keyboard[i]
			_, held := f.holderOf(DeviceKeyboard, key, bindSlot{b.player, i})
			shared = shared || held
		}
		if i := slices.Index(PadActions[:], action); i >= 0 {
			button = c[b.player].Pad[i]
			_, held := f.holderOf(DevicePad, button, bindSlot{b.player, i})
			shared = shared || held
		}
		cells = append(cells, OptionRow{Label: actionLabel(action), Values: [2]string{key, button}, Flagged: shared})
		if b.device == DeviceKeyboard {
			key = "<" + key + ">"
		} else {
			button = "<" + button + ">"
		}
		rows = append(rows, actionLabel(action)+"  "+key+"  "+button)
		b.flagged = append(b.flagged, shared)
	}
	f.options.rows = append(rows, "Back")
	f.options.cells = append(cells, OptionRow{Label: "Back"})
	b.flagged = append(b.flagged, false)
}
