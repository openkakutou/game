package game

import (
	"fmt"
	"io/fs"

	"github.com/openkakutou/game/native/internal/mode"
)

// versusFile is where the game folder holds the Versus mode (roadmap decision 031).
const versusFile = "modes/versus.jsonc"

// VersusMode reads the Versus mode from the game folder's modes/versus.jsonc.
// A missing or invalid file falls back to the built-in copy, with a notice
// saying why, so the game always has its Versus mode.
func VersusMode(fsys fs.FS) (mode.Mode, []string) {
	builtin, err := mode.Versus()
	if err != nil {
		panic(fmt.Sprintf("the built-in versus mode is invalid: %v", err)) // guarded by mode's TestVersus_EmbeddedModeParses
	}
	name, err := ResolvePath(fsys, ".", versusFile)
	if err != nil {
		return builtin, []string{fmt.Sprintf("%s not found, using the built-in Versus mode", versusFile)}
	}
	src, err := fs.ReadFile(fsys, name)
	if err != nil {
		return builtin, []string{fmt.Sprintf("%s: %v, using the built-in Versus mode", versusFile, err)}
	}
	m, err := mode.Parse(name, src)
	if err != nil {
		return builtin, []string{fmt.Sprintf("%v, using the built-in Versus mode", err)}
	}
	return m, nil
}
