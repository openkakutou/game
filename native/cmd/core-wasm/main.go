//go:build js && wasm

// Command core-wasm is the WebAssembly build of the game core, so the web
// game plays matches with the same code as the native one. It sets
// globalThis.OpenKakutouCore with:
//
//	mount(name, bytes)    add a file (Uint8Array) under a slash name from the game root
//	startMatch(json)      {p1, p2, stage, setup:{rounds, timeLimitSeconds}, p2cpu, viewWidth}
//	tick(id, json)        {p1: input, p2: input}, one tick; replies with the state to draw
//	nextRound(id)         start the round after one that ended
//	closeMatch(id)        forget the match
//
// Every function returns a JSON string, {"ok":true,...} or
// {"ok":false,"error":{"reason"}}, and never throws. See package core.
package main

import (
	"syscall/js"

	"github.com/openkakutou/game/native/internal/core"
)

var c = core.New()

func mount(_ js.Value, args []js.Value) any {
	if len(args) != 2 || args[0].Type() != js.TypeString || !args[1].InstanceOf(js.Global().Get("Uint8Array")) {
		return core.Failure("mount(name, bytes) takes a string and a Uint8Array")
	}
	data := make([]byte, args[1].Get("length").Int())
	js.CopyBytesToGo(data, args[1])
	return c.Mount(args[0].String(), data)
}

func startMatch(_ js.Value, args []js.Value) any {
	if len(args) != 1 || args[0].Type() != js.TypeString {
		return core.Failure("startMatch(json) takes a string")
	}
	return c.StartMatch(args[0].String())
}

func tick(_ js.Value, args []js.Value) any {
	if len(args) != 2 || args[0].Type() != js.TypeNumber || args[1].Type() != js.TypeString {
		return core.Failure("tick(id, json) takes a number and a string")
	}
	return c.Tick(args[0].Int(), args[1].String())
}

func nextRound(_ js.Value, args []js.Value) any {
	if len(args) != 1 || args[0].Type() != js.TypeNumber {
		return core.Failure("nextRound(id) takes a number")
	}
	return c.NextRound(args[0].Int())
}

func closeMatch(_ js.Value, args []js.Value) any {
	if len(args) != 1 || args[0].Type() != js.TypeNumber {
		return core.Failure("closeMatch(id) takes a number")
	}
	return c.CloseMatch(args[0].Int())
}

func main() {
	js.Global().Set("OpenKakutouCore", js.ValueOf(map[string]any{
		"mount":      js.FuncOf(mount),
		"startMatch": js.FuncOf(startMatch),
		"tick":       js.FuncOf(tick),
		"nextRound":  js.FuncOf(nextRound),
		"closeMatch": js.FuncOf(closeMatch),
	}))
	select {}
}
