// Package audio plays what the game package describes: the stage music and
// the fighters' PlaySnd effects, through SDL2_mixer. Any audio failure means
// silence plus one logged warning, never a crash or a stall.
package audio

import (
	"fmt"
	"io/fs"
	"log"
	"runtime"

	"github.com/openkakutou/engine/v2/match"
	"github.com/openkakutou/game/native/internal/game"
	"github.com/veandco/go-sdl2/mix"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	sampleRate  = 44100
	mixChannels = 16 // maximum simultaneous effects
)

// chunkKey identifies one character sample.
type chunkKey struct {
	side          match.Side
	group, sample int
}

// Player owns the mixer. The zero value and a nil *Player are silent no-ops,
// as is a Player whose mixer could not be opened.
type Player struct {
	open  bool
	warnf func(format string, a ...any)
	owner *game.Match
	music *mix.Music
	// musicData is the music file's bytes, which the mixer streams from.
	musicData []byte
	chunks    map[chunkKey]*mix.Chunk // a nil chunk records a failed load (warned once)
	cues      []game.Cue
	dropped   int // cues refused for lack of a free channel
	// paused is set while the music and effects are held for the pause menu.
	paused bool
	// NoMusic keeps the stage music off (-nomusic); effects still play.
	NoMusic bool
}

// openAudio is mix.OpenAudio, replaceable so tests can exercise the failure path.
var openAudio = mix.OpenAudio

// New opens the mixer. On failure it logs one warning and returns a silent Player.
func New() *Player {
	p := &Player{warnf: log.Printf, chunks: map[chunkKey]*mix.Chunk{}}
	if err := mix.Init(mix.INIT_OGG | mix.INIT_MP3 | mix.INIT_FLAC); err != nil {
		p.warnf("audio: some music decoders are unavailable: %v", err)
	}
	if err := openAudio(sampleRate, mix.DEFAULT_FORMAT, 2, 2048); err != nil {
		p.warnf("audio: cannot open the mixer, playing without sound: %v", err)
		mix.Quit()
		return p
	}
	mix.AllocateChannels(mixChannels)
	p.open = true
	return p
}

// SetVolumes applies the settings' volumes: the master multiplies the music
// and the effects volume of every channel.
func (p *Player) SetVolumes(a game.Audio) {
	if p == nil || !p.open {
		return
	}
	mix.VolumeMusic(mixerVolume(a.Master, a.Music))
	mix.Volume(-1, mixerVolume(a.Master, a.Effects))
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
		p.owner = next
		if path != "" && !p.NoMusic {
			p.startMusic(f.FS, path)
		}
	}
	if p.owner == nil {
		return
	}
	p.holdForPause(f.Paused)
	p.cues = p.owner.DrainCues(p.cues)
	for _, c := range p.cues {
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
	mix.HaltMusic()
	mix.HaltChannel(-1)
	if p.music != nil {
		p.music.Free()
		p.music = nil
		p.musicData = nil
	}
	p.freeChunks()
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
	if err := m.Play(-1); err != nil {
		m.Free()
		p.warnf("audio: cannot play music %q: %v", path, err)
		return
	}
	p.music = m
	p.musicData = data
}

func (p *Player) playCue(f *game.Flow, c game.Cue) {
	key := chunkKey{c.Side, c.Group, c.Sample}
	chunk, seen := p.chunks[key]
	if !seen {
		var err error
		if chunk, err = loadChunk(f.Fighters[c.Side], c); err != nil {
			p.warnf("audio: skipping sound %d,%d of player %d: %v", c.Group, c.Sample, int(c.Side)+1, err)
			chunk = nil
		}
		p.chunks[key] = chunk
	}
	if chunk == nil {
		return
	}
	if _, err := chunk.Play(-1, 0); err != nil {
		p.dropped++ // every channel is busy: drop the effect silently
	}
}

func loadChunk(fighter *game.Fighter, c game.Cue) (*mix.Chunk, error) {
	snd, ok := fighter.FindSound(c.Group, c.Sample)
	if !ok {
		return nil, fmt.Errorf("no such sample")
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
	p.owner = nil
	p.open = false
	mix.CloseAudio()
	mix.Quit()
}
