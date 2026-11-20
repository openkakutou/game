package game

import (
	"testing"

	"github.com/openkakutou/engine/v2/evaluator"
	"github.com/openkakutou/engine/v2/match"
)

// playing returns n slots, all busy with side 1's unnumbered sounds started
// at times 1..n.
func playing(n int) []SoundSlot {
	s := make([]SoundSlot, n)
	for i := range s {
		s[i] = SoundSlot{Side: match.SideP1, ChannelNo: -1, Playing: true, Started: int64(i + 1)}
	}
	return s
}

// freeSlots returns n silent slots, as the mixer starts.
func freeSlots(n int) []SoundSlot {
	s := make([]SoundSlot, n)
	for i := range s {
		s[i].ChannelNo = -1
	}
	return s
}

func TestRequestSlot_FreeSlotWhenNoChannelGiven(t *testing.T) {
	slots := freeSlots(4)
	slots[0] = SoundSlot{Side: match.SideP1, ChannelNo: -1, Playing: true}
	got := RequestSlot(slots, match.SideP1, -1, false)
	if got != 1 {
		t.Fatalf("slot = %d, want the first free slot 1", got)
	}
}

func TestRequestSlot_SameChannelOfSameFighterIsReplaced(t *testing.T) {
	slots := freeSlots(4)
	slots[2] = SoundSlot{Side: match.SideP1, ChannelNo: 0, Playing: true}
	if got := RequestSlot(slots, match.SideP1, 0, false); got != 2 {
		t.Fatalf("slot = %d, want 2: the fighter's voice on channel 0 is cut by the new one", got)
	}
}

func TestRequestSlot_SameChannelOfOtherFighterIsNotTouched(t *testing.T) {
	slots := freeSlots(4)
	slots[0] = SoundSlot{Side: match.SideP2, ChannelNo: 0, Playing: true}
	if got := RequestSlot(slots, match.SideP1, 0, false); got != 1 {
		t.Fatalf("slot = %d, want free slot 1: channels belong to one fighter", got)
	}
}

func TestRequestSlot_LowPriorityIsDroppedWhileItsChannelPlays(t *testing.T) {
	slots := freeSlots(4)
	slots[1] = SoundSlot{Side: match.SideP1, ChannelNo: 3, Playing: true}
	if got := RequestSlot(slots, match.SideP1, 3, true); got != -1 {
		t.Fatalf("slot = %d, want -1: lowpriority never cuts a busy channel", got)
	}
}

func TestRequestSlot_LowPriorityPlaysOnceItsChannelIsSilent(t *testing.T) {
	slots := freeSlots(4)
	slots[1] = SoundSlot{Side: match.SideP1, ChannelNo: 3}
	if got := RequestSlot(slots, match.SideP1, 3, true); got != 1 {
		t.Fatalf("slot = %d, want 1", got)
	}
}

func TestRequestSlot_ChannelOutOfRangeActsAsNoChannel(t *testing.T) {
	slots := playing(3)
	slots = append(slots, freeSlots(1)...)
	slots[0].ChannelNo = WavChannels
	if got := RequestSlot(slots, match.SideP1, WavChannels, false); got != 3 {
		t.Fatalf("slot = %d, want free slot 3: channel %d is out of range, like -1", got, WavChannels)
	}
}

func TestRequestSlot_AllBusyEvictsOldestUnnumberedFirst(t *testing.T) {
	slots := playing(3)
	slots[0].ChannelNo = 5 // oldest, but numbered
	if got := RequestSlot(slots, match.SideP2, -1, false); got != 1 {
		t.Fatalf("slot = %d, want 1: the oldest unnumbered sound goes first", got)
	}
}

func TestRequestSlot_AllBusyAndNumberedEvictsOldest(t *testing.T) {
	slots := playing(3)
	for i := range slots {
		slots[i].ChannelNo = i + 4
	}
	slots[2].Started = 0
	if got := RequestSlot(slots, match.SideP2, -1, false); got != 2 {
		t.Fatalf("slot = %d, want the oldest slot 2", got)
	}
}

func TestRequestSlot_AllBusyLowPriorityIsDropped(t *testing.T) {
	if got := RequestSlot(playing(3), match.SideP2, -1, true); got != -1 {
		t.Fatalf("slot = %d, want -1", got)
	}
}

func TestRequestSlot_NoSlotsIsDropped(t *testing.T) {
	if got := RequestSlot(nil, match.SideP1, -1, false); got != -1 {
		t.Fatalf("slot = %d, want -1", got)
	}
}

func TestQueueCues_CarriesChannelAndLowPriority(t *testing.T) {
	m := &Match{}
	m.queueCues([2][]evaluator.SoundEvent{{{Group: 1, Sample: 0, Channel: 2, LowPriority: true}}, nil})
	c := m.DrainCues(nil)
	if len(c) != 1 || c[0].Channel != 2 || !c[0].LowPriority {
		t.Fatalf("cues = %+v, want channel 2 lowpriority", c)
	}
}
