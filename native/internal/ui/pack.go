package ui

import (
	"errors"
	"fmt"
	"image"
	"io/fs"
	"path"

	"github.com/openkakutou/game/native/internal/game"
	"github.com/openkakutou/screenpack"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	_ "golang.org/x/image/webp" // registers WebP for image.DecodeConfig and the renderer's image.Decode
	_ "image/png"
)

// packFolder is where the game folder keeps the screenpack in use.
const packFolder = "screenpacks/default"

type faceKey struct {
	font FontRef
	px   int
}

type scopeKey struct{ screen, part string }

// Pack is a resolved screenpack with what layout needs from its files: the
// parsed fonts, the image sizes and the cached font faces. Not safe for
// concurrent use: layout and the renderer share it on the game's one thread.
type Pack struct {
	resolved *screenpack.Resolved
	// fsys holds the files of the pack and of the built-in default, by Source.
	fsys   [2]fs.FS
	fonts  map[FontRef]*sfnt.Font
	faces  map[faceKey]font.Face
	sizes  map[Asset][2]int
	scopes map[scopeKey]scope
	// sheetImages caches a sprite sheet's image path, relative to its pack.
	sheetImages map[Asset]string
}

// LoadPack returns the pack to draw with: screenpacks/default of the game
// folder when it exists and is valid, else the built-in default screenpack.
// A folder that exists but is refused is reported in notices.
func LoadPack(gameFS fs.FS) (pack *Pack, notices []string) {
	builtIn, err := screenpack.Load(screenpack.DefaultFS())
	if err != nil {
		panic(err) // the built-in screenpack is checked by its own tests
	}
	fallback, err := newPack(builtIn, builtIn, screenpack.DefaultFS(), screenpack.DefaultFS())
	if err != nil {
		panic(err) // same
	}
	folder, err := game.ResolvePath(gameFS, ".", packFolder)
	if errors.Is(err, fs.ErrNotExist) {
		return fallback, nil
	}
	if err != nil {
		return fallback, []string{fmt.Sprintf("%s: %v; using the built-in screenpack", packFolder, err)}
	}
	sub, err := fs.Sub(gameFS, folder)
	if err != nil {
		return fallback, []string{fmt.Sprintf("%s: %v; using the built-in screenpack", folder, err)}
	}
	loaded, err := screenpack.Load(sub)
	if err == nil {
		pack, err = newPack(loaded, builtIn, sub, screenpack.DefaultFS())
	}
	if err != nil {
		return fallback, []string{fmt.Sprintf("%s is refused, using the built-in screenpack:\n%v", folder, err)}
	}
	return pack, nil
}

// newPack resolves own against def and parses both packs' fonts.
func newPack(own, def *screenpack.Screenpack, ownFS, defFS fs.FS) (*Pack, error) {
	resolved, err := screenpack.Resolve(own, def)
	if err != nil {
		return nil, err
	}
	p := &Pack{
		resolved:    resolved,
		fsys:        [2]fs.FS{ownFS, defFS},
		fonts:       map[FontRef]*sfnt.Font{},
		faces:       map[faceKey]font.Face{},
		sizes:       map[Asset][2]int{},
		scopes:      map[scopeKey]scope{},
		sheetImages: map[Asset]string{},
	}
	for src, sp := range [2]*screenpack.Screenpack{own, def} {
		for id, file := range sp.Fonts {
			data, err := fs.ReadFile(p.fsys[src], string(file))
			if err != nil {
				return nil, fmt.Errorf("font %s: %w", file, err)
			}
			parsed, err := opentype.Parse(data)
			if err != nil {
				return nil, fmt.Errorf("font %s: %w", file, err)
			}
			p.fonts[FontRef{screenpack.Source(src), id}] = parsed
		}
	}
	return p, nil
}

// Open opens an asset's file, for the renderer to decode.
func (p *Pack) Open(a Asset) (fs.File, error) { return p.fsys[a.Source].Open(a.Path) }

// Face returns the font at a pixel size, created once. It is nil when the
// font is unknown.
func (p *Pack) Face(f FontRef, px int) font.Face {
	key := faceKey{f, px}
	if face, ok := p.faces[key]; ok {
		return face
	}
	parsed, ok := p.fonts[f]
	if !ok {
		return nil
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: float64(px), DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil // only a non-positive size fails, and layout asks for at least 1 px
	}
	p.faces[key] = face
	return face
}

// imageSize reads an image's size once, from its header only; ok is false
// when the file cannot be read as an image.
func (p *Pack) imageSize(a Asset) (w, h int, ok bool) {
	if size, seen := p.sizes[a]; seen {
		return size[0], size[1], size[0] > 0
	}
	var size [2]int
	if file, err := p.Open(a); err == nil {
		cfg, _, err := image.DecodeConfig(file)
		file.Close()
		if err == nil {
			size = [2]int{cfg.Width, cfg.Height}
		}
	}
	p.sizes[a] = size
	return size[0], size[1], size[0] > 0
}

// screenpack is the pack a Source names.
func (p *Pack) screenpack(src screenpack.Source) *screenpack.Screenpack {
	if src == screenpack.FromDefault {
		return p.resolved.Default
	}
	return p.resolved.Pack
}

// sheetImage is the image path, relative to its pack, of a sprite sheet, and
// the sheet.
func (p *Pack) sheetImage(src screenpack.Source, sheet screenpack.Path) (string, *screenpack.SpriteSheet) {
	spec := p.screenpack(src).Sheets[sheet]
	if spec == nil {
		return "", nil
	}
	key := Asset{src, string(sheet)}
	file, ok := p.sheetImages[key]
	if !ok {
		file = path.Join(path.Dir(string(sheet)), string(spec.Image))
		p.sheetImages[key] = file
	}
	return file, spec
}

// scope is where a part's names are looked up: its pack's assets and fonts,
// and text styles from the screen's own then the pack's root.
type scope struct {
	src    screenpack.Source
	styles map[screenpack.StyleID]screenpack.TextStyle
}

// scope returns the scope of a screen part ("" for the screen's own
// background and foreground), found once. A part the pack leaves out comes
// from the default pack and keeps using its styles and assets.
func (p *Pack) scope(screen, part string) scope {
	key := scopeKey{screen, part}
	if sc, ok := p.scopes[key]; ok {
		return sc
	}
	src := p.resolved.Source(screen, part)
	sc := scope{src: src, styles: screenStyles(&p.screenpack(src).Screens, screen)}
	p.scopes[key] = sc
	return sc
}

// screenBase is the fields every screen file has, of the named screen; nil
// for an unknown name.
func screenBase(s *screenpack.Screens, screen string) *screenpack.Screen {
	switch screen {
	case "title":
		return &s.Title.Screen
	case "options":
		return &s.Options.Screen
	case "pause":
		return &s.Pause.Screen
	case "matchSetup":
		return &s.MatchSetup.Screen
	case "versus":
		return &s.Versus.Screen
	case "results":
		return &s.Results.Screen
	case "characterSelect":
		return &s.CharacterSelect.Screen
	case "stageSelect":
		return &s.StageSelect.Screen
	case "error":
		return &s.Error.Screen
	}
	return nil
}

// screenStyles are a screen's own text styles.
func screenStyles(s *screenpack.Screens, screen string) map[screenpack.StyleID]screenpack.TextStyle {
	if base := screenBase(s, screen); base != nil {
		return base.TextStyles
	}
	return nil
}

// screenMenu is the menu part of the named screen, nil when it has none.
func screenMenu(s *screenpack.Screens, screen string) *screenpack.Menu {
	switch screen {
	case "title":
		return s.Title.Menu
	case "options":
		return s.Options.Menu
	case "pause":
		return s.Pause.Menu
	case "matchSetup":
		return s.MatchSetup.Menu
	case "results":
		return s.Results.Menu
	}
	return nil
}

// style finds a text style: the screen's first, then the pack's root.
func (p *Pack) style(sc scope, id screenpack.StyleID) screenpack.TextStyle {
	if st, ok := sc.styles[id]; ok {
		return st
	}
	return p.screenpack(sc.src).TextStyles[id]
}

// SelectColumns is how many cells a row of the character select grid has.
func (p *Pack) SelectColumns() int {
	return p.resolved.Screens.CharacterSelect.Grid.Columns
}

// Sort is the order rule of a select screen's roster ("characterSelect" or
// "stageSelect"): what to sort by (folder, displayName or author) and
// whether descending. A screen without a rule sorts by folder.
func (p *Pack) Sort(screen string) (by string, descending bool) {
	var rule *screenpack.Sort
	switch screen {
	case "characterSelect":
		rule = p.resolved.Screens.CharacterSelect.Sort
	case "stageSelect":
		rule = p.resolved.Screens.StageSelect.Sort
	}
	if rule == nil {
		return "folder", false
	}
	return rule.By, rule.Descending
}

// Sound is the file of a menu sound (move, confirm, cancel or error) heard on
// a screen: the screen's menu overrides the screen, which overrides the root
// file of its pack, which the default pack's backs up. screen "" is no
// screen, so the root files only. ok is false when no file names the sound.
func (p *Pack) Sound(screen, name string) (a Asset, ok bool) {
	src := screenpack.FromPack
	if screen != "" {
		if a, ok := p.screenSound(screen, name); ok {
			return a, true
		}
		src = p.scope(screen, "").src
	}
	if file, found := p.screenpack(src).Sounds[name]; found {
		return Asset{src, string(file)}, true
	}
	if file, found := p.resolved.Default.Sounds[name]; found {
		return Asset{screenpack.FromDefault, string(file)}, true
	}
	return Asset{}, false
}

// screenSound looks a sound up in a screen's menu, then in the screen itself.
func (p *Pack) screenSound(screen, name string) (Asset, bool) {
	menu := p.scope(screen, "menu").src
	if m := screenMenu(&p.screenpack(menu).Screens, screen); m != nil {
		if file, found := m.Sounds[name]; found {
			return Asset{menu, string(file)}, true
		}
	}
	src := p.scope(screen, "").src
	if base := screenBase(&p.screenpack(src).Screens, screen); base != nil {
		if file, found := base.Sounds[name]; found {
			return Asset{src, string(file)}, true
		}
	}
	return Asset{}, false
}

// MenuMusic is the music of a screen: Volume (0 to 1) multiplies the player's
// music volume, and the track loops from LoopStartMs.
type MenuMusic struct {
	Asset       Asset
	Volume      float32
	LoopStartMs int
}

// Music is the music the named screen plays: its own, else the pack's root
// music, and none when the screen says `"music": null` or nothing names one.
// Two screens that name the same file have equal Asset, which the player
// takes as "keep playing".
func (p *Pack) Music(screen string) (m MenuMusic, ok bool) {
	sc := p.scope(screen, "")
	sp := p.screenpack(sc.src)
	var music *screenpack.Music
	if base := screenBase(&sp.Screens, screen); base != nil {
		if base.Silent {
			return MenuMusic{}, false
		}
		music = base.Music
	}
	if music == nil {
		music = sp.Music
	}
	if music == nil {
		return MenuMusic{}, false
	}
	m = MenuMusic{Asset: Asset{sc.src, string(music.File)}, Volume: 1, LoopStartMs: music.LoopStartMs}
	if music.Volume != nil {
		m.Volume = float32(*music.Volume)
	}
	return m, true
}

// defaultWarningColor is the format's own default for the root file's
// warningColor, which the library leaves empty when the file omits it.
const defaultWarningColor = "#dc3c3c"

// warning is the color that tints what the game flags as a problem, from the
// root file of the pack a part comes from.
func (p *Pack) warning(sc scope) string {
	if color := p.screenpack(sc.src).WarningColor; color != "" {
		return color
	}
	return defaultWarningColor
}
