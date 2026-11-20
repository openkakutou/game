package game

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/openkakutou/game/native/internal/lifebar"
)

// HUDLifebar is the lifebar the match HUD draws, read from the content
// file system's lifebars/ folder.
type HUDLifebar struct {
	*lifebar.Lifebar
	// Def is the name of the fight.def it was read from.
	Def string
	// Sprites is its sprite file, found as Ikemen GO looks it up (beside the
	// def, then the content root, then common/ for data/,
	// fightscreen.go:4918), as a name in the content file system; "" when
	// none is found.
	Sprites string
}

// DiscoverLifebars lists the usable folders of lifebars/, with a notice for
// every folder skipped.
func DiscoverLifebars(fsys fs.FS) (entries []lifebar.Entry, notices []string) {
	entries, problems := lifebar.Discover(fsys, joinResolved(fsys, []string{"."}, "lifebars"))
	for _, p := range problems {
		notices = append(notices, p.Error())
	}
	return entries, notices
}

// SelectLifebar loads the lifebar named name among entries, falling back to
// lifebars/default (roadmap decisions 031, 032). It returns nil when neither
// exists, and notices for every fallback taken; the HUD then keeps its
// built-in bars.
func SelectLifebar(fsys fs.FS, entries []lifebar.Entry, name string) (*HUDLifebar, []string) {
	var notices []string
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
	src, err := fs.ReadFile(fsys, entry.Def)
	if err != nil {
		return nil, append(notices, fmt.Sprintf("lifebar %q: %v", entry.Name, err))
	}
	lb, err := lifebar.Parse(string(src))
	if err != nil {
		return nil, append(notices, fmt.Sprintf("lifebar %q: %v", entry.Name, err))
	}
	hud := &HUDLifebar{Lifebar: lb, Def: entry.Def}
	if lb.Sprites != "" {
		hud.Sprites = joinResolved(fsys, []string{path.Dir(entry.Def), ".", CommonFolder(fsys)}, lb.Sprites)
		if _, serr := fs.Stat(fsys, hud.Sprites); serr != nil {
			notices = append(notices, fmt.Sprintf("lifebar %q: sprite file %q not found", entry.Name, lb.Sprites))
			hud.Sprites = ""
		}
	}
	return hud, notices
}
