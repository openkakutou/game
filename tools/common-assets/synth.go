package main

import (
	"encoding/binary"
	"math"
)

const sampleRate = 22050

// voice is a family of procedural sounds; weight (0 to 4) scales how low,
// long and loud it is.
type voice func(weight, variant int) []float64

// namedSounds gives the sounds the sources describe their voice and weight:
// the MUGEN docs' hitsound 5,0 (light) to 5,4 (painful whack), guardsound 6,0,
// and the ground-bounce sounds 7,0 to 7,2 of the common states
// (common1.cns:1534).
var namedSounds = map[[2]int]struct {
	voice  voice
	weight int
}{
	{5, 0}: {thump, 0}, {5, 1}: {thump, 1}, {5, 2}: {thump, 2}, {5, 3}: {thump, 3}, {5, 4}: {thump, 4},
	{6, 0}: {clank, 2},
	{7, 0}: {bounce, 0}, {7, 1}: {bounce, 2}, {7, 2}: {bounce, 4},
}

// otherVoices play the sounds the sources do not describe: the voice and the
// variant come from the (group, sample) key so every sound is distinct.
var otherVoices = []voice{swing, zap, chime, click}

func soundFor(key [2]int) []float64 {
	named, ok := namedSounds[key]
	if ok {
		return named.voice(named.weight, 0)
	}
	index := key[0]*17 + key[1]
	return otherVoices[index%len(otherVoices)](index/len(otherVoices)%5, index)
}

// noise returns deterministic white noise from a seed.
func noise(seed uint32) func() float64 {
	state := seed*2654435761 + 1
	return func() float64 {
		state ^= state << 13
		state ^= state >> 17
		state ^= state << 5
		return float64(state)/float64(1<<31) - 1
	}
}

func envelope(i, n int, decay float64) float64 {
	return math.Exp(-decay * float64(i) / float64(n))
}

// thump is a falling sine body with a noise click, heavier and longer as
// weight grows.
func thump(weight, _ int) []float64 {
	n := sampleRate * (10 + 3*weight) / 100
	rand := noise(uint32(100 + weight))
	out := make([]float64, n)
	phase := 0.0
	for i := range out {
		freq := (260 - 30*float64(weight)) * envelope(i, n, 1.2)
		phase += 2 * math.Pi * freq / sampleRate
		click := rand() * envelope(i, n, 14)
		out[i] = (0.8*math.Sin(phase)*envelope(i, n, 4) + (0.25+0.1*float64(weight))*click) * (0.55 + 0.1*float64(weight))
	}
	return out
}

// bounce is a dull low thud, the sound of a body landing.
func bounce(weight, _ int) []float64 {
	n := sampleRate * (14 + 3*weight) / 100
	out := make([]float64, n)
	phase := 0.0
	for i := range out {
		freq := (120 - 12*float64(weight)) * envelope(i, n, 1.5)
		phase += 2 * math.Pi * freq / sampleRate
		out[i] = math.Sin(phase) * envelope(i, n, 5) * (0.5 + 0.1*float64(weight))
	}
	return out
}

// clank is a short metallic ring: inharmonic partials with a fast decay.
func clank(weight, _ int) []float64 {
	n := sampleRate * 25 / 100
	partials := []float64{1, 2.76, 5.4, 8.93}
	out := make([]float64, n)
	for i := range out {
		sum := 0.0
		for k, ratio := range partials {
			sum += math.Sin(2*math.Pi*(700+60*float64(weight))*ratio*float64(i)/sampleRate) * envelope(i, n, 6+3*float64(k))
		}
		out[i] = sum * 0.22
	}
	return out
}

// swing is band-limited noise sweeping up then down.
func swing(weight, variant int) []float64 {
	n := sampleRate * (16 + 2*weight) / 100
	rand := noise(uint32(variant + 7))
	out := make([]float64, n)
	low := 0.0
	for i := range out {
		position := float64(i) / float64(n)
		alpha := 0.05 + 0.3*math.Sin(math.Pi*position)
		low += alpha * (rand() - low)
		out[i] = low * 2.2 * math.Sin(math.Pi*position)
	}
	return out
}

// zap is a quick rising tone.
func zap(weight, variant int) []float64 {
	n := sampleRate * (14 + 2*weight) / 100
	out := make([]float64, n)
	phase := 0.0
	base := 300 + 40*float64(variant%7)
	for i := range out {
		freq := base * (1 + 3*float64(i)/float64(n))
		phase += 2 * math.Pi * freq / sampleRate
		out[i] = 0.5 * math.Sin(phase) * envelope(i, n, 3)
	}
	return out
}

// chime is two soft sines, a short confirmation tone.
func chime(weight, variant int) []float64 {
	n := sampleRate * (16 + 2*weight) / 100
	out := make([]float64, n)
	base := 520 + 35*float64(variant%9)
	for i := range out {
		t := float64(i) / sampleRate
		out[i] = 0.3 * (math.Sin(2*math.Pi*base*t) + 0.6*math.Sin(2*math.Pi*base*1.5*t)) * envelope(i, n, 5)
	}
	return out
}

// click is a very short tick.
func click(weight, variant int) []float64 {
	n := sampleRate * (4 + weight) / 100
	rand := noise(uint32(variant + 31))
	out := make([]float64, n)
	for i := range out {
		out[i] = 0.5 * rand() * envelope(i, n, 9)
	}
	return out
}

// wav encodes samples as a 16-bit mono PCM RIFF/WAVE blob.
func wav(samples []float64) []byte {
	dataSize := uint32(len(samples) * 2)
	b := []byte("RIFF")
	b = binary.LittleEndian.AppendUint32(b, 36+dataSize)
	b = append(b, "WAVEfmt "...)
	b = binary.LittleEndian.AppendUint32(b, 16)
	b = binary.LittleEndian.AppendUint16(b, 1) // PCM
	b = binary.LittleEndian.AppendUint16(b, 1) // mono
	b = binary.LittleEndian.AppendUint32(b, sampleRate)
	b = binary.LittleEndian.AppendUint32(b, sampleRate*2)
	b = binary.LittleEndian.AppendUint16(b, 2)
	b = binary.LittleEndian.AppendUint16(b, 16)
	b = append(b, "data"...)
	b = binary.LittleEndian.AppendUint32(b, dataSize)
	for _, s := range samples {
		v := int16(math.Round(math.Max(-1, math.Min(1, s)) * 32000))
		b = binary.LittleEndian.AppendUint16(b, uint16(v))
	}
	return b
}
