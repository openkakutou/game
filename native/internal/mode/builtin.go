package mode

import (
	"embed"
	"fmt"
)

//go:embed versus.jsonc watch.jsonc
var builtinFiles embed.FS

// builtinNames are the built-in mode files, in file-name order.
var builtinNames = []string{"versus.jsonc", "watch.jsonc"}

// BuiltIn returns the built-in modes, the embedded files, in file-name order.
func BuiltIn() ([]Mode, error) {
	var modes []Mode
	for _, name := range builtinNames {
		src, err := builtinFiles.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("reading the built-in %s: %w", name, err)
		}
		m, err := Parse(name, src)
		if err != nil {
			return nil, err
		}
		modes = append(modes, m)
	}
	return modes, nil
}
