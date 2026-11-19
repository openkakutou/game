package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openkakutou/game/native/internal/lifebar"
)

// HUDLifebar is the lifebar the match HUD draws, read from the content
// root's lifebars/ folder.
type HUDLifebar struct {
	*lifebar.Lifebar
	// Def is the fight.def it was read from.
	Def string
	// Sprites is its sprite file, found as Ikemen GO looks it up (beside the
	// def, then the content root, then common/ for data/,
	// fightscreen.go:4918); "" when none is found.
	Sprites string
}

// SelectLifebar loads the lifebar config/appearance.json selects, falling
// back to lifebars/default (roadmap decisions 031, 032). It returns nil when
// neither exists, and notices for every folder skipped and every fallback
// taken; the HUD then keeps its built-in bars.
func SelectLifebar(root string) (*HUDLifebar, []string) {
	var notices []string
	entries, problems := lifebar.Discover(joinResolved([]string{root}, "lifebars"))
	for _, p := range problems {
		notices = append(notices, p.Error())
	}
	name, err := lifebar.Selected(filepath.Join(joinResolved([]string{root}, "config"), "appearance.json"))
	if err != nil {
		notices = append(notices, fmt.Sprintf("appearance settings ignored: %v", err))
	}
	entry, ok := lifebar.Pick(entries, name)
	if !ok {
		if name != lifebar.DefaultName {
			notices = append(notices, fmt.Sprintf("lifebar %q not found, and no %q lifebar: built-in bars used", name, lifebar.DefaultName))
		}
		return nil, notices
	}
	if !strings.EqualFold(entry.Name, name) {
		notices = append(notices, fmt.Sprintf("lifebar %q not found: %q used", name, entry.Name))
	}
	src, err := os.ReadFile(entry.Def)
	if err != nil {
		return nil, append(notices, fmt.Sprintf("lifebar %q: %v", entry.Name, err))
	}
	lb, err := lifebar.Parse(string(src))
	if err != nil {
		return nil, append(notices, fmt.Sprintf("lifebar %q: %v", entry.Name, err))
	}
	hud := &HUDLifebar{Lifebar: lb, Def: entry.Def}
	if lb.Sprites != "" {
		hud.Sprites = joinResolved([]string{filepath.Dir(entry.Def), root, CommonFolder(root)}, lb.Sprites)
		if _, serr := os.Stat(hud.Sprites); serr != nil {
			notices = append(notices, fmt.Sprintf("lifebar %q: sprite file %q not found", entry.Name, lb.Sprites))
			hud.Sprites = ""
		}
	}
	return hud, notices
}
