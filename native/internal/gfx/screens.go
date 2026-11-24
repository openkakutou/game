package gfx

import (
	"bytes"
	"fmt"
	"github.com/openkakutou/engine/v2"
	"io"
	"io/fs"
	"path"
	"time"

	"github.com/openkakutou/character/air"
	"github.com/openkakutou/engine/v2/evaluator"
	"github.com/openkakutou/engine/v2/match"
	"github.com/openkakutou/engine/v2/round"
	"github.com/openkakutou/game/native/internal/game"
	"github.com/openkakutou/game/native/internal/ui"
	"github.com/openkakutou/sff"
	"github.com/openkakutou/stage"
)

var (
	white  = Rgb(240, 240, 240)
	grey   = Rgb(140, 140, 150)
	gold   = Rgb(255, 210, 60)
	red    = Rgb(220, 60, 60)
	blue   = Rgb(70, 130, 230)
	green  = Rgb(80, 200, 110)
	darkBg = Rgb(18, 20, 32)
	panel  = Rgb(34, 38, 60)
	skyBg  = Rgb(50, 70, 110)
	floor  = Rgb(70, 60, 50)
)

// defaultGroundY is the screen row of the stage floor when a stage declares none.
const defaultGroundY = 220

// stageSheet is the spriteKey.fighter value of the stage's own sprite sheet;
// a (group, image) pair is only unique within one sheet.
const stageSheet = -1

// layerPlaceholderSize is the tile drawn for a layer sprite that cannot be
// loaded, the same 32 px tile as the web build.
const layerPlaceholderSize = 32

type spriteKey struct{ fighter, group, image int }

// Renderer draws a game.Flow. It caches sprite textures per match so
// nothing is decoded or uploaded while the match runs.
type Renderer struct {
	win       *Window
	sprites   map[spriteKey]*Texture
	loadedFor *game.Match
	// faces are the fighters' lifebar portraits.
	faces [2]faceSprite
	// ghosts is the buffer of the afterimage ghosts drawn.
	ghosts []game.GhostDraw
	// behind/front are the stage's layer indexes, split once per match.
	behind, front []int
	// stageAxis is each loaded stage sprite's axis, from the stage's sheet.
	stageAxis map[spriteKey][2]int
	// fxAxis is each common fightfx sprite's axis.
	fxAxis map[spriteKey][2]int
	// camX is the match camera in view pixels; the view mapping is set up
	// once per match.
	camX              float64
	stageScale, shift float64
	// lifebar draws the flow's lifebar, when it has one.
	lifebar *lifebarHUD
	// sample draws the candidate lifebar of the appearance screen.
	sample *lifebarHUD
	// The screenpack screens: views holds the packs loaded by name (the one
	// in use, and the candidate while the appearance screen is open), load
	// loads another. layout and painter are those of the view being drawn,
	// list is the reused draw list.
	views   map[string]*packView
	active  string
	load    func(name string) *ui.Pack
	layout  *ui.Layout
	list    ui.DrawList
	painter *listPainter
	// portraits are the characters' portrait textures, loaded on first use.
	portraits *portraitCache
	// packKind is the screenpack screen shown, and packSince when it appeared,
	// for the transitions; the rest is what the screens' rows are built from
	// (reused, rebuilt only when what they show changes).
	packKind  packScreen
	packSince time.Time
	rows      []ui.Row
	labels    []string
	setup     setupRows
	result    resultTexts
	countdown countdownText
}

// NewRenderer builds a renderer for win drawing the screens of pack; load
// loads the screenpack of another name, to preview it or switch to it.
func NewRenderer(win *Window, pack *ui.Pack, load func(name string) *ui.Pack) *Renderer {
	portraits := newPortraitCache()
	r := &Renderer{
		win: win, sprites: map[spriteKey]*Texture{}, stageAxis: map[spriteKey][2]int{},
		portraits: portraits, views: map[string]*packView{}, active: pack.Name(), load: load,
	}
	r.views[r.active] = newPackView(win, pack, portraits)
	return r
}

// Draw renders the current screen of f.
func (r *Renderer) Draw(f *game.Flow) {
	w := r.win
	r.portraits.bind(f)
	kind := packScreenOf(f)
	preview, sample := f.PreviewPack(), f.PreviewLifebar()
	if preview == "" {
		r.releaseCandidates()
	}
	if sample == nil {
		r.releaseSample()
	}
	if kind != r.packKind {
		r.packKind, r.packSince = kind, time.Now()
	}
	if kind != packNone && kind != packPause && kind != packMoveList {
		r.drawPack(f, kind)
		if sample != nil {
			r.drawSample(f, sample)
		}
		return
	}
	w.Rect(0, 0, VW, VH, darkBg)
	if f.Screen == game.ScreenMatch {
		r.drawMatch(f)
		if kind == packPause || kind == packMoveList {
			r.drawPack(f, kind)
		}
	}
}

// sampleShift moves the sample lifebar below the appearance page's menu.
const sampleShift = VH / 2

// drawSample draws the candidate lifebar over the appearance screen, its
// sprites uploaded on first use.
func (r *Renderer) drawSample(f *game.Flow, hud *game.HUDLifebar) {
	if r.sample == nil || r.sample.hud != hud {
		r.releaseSample()
		r.sample = newLifebarHUD(hud, f.FS)
	}
	ticks := int(time.Since(r.packSince).Milliseconds()) * game.TickRateHz / 1000
	// The lifebar sits at the top in a match; on the page it would cover the
	// heading, so it is shown in the empty lower half.
	r.win.BeginShift(sampleShift)
	r.sample.drawSample(r.win, ticks)
	r.win.EndShift()
}

func (r *Renderer) releaseSample() {
	if r.sample != nil {
		r.sample.delete()
		r.sample = nil
	}
}

// loadSprites decodes and uploads every sprite the match's animations
// reference, once, so the match loop never decodes.
func (r *Renderer) loadSprites(f *game.Flow) {
	for _, t := range r.sprites {
		t.Delete()
	}
	clear(r.sprites)
	r.loadedFor = f.Match
	r.behind, r.front = game.SplitLayers(f.Stage.Elements)
	r.stageScale = game.StageScale(f.Stage, VW)
	r.shift = game.StageGroundShift(f.Stage, r.stageScale, VH)
	if r.lifebar != nil {
		r.lifebar.tick = -1
	}
	r.loadStageSprites(f)
	r.loadSparkSprites(f)
	r.loadFaces(f)
	for i, fighter := range f.Fighters {
		file, err := openSprites(f.FS, fighter.SffPath)
		if err != nil {
			continue
		}
		for _, a := range fighter.Char.Animations {
			for _, fr := range a.Frames {
				k := spriteKey{i, fr.Group, fr.Image}
				if fr.IsBlank() || r.sprites[k] != nil {
					continue
				}
				px, wd, ht, err := sff.ResolveCharacterSpritePixels(file, fr.Group, fr.Image, fighter.Palette)
				if err != nil || wd == 0 || ht == 0 {
					continue
				}
				r.sprites[k] = NewTexture(px, wd, ht)
			}
		}
	}
}

// faceSprite is a fighter's lifebar portrait.
type faceSprite struct {
	tex  *Texture
	axis [2]int
}

// loadFaces uploads each fighter's portrait, the sprite the lifebar's
// face.spr names in its character's sheet, in its palette.
func (r *Renderer) loadFaces(f *game.Flow) {
	for i := range r.faces {
		if r.faces[i].tex != nil {
			r.faces[i].tex.Delete()
		}
		r.faces[i] = faceSprite{}
	}
	if f.Lifebar == nil {
		return
	}
	for i, fighter := range f.Fighters {
		ref := f.Lifebar.Face[i].FaceSprite
		if ref.Group < 0 {
			continue
		}
		file, err := openSprites(f.FS, fighter.SffPath)
		if err != nil {
			continue
		}
		px, wd, ht, err := sff.ResolveCharacterSpritePixels(file, ref.Group, ref.Image, fighter.Palette)
		if err != nil || wd == 0 || ht == 0 {
			continue
		}
		r.faces[i] = faceSprite{tex: NewTexture(px, wd, ht), axis: spriteAxis(file, ref.Group, ref.Image)}
	}
}

// spriteAxis is a sprite's axis in its sheet, 0,0 when it cannot be read.
func spriteAxis(file io.ReaderAt, group, image int) [2]int {
	groups, err := sff.Load(file)
	if err != nil {
		return [2]int{}
	}
	for _, g := range groups {
		for _, sp := range g.Sprites {
			if sp.Group == group && sp.Image == image {
				return [2]int{sp.AxisX, sp.AxisY}
			}
		}
	}
	return [2]int{}
}

// commonFXSheet is the spriteKey.fighter value of the common fightfx sheet.
const commonFXSheet = -3

// loadSparkSprites uploads the sprites of the common fightfx animations, with
// their axes, for the hit sparks.
func (r *Renderer) loadSparkSprites(f *game.Flow) {
	anims := f.Match.CommonAnimations()
	if len(anims) == 0 {
		return
	}
	file, err := openSprites(f.FS, path.Join(game.CommonFolder(f.FS), "fightfx.sff"))
	if err != nil {
		return
	}
	groups, err := sff.Load(file)
	if err != nil {
		return
	}
	if r.fxAxis == nil {
		r.fxAxis = map[spriteKey][2]int{}
	}
	for _, g := range groups {
		for _, sp := range g.Sprites {
			r.fxAxis[spriteKey{commonFXSheet, sp.Group, sp.Image}] = [2]int{sp.AxisX, sp.AxisY}
		}
	}
	for _, a := range anims {
		for _, fr := range a.Frames {
			k := spriteKey{commonFXSheet, fr.Group, fr.Image}
			if fr.IsBlank() || r.sprites[k] != nil {
				continue
			}
			px, wd, ht, err := sff.ResolveSpritePixels(file, fr.Group, fr.Image, nil)
			if err != nil || wd == 0 || ht == 0 {
				continue
			}
			r.sprites[k] = NewTexture(px, wd, ht)
		}
	}
}

// drawSparks draws the match's hit sparks over the stage, common ones from
// the fightfx sheet, own ones from their character's.
func (r *Renderer) drawSparks(f *game.Flow, groundY, camX float32) {
	m := f.Match
	for _, sp := range m.Sparks.Live {
		anims, sheet := m.CommonAnimations(), commonFXSheet
		if sp.Own {
			anims, sheet = f.Fighters[sp.Side].Char.Animations, int(sp.Side)
		}
		fr, ok := currentFrame(anims, sp.Anim, sp.Age)
		if !ok || fr.IsBlank() {
			continue
		}
		k := spriteKey{sheet, fr.Group, fr.Image}
		t := r.sprites[k]
		if t == nil {
			continue
		}
		axis, ok := r.fxAxis[k]
		if sp.Own {
			meta, found := spriteMeta(f.Fighters[sp.Side], fr)
			axis, ok = [2]int{meta.AxisX, meta.AxisY}, found
		}
		if !ok {
			continue
		}
		flip := sp.Facing == match.FacingLeft
		sx := VW/2 + float32(sp.X) - camX
		sy := fighterScreenY(groundY, float32(sp.Y))
		ox := float32(fr.X - axis[0])
		if flip {
			ox = -float32(fr.X) - float32(t.W-axis[0])
		}
		if additiveBlend(string(fr.Blend)) {
			r.win.SpriteAdd(t, sx+ox, sy+float32(fr.Y-axis[1]), flip)
			continue
		}
		r.win.Sprite(t, sx+ox, sy+float32(fr.Y-axis[1]), flip)
	}
}

// drawProjectiles draws each side's live helpers and projectiles with that
// side's own animations and sprites.
func (r *Renderer) drawProjectiles(f *game.Flow, groundY, camX float32) {
	for side := range f.Fighters {
		for _, h := range f.Match.Helpers(match.Side(side)) {
			c := &h.Context
			r.drawOwnAnim(f, side, c.Anim, c.AnimElapsed, angleDrawn(c.Position, c.Facing, c), groundY, camX)
		}
		for _, p := range f.Match.Projectiles(match.Side(side)) {
			r.drawOwnAnim(f, side, p.Anim, p.AnimTime, unscaled(p.Position, p.Facing), groundY, camX)
		}
	}
}

// drawGhosts draws side's afterimage ghosts behind it, oldest first, each
// tinted as its step asks.
func (r *Renderer) drawGhosts(f *game.Flow, side int, groundY, camX float32) {
	r.ghosts = f.Match.Ghosts(side, r.ghosts)
	for k := len(r.ghosts) - 1; k >= 0; k-- {
		g := r.ghosts[k]
		// A ghost of a thrown fighter is drawn from the thrower's
		// animations and sprites, as the fighter is.
		owner := int(g.Anim.Owner)
		fighter := f.Fighters[owner]
		fr, ok := currentFrame(fighter.Char.Animations, g.Anim.No, g.Anim.Time)
		if !ok || fr.IsBlank() {
			continue
		}
		t := r.sprites[spriteKey{owner, fr.Group, fr.Image}]
		sp, found := spriteMeta(fighter, fr)
		if t == nil || !found {
			continue
		}
		flip := g.Facing == match.FacingLeft
		ox := float32(fr.X - sp.AxisX)
		if flip {
			ox = -float32(fr.X) - float32(sp.Width-sp.AxisX)
		}
		sx := VW/2 + float32(g.Position.X) - camX
		sy := fighterScreenY(groundY, float32(g.Position.Y))
		r.win.SpriteTinted(t, sx+ox, sy+float32(fr.Y-sp.AxisY), flip, f.Match.AfterImages[side].Additive, g.Mul, g.Add)
	}
}

// superDarkness is how dark the stage is drawn: half during a darkening
// SuperPause (Ikemen GO's superbrightness 0.5), else not at all.
func superDarkness(s match.MatchState) float32 {
	if s.SuperTime > 0 && s.SuperDarken {
		return 0.5
	}
	return 0
}

// explodLayer picks which explods a drawExplods call draws.
type explodLayer int

const (
	explodsBehind explodLayer = iota // sprpriority below 0: behind the fighters
	explodsFront                     // the others, over the fighters
	explodsOnTop                     // ontop: over everything but the HUD
)

func (l explodLayer) has(e engine.Explod) bool {
	switch {
	case e.OnTop:
		return l == explodsOnTop
	case e.SprPriority < 0:
		return l == explodsBehind
	}
	return l == explodsFront
}

// drawExplods draws the explods of layer: common ones from the fightfx
// sheet, the others with their side's sprites.
// shortcut: vfacing (upside down) is not drawn.
func (r *Renderer) drawExplods(f *game.Flow, layer explodLayer, groundY, camX float32) {
	for side := range f.Fighters {
		for _, e := range f.Match.Explods(match.Side(side)) {
			if !layer.has(e) {
				continue
			}
			if e.Common {
				r.drawCommonAnim(f, e.Anim, e.AnimTime, placed{pos: e.Position, facing: e.Facing, scale: e.Scale, trans: e.Trans, alpha: e.Alpha}, groundY, camX)
				continue
			}
			r.drawOwnAnim(f, side, e.Anim, e.AnimTime, placed{pos: e.Position, facing: e.Facing, scale: e.Scale, trans: e.Trans, alpha: e.Alpha}, groundY, camX)
		}
	}
}

// drawCommonAnim draws the common fightfx animation anim, animTime ticks
// in, at a world position and facing.
func (r *Renderer) drawCommonAnim(f *game.Flow, anim, animTime int, at placed, groundY, camX float32) {
	fr, ok := currentFrame(f.Match.CommonAnimations(), anim, animTime)
	if !ok || fr.IsBlank() {
		return
	}
	k := spriteKey{commonFXSheet, fr.Group, fr.Image}
	t := r.sprites[k]
	axis, found := r.fxAxis[k]
	if t == nil || !found {
		return
	}
	r.blit(t, fr, axis, at, groundY, camX)
}

// fighterWidth is the width of the fighters' space, 320 units: the match's
// camera and the fighters are in it.
const fighterWidth = 320

// placed is where an animation is drawn: a world position, a facing and a
// scale.
type placed struct {
	pos    match.Position
	facing match.Facing
	scale  [2]float64
	// trans and alpha are an explod's trans type ("" for the frame's own
	// blend) and alpha.
	trans string
	alpha [2]int
	// angle is the counter-clockwise turn in degrees (an AngleDraw).
	angle float64
}

// angleDrawn is how a fighter or helper context is drawn: its AngleDraw's
// scale and angle while it runs one (negated facing left, char.go:13533-
// 13540), else at its own size.
func angleDrawn(pos match.Position, facing match.Facing, c *evaluator.Context) placed {
	at := unscaled(pos, facing)
	if !c.AngleDraw {
		return at
	}
	at.scale, at.angle = c.AngleDrawScale, c.Angle
	if facing == match.FacingLeft {
		at.angle = -at.angle
	}
	return at
}

// unscaled is a placement at pos and facing, at its own size.
func unscaled(pos match.Position, facing match.Facing) placed {
	return placed{pos: pos, facing: facing, scale: [2]float64{1, 1}}
}

// blit draws texture t of frame fr, whose sprite axis is axis, where at
// places it, scaled around the axis, additively when the frame asks.
func (r *Renderer) blit(t *Texture, fr air.Frame, axis [2]int, at placed, groundY, camX float32) {
	flip := at.facing == match.FacingLeft
	kx, ky := float32(at.scale[0]), float32(at.scale[1])
	sx := VW/2 + float32(at.pos.X) - camX
	sy := fighterScreenY(groundY, float32(at.pos.Y))
	ox := float32(fr.X - axis[0])
	if flip {
		ox = -float32(fr.X) - float32(t.W-axis[0])
	}
	x, y := sx+ox*kx, sy+float32(fr.Y-axis[1])*ky
	w, h := float32(t.W)*kx, float32(t.H)*ky
	blend, alpha := spriteBlend(at.trans, at.alpha, string(fr.Blend))
	r.win.SpriteRotated(t, x, y, w, h, flip, blend, alpha, sx, sy, float32(at.angle))
}

// spriteBlend is how a trans type (an explod's, or a fighter's Trans) mixes
// a sprite (paramTrans): add and
// add1 add it, addalpha draws it at its source alpha, sub subtracts it, none
// draws it plainly; without one, the frame's own blend applies.
// shortcut: add1's halved backdrop and addalpha's destination weight are not
// applied.
func spriteBlend(trans string, alpha [2]int, frameBlend string) (Blend, float32) {
	switch trans {
	case "add", "add1":
		return BlendAdd, 1
	case "addalpha":
		return BlendNormal, float32(alpha[0]) / 255
	case "sub", "subadd":
		return BlendSub, 1
	case "none", "default":
		return BlendNormal, 1
	}
	if additiveBlend(frameBlend) {
		return BlendAdd, 1
	}
	return BlendNormal, 1
}

// drawOwnAnim draws side's animation anim, animTime ticks in, at a world
// position and facing, with side's sprites.
func (r *Renderer) drawOwnAnim(f *game.Flow, side, anim, animTime int, at placed, groundY, camX float32) {
	fighter := f.Fighters[side]
	fr, ok := currentFrame(fighter.Char.Animations, anim, animTime)
	if !ok || fr.IsBlank() {
		return
	}
	t := r.sprites[spriteKey{side, fr.Group, fr.Image}]
	if t == nil {
		return
	}
	sp, ok := spriteMeta(fighter, fr)
	if !ok {
		return
	}
	r.blit(t, fr, [2]int{sp.AxisX, sp.AxisY}, at, groundY, camX)
}

// openSprites reads the sprite file name of fsys, to hand to the sff reader.
func openSprites(fsys fs.FS, name string) (*bytes.Reader, error) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// loadStageSprites uploads every sprite the stage's layers can show, from the
// stage's own sheet. A missing sheet or sprite just leaves its layers on the
// placeholder tile.
func (r *Renderer) loadStageSprites(f *game.Flow) {
	path := game.StageSpritePath(f.FS, f.StageDefPath, f.Stage)
	if path == "" {
		return
	}
	file, err := openSprites(f.FS, path)
	if err != nil {
		return
	}
	clear(r.stageAxis)
	groups, err := sff.Load(file)
	if err != nil {
		return
	}
	for _, g := range groups {
		for _, sp := range g.Sprites {
			r.stageAxis[spriteKey{stageSheet, sp.Group, sp.Image}] = [2]int{sp.AxisX, sp.AxisY}
		}
	}
	load := func(ref stage.SpriteRef) {
		k := spriteKey{stageSheet, ref.Group, ref.Image}
		if ref.IsBlank() || r.sprites[k] != nil {
			return
		}
		px, wd, ht, err := sff.ResolveSpritePixels(file, ref.Group, ref.Image, nil)
		if err != nil || wd == 0 || ht == 0 {
			return
		}
		r.sprites[k] = NewTexture(px, wd, ht)
	}
	for _, e := range f.Stage.Elements {
		load(e.Sprite)
		if e.Type == stage.BGElementAnim {
			for _, fr := range f.Stage.Animations[e.ActionNumber].Frames {
				load(fr.Sprite)
			}
		}
	}
}

// drawLayers draws the stage elements at idx (back-to-front) as they stand
// ticks into the round with the camera at camX view pixels, scaled from the
// stage's localcoord to the view and tiled as the element asks.
func (r *Renderer) drawLayers(f *game.Flow, idx []int, ticks int, camX float64) {
	for _, i := range idx {
		e := f.Stage.Elements[i]
		ref, ok := game.LayerSprite(f.Stage, e, ticks)
		if !ok {
			continue
		}
		k := spriteKey{stageSheet, ref.Group, ref.Image}
		axis := r.stageAxis[k]
		ox, oy := game.LayerOrigin(e, axis[0], axis[1], camX, r.stageScale, VW, r.shift)
		t := r.sprites[k]
		if t == nil {
			r.win.Rect(float32(ox), float32(oy), layerPlaceholderSize, layerPlaceholderSize, grey)
			continue
		}
		sclX, sclY := game.ElementScale(e)
		wd, ht := float64(t.W)*r.stageScale*sclX, float64(t.H)*r.stageScale*sclY
		x0, nx := game.TileRun(ox, wd+float64(e.TileSpacingX)*r.stageScale, VW, e.TileX)
		y0, ny := game.TileRun(oy, ht+float64(e.TileSpacingY)*r.stageScale, VH, e.TileY)
		for ty := range ny {
			for tx := range nx {
				x := x0 + float64(tx)*(wd+float64(e.TileSpacingX)*r.stageScale)
				y := y0 + float64(ty)*(ht+float64(e.TileSpacingY)*r.stageScale)
				if f.Match.StagePalFX.Active() {
					mul, add := f.Match.StagePalFX.Tint()
					r.win.SpriteTintedSized(t, float32(x), float32(y), float32(wd), float32(ht), false, false, mul, add)
					continue
				}
				r.win.SpriteSized(t, float32(x), float32(y), float32(wd), float32(ht), false)
			}
		}
	}
}

// currentFrame resolves which frame of the fighter's animation is showing.
func currentFrame(anims []air.Animation, animNo, animTime int) (air.Frame, bool) {
	for _, a := range anims {
		if a.Number != animNo || len(a.Frames) == 0 {
			continue
		}
		total := 0
		for i, fr := range a.Frames {
			if fr.Time < 0 {
				return fr, true
			}
			total += fr.Time
			_ = i
		}
		t := animTime
		if t >= total {
			loopLen := 0
			for _, fr := range a.Frames[a.LoopStart:] {
				loopLen += fr.Time
			}
			if loopLen <= 0 {
				return a.Frames[len(a.Frames)-1], true
			}
			start := total - loopLen
			t = start + (t-start)%loopLen
		}
		acc := 0
		for _, fr := range a.Frames {
			acc += fr.Time
			if t < acc {
				return fr, true
			}
		}
		return a.Frames[len(a.Frames)-1], true
	}
	return air.Frame{}, false
}

func (r *Renderer) drawMatch(f *game.Flow) {
	w := r.win
	m := f.Match
	if r.loadedFor != m {
		r.loadSprites(f)
	}
	groundY := float32(game.GroundLine(f.Stage, r.stageScale, r.shift, defaultGroundY))
	if len(f.Stage.Elements) == 0 { // a stage with no art keeps the flat backdrop
		w.Rect(0, 0, VW, groundY, skyBg)
		w.Rect(0, groundY, VW, VH-groundY, floor)
	}

	p1 := m.State.Fighters[match.SideP1]
	p2 := m.State.Fighters[match.SideP2]
	r.camX = m.CameraX() * VW / fighterWidth // the match's camera, in view pixels
	camX := float32(r.camX)
	// The screen shake moves the whole match view, not the HUD.
	w.BeginShift(float32(m.Shake.Offset() * VW / fighterWidth))
	// An EnvColor fills the screen instead of the stage, the fighters over it
	// only when under (system.go:3817-3830).
	env := m.Env.Active()
	if env {
		c := m.Env.Color
		w.Rect(0, 0, VW, VH, Color{float32(c[0]) / 255, float32(c[1]) / 255, float32(c[2]) / 255, 1})
	} else {
		r.drawLayers(f, r.behind, m.Ticks(), r.camX)
	}
	if a := superDarkness(m.State); a > 0 && !env {
		w.Rect(0, 0, VW, VH, Color{0, 0, 0, a})
	}
	if env && !m.Env.Under {
		w.EndShift()
		r.drawHUD(f)
		return
	}
	r.drawExplods(f, explodsBehind, groundY, camX)
	for i, fs := range []match.FighterState{p1, p2} {
		r.drawGhosts(f, i, groundY, camX)
		// An Offset moves the drawing, x not mirrored (char.go:10441-10445).
		off := m.FighterContext(i).DrawOffset
		sx := VW/2 + float32(fs.Position.X+off[0]) - camX
		sy := fighterScreenY(groundY, float32(fs.Position.Y)) + float32(off[1])
		flip := fs.Facing == match.FacingLeft
		drawn := false
		// A thrown fighter may show its thrower's animation, drawn with the
		// thrower's sprites.
		anim := m.Anim[i]
		owner := int(anim.Owner)
		if fr, ok := currentFrame(f.Fighters[owner].Char.Animations, anim.No, anim.Time); ok && !fr.IsBlank() {
			if t := r.sprites[spriteKey{owner, fr.Group, fr.Image}]; t != nil {
				if sp, ok := spriteMeta(f.Fighters[owner], fr); ok {
					ox := float32(fr.X - sp.AxisX)
					if flip {
						ox = -float32(fr.X) - float32(sp.Width-sp.AxisX)
					}
					x, y := sx+ox, sy+float32(fr.Y-sp.AxisY)
					trans, transAlpha := m.Trans(i)
					blend, alpha := spriteBlend(trans, transAlpha, string(fr.Blend))
					at := angleDrawn(fs.Position, fs.Facing, m.FighterContext(i))
					switch {
					case at.angle != 0 || at.scale != [2]float64{1, 1}:
						// shortcut: a rotated or scaled fighter is drawn without its PalFX tint.
						kx, ky := float32(at.scale[0]), float32(at.scale[1])
						w.SpriteRotated(t, sx+ox*kx, sy+float32(fr.Y-sp.AxisY)*ky, float32(t.W)*kx, float32(t.H)*ky, flip, blend, alpha, sx, sy, float32(at.angle))
					case m.PalFX[i].Active():
						mul, add := m.PalFX[i].Tint()
						w.SpriteTinted(t, x, y, flip, blend == BlendAdd, mul, add)
					case m.AllPalFX:
						mul, add := m.StagePalFX.Tint()
						w.SpriteTinted(t, x, y, flip, blend == BlendAdd, mul, add)
					default:
						w.SpriteBlended(t, x, y, float32(t.W), float32(t.H), flip, blend, alpha)
					}
					drawn = true
				}
			}
		}
		if !drawn { // no sprite available: a placeholder body keeps the fighter visible
			col := blue
			if i == 1 {
				col = red
			}
			w.Rect(sx-8, sy-60, 16, 60, col)
		}
	}
	r.drawProjectiles(f, groundY, camX)
	r.drawExplods(f, explodsFront, groundY, camX)
	if !env {
		r.drawLayers(f, r.front, m.Ticks(), r.camX)
	}
	r.drawSparks(f, groundY, camX)
	r.drawExplods(f, explodsOnTop, groundY, camX)
	w.EndShift()
	r.drawHUD(f)
}

// fighterScreenY is the view row of a fighter at height y (the engine's Y
// points up) over the ground line.
func fighterScreenY(groundY, y float32) float32 { return groundY - y }

func spriteMeta(fighter *game.Fighter, fr air.Frame) (sff.Sprite, bool) {
	sp, err := fighter.Char.ResolveSprite(fr)
	return sp, err == nil
}

func (r *Renderer) drawHUD(f *game.Flow) {
	switch {
	case f.HideLifebars:
	case f.Lifebar == nil:
		r.drawBuiltInBars(f)
	default:
		r.drawLifebar(f)
		if game.HasCalls(f.Lifebar.Lifebar) {
			return // the lifebar draws its own calls
		}
	}
	r.drawBanner(f.Match)
}

// drawLifebar draws the flow's lifebar, its sprites uploaded on first use.
func (r *Renderer) drawLifebar(f *game.Flow) {
	if r.lifebar == nil || r.lifebar.hud != f.Lifebar {
		if r.lifebar != nil {
			r.lifebar.delete()
		}
		r.lifebar = newLifebarHUD(f.Lifebar, f.FS)
	}
	r.lifebar.draw(r.win, f, r.faces)
}

// drawBuiltInBars is the HUD used when no lifebar is installed.
func (r *Renderer) drawBuiltInBars(f *game.Flow) {
	w := r.win
	m := f.Match
	for i := 0; i < 2; i++ {
		fs := m.State.Fighters[i]
		frac := float32(lifeFraction(fs.Health, m.LifeMax[i]))
		pow := float32(powerLevel{value: float64(fs.Power), max: float64(m.PowerMax[i])}.fraction())
		col := blue
		if i == 1 {
			col = red
		}
		const bw = 125
		x := float32(10)
		if i == 1 {
			x = VW - 10 - bw
		}
		w.Rect(x-1, 9, bw+2, 12, Rgb(0, 0, 0))
		bx, fw := x, bw*frac
		if i == 1 {
			bx = x + bw - fw // P2's bar drains toward the center
		}
		w.Rect(bx, 10, fw, 10, green)
		w.Rect(x, 24, bw*min(pow, 1), 4, gold)
		name := f.Fighters[i].Char.Name
		if i == 0 {
			w.Text(x, 31, 1, col, name)
		} else {
			w.Text(x+bw-float32(TextWidth(name)), 31, 1, col, name)
		}
		for k := 0; k < m.Progress.RoundsToWin(); k++ {
			c := panel
			if k < m.Progress.Wins[i] {
				c = gold
			}
			px := x + float32(k)*10
			if i == 1 {
				px = x + bw - 8 - float32(k)*10
			}
			w.Rect(px, 40, 8, 8, c)
		}
	}
	w.Text(VW/2-12, 10, 2, white, timerText(f))
}

// drawBanner shows the round number during the intros, "Fight!" as the
// fight starts and the outcome once the round is decided.
func (r *Renderer) drawBanner(m *game.Match) {
	if text := bannerText(m); text != "" {
		r.win.TextCentered(90, 3, gold, text)
	}
}

// bannerText is drawBanner's text, "" for none.
func bannerText(m *game.Match) string {
	return banner(m.LastRound.Outcome, m.Setup.Training, m.RoundState(), m.State.Round, m.State.RoundTimer, m.Setup.RoundTimerTicks())
}

// banner is the text for a round decided with outcome (OutcomeNone while
// fought) at RoundState state, round number roundNo, timer ticks left of
// total.
func banner(outcome round.Outcome, training bool, state, roundNo, timer, total int) string {
	switch {
	case outcome != round.OutcomeNone:
		return outcomeBanner(outcome)
	case training:
		return ""
	case state < 2:
		return fmt.Sprintf("Round %d", roundNo)
	case total > 0 && timer > total-game.TickRateHz:
		return "Fight!"
	}
	return ""
}

func outcomeBanner(outcome round.Outcome) string {
	switch outcome {
	case round.OutcomeKO:
		return "KO"
	case round.OutcomeDoubleKO:
		return "Double KO"
	case round.OutcomeTimeout:
		return "Time over"
	default:
		return "Draw"
	}
}
