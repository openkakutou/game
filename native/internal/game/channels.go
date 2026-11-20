package game

import "github.com/openkakutou/engine/v2/match"

// WavChannels is Ikemen GO's default sound channel count
// (src/resources/defaultConfig.ini WavChannels = 32): the size of the
// effect pool, and the bound of a PlaySnd channel number.
const WavChannels = 32

// SoundSlot is one mixer channel of the fighters' effect pool.
type SoundSlot struct {
	Side match.Side
	// ChannelNo is the PlaySnd channel the slot plays, -1 for none.
	ChannelNo int
	Playing   bool
	// Started orders the sounds by start, oldest lowest.
	Started int64
}

// RequestSlot returns the slot a fighter's sound plays on, -1 when it is
// dropped, after Ikemen GO's SoundChannels.Request (src/sound.go:1234-1330,
// commit 54d4148a, without per-sound priority, which the engine does not
// report): the fighter's own slot on that channel is reused, cutting its
// sound unless lowPriority; else a free slot; else, unless lowPriority, the
// oldest unnumbered sound is cut, then the oldest numbered one.
func RequestSlot(slots []SoundSlot, side match.Side, channelNo int, lowPriority bool) int {
	if channelNo < 0 || channelNo >= WavChannels {
		channelNo = -1
	}
	if channelNo >= 0 {
		for i, s := range slots {
			if s.Side != side || s.ChannelNo != channelNo {
				continue
			}
			if s.Playing && lowPriority {
				return -1
			}
			return i
		}
	}
	for i, s := range slots {
		if !s.Playing {
			return i
		}
	}
	if lowPriority {
		return -1
	}
	oldest, oldestNumbered := -1, -1
	for i, s := range slots {
		if s.ChannelNo < 0 {
			if oldest < 0 || s.Started < slots[oldest].Started {
				oldest = i
			}
			continue
		}
		if oldestNumbered < 0 || s.Started < slots[oldestNumbered].Started {
			oldestNumbered = i
		}
	}
	if oldest >= 0 {
		return oldest
	}
	return oldestNumbered
}
