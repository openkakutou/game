package game

import (
	"fmt"
	"github.com/openkakutou/game/native/internal/bmfont"
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
	// Fonts are the [Files] fonts that loaded, by number; text in a font
	// missing here is printed in the built-in font.
	Fonts map[int]*bmfont.Font
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
	hud, more := loadLifebar(fsys, entry)
	return hud, append(notices, more...)
}

// LoadLifebarDef loads the fight.def def names (Ikemen GO's -fight), found
// as given, then under lifebars/ and common/ (FightScreen.resolvePath,
// fightscreen.go:6024-6040, data/ being common/). It returns nil and a notice
// when it cannot be read; the caller then keeps the selected lifebar.
func LoadLifebarDef(fsys fs.FS, def string) (*HUDLifebar, []string) {
	found := joinResolved(fsys, []string{".", "lifebars", CommonFolder(fsys)}, def)
	if _, err := fs.Stat(fsys, found); err != nil || found == "" {
		return nil, []string{fmt.Sprintf("-fight %q: not found", def)}
	}
	return loadLifebar(fsys, lifebar.Entry{Name: def, Def: found})
}

// loadLifebar reads entry's fight.def, its sprite file and its fonts.
func loadLifebar(fsys fs.FS, entry lifebar.Entry) (*HUDLifebar, []string) {
	var notices []string
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
	hud.Fonts, notices = loadLifebarFonts(fsys, entry, lb.Fonts, notices)
	return hud, notices
}

// loadLifebarFonts loads each [Files] font, searched like the sprite file
// plus each folder's font/ (LoadFile with "font/", fightscreen.go:5034). A
// font that cannot be read gets a notice; its text then uses the built-in
// font (Ikemen GO draws nothing).
func loadLifebarFonts(fsys fs.FS, entry lifebar.Entry, names map[int]string, notices []string) (map[int]*bmfont.Font, []string) {
	fonts := map[int]*bmfont.Font{}
	dir := path.Dir(entry.Def)
	common := CommonFolder(fsys)
	bases := []string{dir, path.Join(dir, "font"), ".", "font", common, path.Join(common, "font")}
	for n, name := range names {
		f, err := bmfont.Load(fsys, joinResolved(fsys, bases, name))
		if err != nil {
			notices = append(notices, fmt.Sprintf("lifebar %q: font%d %q not loaded, built-in font used: %v", entry.Name, n, name, err))
			continue
		}
		fonts[n] = f
	}
	return fonts, notices
}
