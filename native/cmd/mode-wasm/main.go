//go:build js && wasm

// Command mode-wasm is the WebAssembly build of the mode loader
// (internal/mode), so the web game validates modes with the same code as the
// native one. It sets globalThis.OpenKakutouMode.parse(file, text), which
// returns a JSON string: {"ok": true, "mode": {...}} or
// {"ok": false, "error": {"file", "field", "reason"}}.
package main

import (
	"encoding/json"
	"errors"
	"syscall/js"

	"github.com/openkakutou/game/native/internal/mode"
)

type response struct {
	OK    bool        `json:"ok"`
	Mode  *mode.Mode  `json:"mode,omitempty"`
	Error *mode.Error `json:"error,omitempty"`
}

func parse(_ js.Value, args []js.Value) any {
	if len(args) != 2 || args[0].Type() != js.TypeString || args[1].Type() != js.TypeString {
		return encode(response{Error: &mode.Error{Reason: "parse(file, text) takes two strings"}})
	}
	file := args[0].String()
	m, err := mode.Parse(file, []byte(args[1].String()))
	var me *mode.Error
	if errors.As(err, &me) {
		return encode(response{Error: me})
	}
	if err != nil {
		return encode(response{Error: &mode.Error{File: file, Reason: err.Error()}})
	}
	return encode(response{OK: true, Mode: &m})
}

func encode(r response) string {
	b, err := json.Marshal(r)
	if err != nil {
		return `{"ok":false,"error":{"reason":"cannot encode the result"}}`
	}
	return string(b)
}

func main() {
	js.Global().Set("OpenKakutouMode", js.ValueOf(map[string]any{"parse": js.FuncOf(parse)}))
	select {}
}
