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
type portraitCache struct {
	fsys     fs.FS
	chars    []game.Entry
	textures map[ui.PortraitRef]*Texture
}

func newPortraitCache() *portraitCache {
	return &portraitCache{textures: map[ui.PortraitRef]*Texture{}}
}

// bind points the cache at the flow's content and roster.
func (c *portraitCache) bind(f *game.Flow) { c.fsys, c.chars = f.FS, f.Chars }

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
	var tex *Texture
	if ref.Character < len(c.chars) {
		px, w, h, err := game.LoadPortrait(c.fsys, c.chars[ref.Character].Sprite, ref.Large)
		if err == nil && w > 0 && h > 0 {
			tex = NewTexture(px, w, h)
		}
	}
	c.textures[ref] = tex
	return tex
}

// PortraitSize implements ui.PortraitSizes.
func (c *portraitCache) PortraitSize(character int, large bool) (w, h int, ok bool) {
	tex := c.texture(ui.PortraitRef{Character: character, Large: large})
	if tex == nil {
		return 0, 0, false
	}
	return tex.W, tex.H, true
}
