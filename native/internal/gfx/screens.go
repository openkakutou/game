package gfx

import (
	"fmt"
	"os"

	"github.com/openkakutou/character/air"
	"github.com/openkakutou/engine/match"
	"github.com/openkakutou/engine/round"
	"github.com/openkakutou/game/native/internal/game"
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
	// behind/front are the stage's layer indexes, split once per match.
	behind, front []int
	// stageAxis is each loaded stage sprite's axis, from the stage's sheet.
	stageAxis map[spriteKey][2]int
	// The stage camera and view mapping, set up once per match.
	cam               game.StageCamera
	camX              float64
	camTick           int
	stageScale, shift float64
	// lifebar draws the flow's lifebar, when it has one.
	lifebar *lifebarHUD
}

// NewRenderer builds a renderer for win.
func NewRenderer(win *Window) *Renderer {
	return &Renderer{win: win, sprites: map[spriteKey]*Texture{}, stageAxis: map[spriteKey][2]int{}}
}

// Draw renders the current screen of f.
func (r *Renderer) Draw(f *game.Flow) {
	w := r.win
	w.Rect(0, 0, VW, VH, darkBg)
	switch f.Screen {
	case game.ScreenSelect:
		r.drawList(f, "Select your fighter", fmt.Sprintf("Player %d", f.Picking+1), names(f.Chars), f.Cursor)
	case game.ScreenStage:
		r.drawList(f, "Select a stage", "", names(f.Stages), f.Cursor)
	case game.ScreenSetup:
		r.drawSetup(f)
	case game.ScreenMatch:
		r.drawMatch(f)
		if f.Paused {
			r.drawPause(f)
		}
	case game.ScreenResult:
		r.drawResult(f)
	case game.ScreenError:
		w.TextCentered(70, 2, red, "Something went wrong")
		r.wrapped(100, f.Message)
		w.TextCentered(200, 1, grey, "Press enter to quit")
	}
}

// drawPause dims the frozen match and lists the pause menu, titled PAUSE as
// Ikemen GO's default [Pause Menu].
func (r *Renderer) drawPause(f *game.Flow) {
	w := r.win
	w.Rect(0, 0, VW, VH, Color{0, 0, 0, 0.6})
	w.TextCentered(70, 3, white, "PAUSE")
	for i, item := range f.PauseItems() {
		c := grey
		if i == f.PauseCursor {
			c = gold
		}
		w.TextCentered(float32(110+i*16), 1, c, item.Label())
	}
}

func names(es []game.Entry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Name
	}
	return out
}

func (r *Renderer) wrapped(y float32, msg string) {
	const perLine = 50
	runes := []rune(msg)
	for len(runes) > 0 {
		n := min(perLine, len(runes))
		r.win.TextCentered(y, 1, white, string(runes[:n]))
		runes = runes[n:]
		y += 12
	}
}

func (r *Renderer) drawList(f *game.Flow, title, sub string, items []string, cursor int) {
	w := r.win
	w.TextCentered(16, 2, gold, title)
	if sub != "" {
		col := blue
		if f.Picking == 1 {
			col = red
		}
		w.TextCentered(40, 1.5, col, sub)
	}
	if f.Screen == game.ScreenSelect {
		w.Text(8, 60, 1, blue, "P1: "+f.Chars[f.Picks[0]].Name)
		if f.Picking == 1 {
			w.Text(8, 72, 1, red, "P2: choosing")
		}
	}
	const rows = 9
	first := max(0, min(cursor-rows/2, len(items)-rows))
	for i := first; i < min(len(items), first+rows); i++ {
		y := float32(90 + (i-first)*14)
		if i == cursor {
			w.Rect(70, y-3, 180, 13, panel)
			w.Text(76, y, 1, gold, ">")
			w.Text(90, y, 1, white, items[i])
		} else {
			w.Text(90, y, 1, grey, items[i])
		}
	}
	w.TextCentered(222, 1, grey, "Arrows/pad move  Enter/A confirm  Esc/B back")
}

func (r *Renderer) drawSetup(f *game.Flow) {
	w := r.win
	w.TextCentered(16, 2, gold, "Match setup")
	s := f.Setup()
	time := "No limit"
	if s.TimeLimitSeconds > 0 {
		time = fmt.Sprintf("%d seconds", s.TimeLimitSeconds)
	}
	p2 := "Human"
	if s.CPU[1] {
		p2 = "CPU"
	}
	rows := []string{fmt.Sprintf("Rounds   < best of %d >", s.Rounds), "Time     < " + time + " >", "Player 2 < " + p2 + " >", "Start match"}
	for i, text := range rows {
		y := float32(80 + i*24)
		col := grey
		if i == f.SetupRow {
			w.Rect(50, y-5, 220, 18, panel)
			col = white
		}
		w.Text(60, y, 1.5, col, text)
	}
	w.TextCentered(222, 1, grey, "Up/down row  Left/right change  Enter")
}

func (r *Renderer) drawResult(f *game.Flow) {
	w := r.win
	col := blue
	if f.ResultWinner == match.SideP2 {
		col = red
	}
	w.TextCentered(70, 3, col, fmt.Sprintf("Player %d wins", int(f.ResultWinner)+1))
	p := f.Match.Progress
	w.TextCentered(120, 2, white, fmt.Sprintf("%d - %d", p.Wins[0], p.Wins[1]))
	w.TextCentered(170, 1, grey, "Enter: rematch   Esc: back to selection")
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
	r.cam = game.NewStageCamera(f.Stage, game.CameraTension(f.Stage), r.stageScale, VW)
	r.camTick = -1
	if r.lifebar != nil {
		r.lifebar.tick = -1
	}
	r.loadStageSprites(f)
	for i, fighter := range f.Fighters {
		file, err := os.Open(fighter.SffPath)
		if err != nil {
			continue
		}
		for _, a := range fighter.Char.Animations {
			for _, fr := range a.Frames {
				k := spriteKey{i, fr.Group, fr.Image}
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
		file.Close()
	}
}

// loadStageSprites uploads every sprite the stage's layers can show, from the
// stage's own sheet. A missing sheet or sprite just leaves its layers on the
// placeholder tile.
func (r *Renderer) loadStageSprites(f *game.Flow) {
	path := game.StageSpritePath(f.ContentRoot, f.StageDefPath, f.Stage)
	if path == "" {
		return
	}
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
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
	r.followFighters(m.Ticks(), p1.Position.X, p2.Position.X)
	camX := float32(r.camX)
	r.drawLayers(f, r.behind, m.Ticks(), r.camX)
	for i, fs := range []match.FighterState{p1, p2} {
		sx := VW/2 + float32(fs.Position.X) - camX
		sy := groundY + float32(fs.Position.Y)
		flip := fs.Facing == match.FacingLeft
		drawn := false
		if fr, ok := currentFrame(f.Fighters[i].Char.Animations, m.AnimNo[i], m.AnimTime[i]); ok && !fr.IsBlank() {
			if t := r.sprites[spriteKey{i, fr.Group, fr.Image}]; t != nil {
				if sp, ok := spriteMeta(f.Fighters[i], fr); ok {
					ox := float32(fr.X - sp.AxisX)
					if flip {
						ox = -float32(fr.X) - float32(sp.Width-sp.AxisX)
					}
					w.Sprite(t, sx+ox, sy+float32(fr.Y-sp.AxisY), flip)
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
	r.drawLayers(f, r.front, m.Ticks(), r.camX)
	r.drawHUD(f)
}

// followFighters moves the camera once per simulated tick, snapping to the
// fighters on a round's first tick (Ikemen GO camera.go:210).
func (r *Renderer) followFighters(tick int, x1, x2 float64) {
	if tick == r.camTick {
		return
	}
	snap := tick < r.camTick || r.camTick < 0
	if snap {
		r.camX = r.cam.Start()
	}
	r.camX = r.cam.Follow(r.camX, min(x1, x2), max(x1, x2), snap)
	r.camTick = tick
}

func spriteMeta(fighter *game.Fighter, fr air.Frame) (sff.Sprite, bool) {
	sp, err := fighter.Char.ResolveSprite(fr)
	return sp, err == nil
}

func (r *Renderer) drawHUD(f *game.Flow) {
	if f.Lifebar == nil {
		r.drawBuiltInBars(f)
	} else {
		r.drawLifebar(f)
	}
	r.drawBanner(f.Match)
}

// drawLifebar draws the flow's lifebar, its sprites uploaded on first use.
func (r *Renderer) drawLifebar(f *game.Flow) {
	if r.lifebar == nil || r.lifebar.hud != f.Lifebar {
		if r.lifebar != nil {
			r.lifebar.delete()
		}
		r.lifebar = newLifebarHUD(f.Lifebar)
	}
	r.lifebar.draw(r.win, f)
}

// drawBuiltInBars is the HUD used when no lifebar is installed.
func (r *Renderer) drawBuiltInBars(f *game.Flow) {
	w := r.win
	m := f.Match
	for i := 0; i < 2; i++ {
		fs := m.State.Fighters[i]
		frac := float32(max(0, fs.Health)) / game.DefaultHealth
		pow := float32(fs.Power) / game.MaxPower
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

// drawBanner shows the round number as a round starts and its outcome as
// it ends.
func (r *Renderer) drawBanner(m *game.Match) {
	w := r.win
	switch {
	case m.RoundEnded():
		w.TextCentered(90, 3, gold, roundBanner(m))
	case m.State.RoundTimer > m.Setup.RoundTimerTicks()-game.TickRateHz && m.Setup.RoundTimerTicks() > 0:
		w.TextCentered(90, 3, gold, fmt.Sprintf("Round %d", m.State.Round))
	}
}

func roundBanner(m *game.Match) string {
	switch m.LastRound.Outcome {
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
