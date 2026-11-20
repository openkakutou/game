package mode

import _ "embed"

//go:embed versus.jsonc
var versusSrc []byte

// Versus returns the built-in Versus mode, the embedded versus.jsonc.
func Versus() (Mode, error) {
	return Parse("versus.jsonc", versusSrc)
}
