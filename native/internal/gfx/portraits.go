package gfx

import (
	"io/fs"

	"github.com/openkakutou/game/native/internal/game"
	"github.com/openkakutou/game/native/internal/ui"
)

// portraitCache holds the portrait textures of the roster's characters, each
// decoded from its sprite file and uploaded the first time a screen asks for
// it. A character without the portrait is remembered as missing, so nothing
// is ever read twice. It implements ui.PortraitSizes.
//
// At most portraitsPerFrame portraits are decoded per frame, so a big
// roster's first select screen fills in over a few frames instead of
// freezing one; a portrait not decoded yet is drawn as missing meanwhile.
type portraitCache struct {
	fsys     fs.FS
	chars    []game.Entry
	textures map[ui.PortraitRef]*Texture
	// decode reads and uploads one portrait, nil when it has none.
	decode func(ui.PortraitRef) *Texture
	// budget is what this frame may still decode.
	budget int
}

// portraitsPerFrame bounds the decoding work of one frame.
const portraitsPerFrame = 8

func newPortraitCache() *portraitCache {
	c := &portraitCache{textures: map[ui.PortraitRef]*Texture{}}
	c.decode = c.load
	return c
}

// bind points the cache at the flow's content and roster, once per frame.
func (c *portraitCache) bind(f *game.Flow) {
	c.fsys, c.chars = f.FS, f.Chars
	c.newFrame()
}

// newFrame gives the frame its decoding budget.
func (c *portraitCache) newFrame() { c.budget = portraitsPerFrame }

// clear deletes every portrait texture: the roster's order changed, so the
// indexes they are keyed by name other characters now.
func (c *portraitCache) clear() {
	for _, tex := range c.textures {
		if tex != nil {
			tex.Delete()
		}
	}
	clear(c.textures)
}

// texture returns a portrait's texture, nil when the character has none.
func (c *portraitCache) texture(ref ui.PortraitRef) *Texture {
	if tex, seen := c.textures[ref]; seen {
		return tex
	}
	if c.budget <= 0 {
		return nil // decoded on a later frame
	}
	c.budget--
	tex := c.decode(ref)
	c.textures[ref] = tex
	return tex
}

// load decodes a roster character's portrait and uploads it.
func (c *portraitCache) load(ref ui.PortraitRef) *Texture {
	if ref.Character >= len(c.chars) {
		return nil
	}
	px, w, h, err := game.LoadPortrait(c.fsys, c.chars[ref.Character].Sprite, ref.Large)
	if err != nil || w == 0 || h == 0 {
		return nil
	}
	return NewTexture(px, w, h)
}

// PortraitSize implements ui.PortraitSizes.
func (c *portraitCache) PortraitSize(character int, large bool) (w, h int, ok bool) {
	tex := c.texture(ui.PortraitRef{Character: character, Large: large})
	if tex == nil {
		return 0, 0, false
	}
	return tex.W, tex.H, true
}
