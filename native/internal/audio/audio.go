// Package audio plays what the game package describes: the stage music, the
// fighters' PlaySnd effects, and the screenpack's menu music and sounds,
// through SDL2_mixer. Any audio failure means
// silence plus one logged warning, never a crash or a stall.
package audio

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"runtime"

	"github.com/openkakutou/engine/v2/match"
	"github.com/openkakutou/game/native/internal/game"
	"github.com/openkakutou/game/native/internal/ui"
	"github.com/veandco/go-sdl2/mix"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	sampleRate   = 44100
	mixChannels  = game.WavChannels // maximum simultaneous fighter effects, Ikemen GO's default
	menuChannels = 4                // maximum simultaneous menu sounds
	// The two channel groups: a match ending or starting halts the fighters'
	// effects only, so the menu sound that caused it is heard.
	fighterGroup = 1
	menuGroup    = 2
)

// chunkKey identifies one character sample, or one common sample when
// common is true (side is then ignored).
type chunkKey struct {
	side            match.Side
	group, sample   int
	common, lifebar bool
}

// Player owns the mixer. The zero value and a nil *Player are silent no-ops,
// as is a Player whose mixer could not be opened.
type Player struct {
	open  bool
	warnf func(format string, a ...any)
	// infof logs what loaded for the first time (menu music and sounds).
	infof func(format string, a ...any)
	// pack names the menu music and sounds; nil plays none.
	pack    *ui.Pack
	volumes game.Audio
	// menuMusic is what the menu music is, while menuPlaying: the screens
	// naming the same file keep it playing.
	menuMusic   ui.MenuMusic
	menuPlaying bool
	// menuFailed is a menu music that could not be loaded: it is not tried
	// again (nor warned about) until a match has come and gone.
	menuFailed   ui.Asset
	menuFailedOK bool
	// menuChunks are the pack's menu sounds, loaded on first use (a nil chunk
	// records a failed load, warned once); menuCues is the drained queue.
	menuChunks map[ui.Asset]*mix.Chunk
	menuCues   []game.MenuCue
	owner      *game.Match
	music      *mix.Music
	// musicData is the music file's bytes, which the mixer streams from.
	musicData []byte
	chunks    map[chunkKey]*mix.Chunk // a nil chunk records a failed load (warned once)
	cues      []game.Cue
	dropped   int // cues refused for lack of a free channel
	// slots mirrors the fighter channels 0..mixChannels-1 for RequestSlot;
	// started numbers the effects in play order.
	slots   []game.SoundSlot
	started int64
	// loopRestarts counts the menu track's restarts from its loopStartMs.
	loopRestarts int
	// paused is set while the music and effects are held for the pause menu.
	paused bool
	// NoMusic keeps the stage music off (-nomusic); effects still play.
	NoMusic bool
}

// openAudio is mix.OpenAudio, replaceable so tests can exercise the failure path.
var openAudio = mix.OpenAudio

// New opens the mixer. On failure it logs one warning and returns a silent Player.
func New() *Player {
	p := &Player{warnf: log.Printf, infof: log.Printf, chunks: map[chunkKey]*mix.Chunk{}, menuChunks: map[ui.Asset]*mix.Chunk{}}
	if err := mix.Init(mix.INIT_OGG | mix.INIT_MP3 | mix.INIT_FLAC); err != nil {
		p.warnf("audio: some music decoders are unavailable: %v", err)
	}
	if err := openAudio(sampleRate, mix.DEFAULT_FORMAT, 2, 2048); err != nil {
		p.warnf("audio: cannot open the mixer, playing without sound: %v", err)
		mix.Quit()
		return p
	}
	mix.AllocateChannels(mixChannels + menuChannels)
	mix.GroupChannels(0, mixChannels-1, fighterGroup)
	mix.GroupChannels(mixChannels, mixChannels+menuChannels-1, menuGroup)
	p.open = true
	return p
}

// SetPack gives the player the screenpack whose menu music and sounds it plays.
// Another pack than the one in use releases the old one's loaded sounds and
// stops its music (the next screen starts the new pack's).
func (p *Player) SetPack(pack *ui.Pack) {
	if p == nil || pack == p.pack {
		return
	}
	p.pack = pack
	p.freeMenuChunks()
	p.menuFailedOK = false
	if p.menuPlaying {
		p.stopMusic()
	}
}

// freeMenuChunks releases every loaded menu sound.
func (p *Player) freeMenuChunks() {
	for k, c := range p.menuChunks {
		if c != nil {
			c.Free()
		}
		delete(p.menuChunks, k)
	}
}

// SetVolumes applies the settings' volumes: the master multiplies the music
// and the effects volume of every channel.
func (p *Player) SetVolumes(a game.Audio) {
	if p == nil || !p.open {
		return
	}
	p.volumes = a
	p.applyMusicVolume()
	mix.Volume(-1, mixerVolume(a.Master, a.Effects))
}

// applyMusicVolume sets the music volume: master times music, and for the
// menu music times the screenpack's own volume for it.
func (p *Player) applyMusicVolume() {
	v := mixerVolume(p.volumes.Master, p.volumes.Music)
	if p.menuPlaying {
		v = int(float32(v) * p.menuMusic.Volume)
	}
	mix.VolumeMusic(v)
}

// mixerVolume is the mixer's 0 to 128 volume of two percentages multiplied.
func mixerVolume(master, part int) int {
	return mix.MAX_VOLUME * master * part / 10000
}

// Driver names the SDL audio driver in use, or "off" when the mixer is not open.
func (p *Player) Driver() string {
	if p == nil || !p.open {
		return "off"
	}
	return sdl.GetCurrentAudioDriver()
}

// Update applies one frame: it starts or stops the music when the match owner
// changes, then plays the owner's pending effects.
func (p *Player) Update(f *game.Flow) {
	if p == nil || !p.open {
		return
	}
	next, path, changed := game.MusicChange(p.owner, f)
	if changed {
		p.stopAll()
		p.menuFailedOK = false
		p.owner = next
		if path != "" && !p.NoMusic {
			p.startMusic(f.FS, path)
		}
	}
	p.playMenuCues(f)
	if p.owner == nil {
		_, _, menu := f.Music()
		p.syncMenuMusic(menu)
		return
	}
	p.holdForPause(f.Paused)
	p.cues = p.owner.DrainCues(p.cues)
	for _, c := range p.cues {
		if c.Stop {
			p.stopCue(c)
			continue
		}
		p.playCue(f, c)
	}
}

// holdForPause pauses the music and every playing effect when the pause menu
// opens, and resumes them where they were when it closes.
func (p *Player) holdForPause(paused bool) {
	if paused == p.paused {
		return
	}
	p.paused = paused
	if paused {
		mix.PauseMusic()
		mix.Pause(-1)
		return
	}
	mix.ResumeMusic()
	mix.Resume(-1)
}

func (p *Player) stopAll() {
	p.holdForPause(false)
	p.stopMusic()
	mix.HaltGroup(fighterGroup)
	p.freeChunks()
}

// stopMusic halts and frees the music, whoever owns it; effects play on.
func (p *Player) stopMusic() {
	p.menuPlaying = false
	p.applyMusicVolume()
	mix.HaltMusic()
	if p.music != nil {
		p.music.Free()
		p.music = nil
		p.musicData = nil
	}
}

func (p *Player) freeChunks() {
	for k, c := range p.chunks {
		if c != nil {
			c.Free()
		}
		delete(p.chunks, k)
	}
}

func (p *Player) startMusic(fsys fs.FS, path string) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		p.warnf("audio: cannot load music %q: %v", path, err)
		return
	}
	p.playMusic(path, data, -1)
}

// playMusic loads a music file's bytes and plays it loops times (-1 for
// ever); any failure is one warning and silence.
func (p *Player) playMusic(path string, data []byte, loops int) {
	rw, err := sdl.RWFromMem(data)
	if err != nil {
		p.warnf("audio: cannot load music %q: %v", path, err)
		return
	}
	m, err := mix.LoadMUSRW(rw, 1) // the mixer frees rw with the music
	if err != nil {
		p.warnf("audio: cannot load music %q: %v", path, err)
		return
	}
	if err := m.Play(loops); err != nil {
		m.Free()
		p.warnf("audio: cannot play music %q: %v", path, err)
		return
	}
	p.music = m
	p.musicData = data
}

// syncMenuMusic makes the music the one the pack gives the menu screen: it
// starts a track when none plays or the screen names another file, lets a
// track go on when the new screen names the same file, and stops it when the
// screen has no music (music null) or there is no screen.
// A track with loopStartMs plays once, and restartLoop starts it again from
// that position each time it ends.
func (p *Player) syncMenuMusic(screen string) {
	want, ok := ui.MenuMusic{}, false
	if p.pack != nil && screen != "" && !p.NoMusic {
		want, ok = p.pack.Music(screen)
	}
	if !ok {
		if p.menuPlaying {
			p.stopMusic()
		}
		return
	}
	if p.menuFailedOK && p.menuFailed == want.Asset {
		return
	}
	if p.menuPlaying && p.menuMusic.Asset == want.Asset {
		if p.menuMusic != want {
			p.menuMusic = want
			p.applyMusicVolume()
		}
		p.restartLoop()
		return
	}
	p.stopMusic()
	data, err := p.readAsset(want.Asset)
	if err != nil {
		p.warnf("audio: cannot load music %q: %v", want.Asset.Path, err)
		p.menuFailed, p.menuFailedOK = want.Asset, true
		return
	}
	loops := -1
	if want.LoopStartMs > 0 {
		loops = 1 // restartLoop plays it again from loopStartMs
	}
	p.playMusic(want.Asset.Path, data, loops)
	if p.music == nil {
		p.menuFailed, p.menuFailedOK = want.Asset, true
		return
	}
	p.menuPlaying, p.menuMusic = true, want
	p.applyMusicVolume()
	p.infof("audio: menu music loaded: %s", want.Asset.Path)
}

// restartLoop plays the menu track again from its loopStartMs once it has
// ended (the format's loop point, screenpack docs/format.md "music").
func (p *Player) restartLoop() {
	if p.menuMusic.LoopStartMs <= 0 || p.music == nil || mix.PlayingMusic() {
		return
	}
	if err := p.music.FadeInPos(1, 0, float64(p.menuMusic.LoopStartMs)/1000); err != nil {
		p.warnf("audio: cannot loop music %q: %v", p.menuMusic.Asset.Path, err)
		p.menuMusic.LoopStartMs = 0 // warned once; the track stays silent
		return
	}
	p.loopRestarts++
}

// readAsset reads a pack file.
func (p *Player) readAsset(a ui.Asset) ([]byte, error) {
	file, err := p.pack.Open(a)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

// playMenuCues plays the menu sounds the flow queued, whatever screen is
// shown: each is the pack's file for that sound on that screen, loaded the
// first time it is heard.
func (p *Player) playMenuCues(f *game.Flow) {
	p.menuCues = f.DrainMenuCues(p.menuCues)
	if p.pack == nil {
		return
	}
	for _, c := range p.menuCues {
		asset, ok := p.pack.Sound(c.Screen, string(c.Sound))
		if !ok {
			continue
		}
		chunk, seen := p.menuChunks[asset]
		if !seen {
			chunk = p.loadMenuChunk(asset)
			p.menuChunks[asset] = chunk
		}
		if chunk == nil {
			continue
		}
		channel := mix.GroupAvailable(menuGroup)
		if channel < 0 {
			channel = mix.GroupOldest(menuGroup) // all busy: the oldest sound makes room
		}
		chunk.Play(channel, 0)
	}
}

func (p *Player) loadMenuChunk(asset ui.Asset) *mix.Chunk {
	data, err := p.readAsset(asset)
	if err != nil {
		p.warnf("audio: cannot load sound %q: %v", asset.Path, err)
		return nil
	}
	rw, err := sdl.RWFromMem(data)
	if err != nil {
		p.warnf("audio: cannot load sound %q: %v", asset.Path, err)
		return nil
	}
	chunk, err := mix.LoadWAVRW(rw, true)
	runtime.KeepAlive(data) // the RWops reads data until LoadWAVRW returns
	if err != nil {
		p.warnf("audio: cannot load sound %q: %v", asset.Path, err)
		return nil
	}
	p.infof("audio: menu sound loaded: %s", asset.Path)
	return chunk
}

func (p *Player) playCue(f *game.Flow, c game.Cue) {
	key := chunkKey{c.Side, c.Group, c.Sample, c.Common, c.Lifebar}
	if c.Common {
		key.side = 0
	}
	chunk, seen := p.chunks[key]
	if !seen {
		var err error
		if chunk, err = loadChunk(f, c); err != nil {
			p.warnf("audio: skipping sound %d,%d of player %d: %v", c.Group, c.Sample, int(c.Side)+1, err)
			chunk = nil
		}
		p.chunks[key] = chunk
	}
	if chunk == nil {
		return
	}
	channel := p.requestChannel(c)
	if channel < 0 {
		p.dropped++ // lowpriority on a busy channel, or nothing to cut
		return
	}
	mix.HaltChannel(channel)
	if _, err := chunk.Play(channel, 0); err != nil {
		p.dropped++
		return
	}
	p.started++
	p.slots[channel] = game.SoundSlot{Side: c.Side, ChannelNo: c.Channel, Playing: true, Started: p.started}
}

// stopCue is a StopSnd (stopSnd.Run): every fighter sound for channel -1,
// else the side's sound on that channel.
func (p *Player) stopCue(c game.Cue) {
	if c.Channel < 0 {
		mix.HaltGroup(fighterGroup)
		return
	}
	for i, s := range p.slots {
		if s.Side == c.Side && s.ChannelNo == c.Channel {
			mix.HaltChannel(i)
		}
	}
}

// requestChannel returns the fighter channel the cue plays on, -1 to drop it
// (game.RequestSlot), reading which channels still play from the mixer.
func (p *Player) requestChannel(c game.Cue) int {
	if p.slots == nil {
		p.slots = make([]game.SoundSlot, mixChannels)
		for i := range p.slots {
			p.slots[i].ChannelNo = -1
		}
	}
	for i := range p.slots {
		p.slots[i].Playing = mix.Playing(i) != 0
	}
	return game.RequestSlot(p.slots, c.Side, c.Channel, c.LowPriority)
}

func loadChunk(f *game.Flow, c game.Cue) (*mix.Chunk, error) {
	snd, err := f.FindCueSound(c)
	if err != nil {
		return nil, err
	}
	wav := game.WAVBytes(snd)
	if wav == nil {
		return nil, fmt.Errorf("sample has no playable PCM")
	}
	rw, err := sdl.RWFromMem(wav)
	if err != nil {
		return nil, err
	}
	chunk, err := mix.LoadWAVRW(rw, true)
	runtime.KeepAlive(wav) // the RWops reads wav until LoadWAVRW returns
	return chunk, err
}

// Close stops everything and releases the mixer. It is safe to call twice.
func (p *Player) Close() {
	if p == nil || !p.open {
		return
	}
	p.stopAll()
	p.freeMenuChunks()
	mix.HaltChannel(-1)
	p.owner = nil
	p.open = false
	mix.CloseAudio()
	mix.Quit()
}
