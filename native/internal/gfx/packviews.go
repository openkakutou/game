package gfx

import (
	"github.com/openkakutou/game/native/internal/game"
	"github.com/openkakutou/game/native/internal/ui"
)

// packView is a loaded screenpack with what draws it: its layout, and the
// painter holding its textures and glyph atlases (assets of two packs may
// share a path, so each pack has its own).
type packView struct {
	pack    *ui.Pack
	layout  *ui.Layout
	painter *listPainter
}

func newPackView(win *Window, pack *ui.Pack, portraits *portraitCache) *packView {
	return &packView{pack: pack, layout: ui.NewLayout(pack), painter: newListPainter(win, pack, portraits)}
}

// view is the loaded pack named name, loaded now when it is not.
func (r *Renderer) view(name string) *packView {
	v := r.views[name]
	if v == nil {
		v = newPackView(r.win, r.load(name), r.portraits)
		r.views[name] = v
	}
	return v
}

// viewFor is the view f's screen is drawn with: the candidate pack while the
// appearance screen previews one, else the pack in use.
func (r *Renderer) viewFor(f *game.Flow) *packView {
	if preview := f.PreviewPack(); preview != "" {
		return r.view(preview)
	}
	return r.view(r.active)
}

// SetPack makes the pack named name the one in use, loading it unless the
// preview did, releases the textures, glyph atlases and portraits of every
// other one, and returns it.
func (r *Renderer) SetPack(name string) *ui.Pack {
	v := r.view(name)
	r.active = name
	r.releaseCandidates()
	r.portraits.clear()
	return v.pack
}

// releaseCandidates releases every loaded pack but the one in use.
func (r *Renderer) releaseCandidates() {
	if len(r.views) <= 1 {
		return
	}
	for name, v := range r.views {
		if name != r.active {
			v.painter.release()
			delete(r.views, name)
		}
	}
}
