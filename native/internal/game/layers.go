package game

import (
	"io/fs"
	"path"
	"sort"

	"github.com/openkakutou/stage"
)

// FrontLayerNo is the first layerNo drawn in front of the fighters; lower
// layers draw behind them (the stage library's documented convention, same
// split as the web build's splitElementsByLayer).
const FrontLayerNo = 1

// SplitLayers returns the indexes of elements drawn behind and in front of
// the fighters, each back-to-front: stable sort on LayerNo, ties keeping
// .def file order. Computed once per match, not per frame.
func SplitLayers(elements []stage.BGElement) (behind, front []int) {
	order := make([]int, len(elements))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return elements[order[a]].LayerNo < elements[order[b]].LayerNo
	})
	for _, i := range order {
		if elements[i].LayerNo < FrontLayerNo {
			behind = append(behind, i)
		} else {
			front = append(front, i)
		}
	}
	return behind, front
}

// LayerSprite is the sprite element e shows ticks into the round. ok is
// false when it shows nothing: a blank sprite, an animated element whose
// action block is missing or whose current frame is blank, or an unknown type.
func LayerSprite(st *stage.Stage, e stage.BGElement, ticks int) (ref stage.SpriteRef, ok bool) {
	switch e.Type {
	case stage.BGElementNormal, stage.BGElementParallax:
		ref = e.Sprite
	case stage.BGElementAnim:
		anim, found := st.Animations[e.ActionNumber]
		if !found {
			return ref, false
		}
		ref = stage.ResolveAnimationFrame(anim, ticks)
	default:
		return ref, false
	}
	return ref, !ref.IsBlank()
}

// StageSpritePath is the name in fsys of the stage's sprite sheet: beside its
// .def first, then from the content root (like Ikemen GO). It is "" when the stage names none
// or the reference is rejected (symbolic link, absolute, escaping).
func StageSpritePath(fsys fs.FS, defPath string, st *stage.Stage) string {
	if st.BGdef.SpriteFile == "" {
		return ""
	}
	return joinResolved(fsys, stageBases(defPath), st.BGdef.SpriteFile)
}

// stageBases are the folders a stage's files are looked up in, in order.
func stageBases(defPath string) []string {
	return []string{path.Dir(defPath), "."}
}
