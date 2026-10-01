package source

import (
	"encoding/binary"
	"fmt"
)

const FPS = 50
const EntranceTicks = 54
const CurtainTicks = 251

// Clock preserves the native entrance, four-phase scroll clock and authored
// profile changes. Drawing stays in DCK; this controller only supplies poses.
type Clock struct {
	Tick, MainTick, Stage                     int
	Panel, PanelTick                          int
	EntryY, EntryFrame                        int
	ScrollY, BackgroundY, Profile             int
	ProfileTick, ProfileLimit, WaveCursor     int
	StartTick, WaitTick                       int
	Moving, Started                           bool
	Levels                                    [3]int
	BallFrames                                [3]int
	EntranceY, EntranceFrames, Wave, Profiles []int
}

func Words(b []byte) []int {
	out := make([]int, len(b)/2)
	for i := range out {
		out[i] = int(int16(binary.BigEndian.Uint16(b[i*2:])))
	}
	return out
}
func NewClock(entranceY, entranceFrames, wave, profiles []byte) (*Clock, error) {
	c := &Clock{Stage: 1, EntranceY: Words(entranceY), EntranceFrames: Words(entranceFrames), Wave: Words(wave), Profiles: Words(profiles), ProfileLimit: 150, Levels: [3]int{10, 10, 10}, EntryY: -32}
	if len(c.EntranceY) != 54 || len(c.EntranceFrames) != 56 || len(c.Wave) != 189 || len(c.Profiles) != 560 {
		return nil, fmt.Errorf("source: incomplete native motion tables")
	}
	return c, nil
}
func (c *Clock) Step(volumes [3]int) {
	c.Tick++
	if c.Tick <= CurtainTicks {
		c.Stage = 1
		c.Panel = min(3, (c.Tick-1)/63)
		c.PanelTick = c.Tick - c.Panel*63
		return
	}
	if c.Tick <= CurtainTicks+EntranceTicks {
		c.Stage = 0
		index := c.Tick - CurtainTicks - 1
		if c.EntranceY[index] >= 0 {
			c.EntryY, c.EntryFrame = c.EntranceY[index], c.EntranceFrames[index]
		}
		return
	}
	if c.Stage != 2 {
		c.Stage = 2
		c.MainTick = 0
	}
	c.MainTick++
	frame := c.MainTick - 1
	profileStep := frame < 8 && frame%4 == 0 || frame >= 8 && frame%8 == 4
	waitStep := frame < 8 && frame%4 == 2 || frame >= 8 && frame%8 == 6
	if profileStep {
		if !c.Started {
			c.StartTick++
			if c.StartTick == 100 {
				c.Wave[1] = 2
				c.Started = true
				c.StartTick = 0
			}
		}
		c.ProfileTick++
		if c.ProfileTick > c.ProfileLimit {
			c.ProfileTick = 0
			c.Profile = (c.Profile + 1) % 28
			c.ProfileLimit = 40
		}
	}
	if c.Wave[c.WaveCursor] < 0 {
		c.WaveCursor = 0
	}
	c.ScrollY = c.Wave[c.WaveCursor]
	c.WaveCursor++
	if waitStep && !c.Moving {
		c.WaitTick++
		if c.WaitTick > 300 {
			c.Moving = true
		}
	}
	if c.Moving && c.MainTick > 2399 {
		c.BackgroundY = (c.BackgroundY + 4) % 200
	}
	for i, v := range volumes {
		v = max(0, min(15, v))
		if v-c.Levels[i] > 1 {
			v = c.Levels[i] + 1
		}
		c.Levels[i] = max(6, v)
		c.BallFrames[i] = 15 - c.Levels[i]
	}
}
func (c *Clock) ColumnY(column int) int { return c.ScrollY + c.Profiles[c.Profile*20+column]/160 }
func (c *Clock) CurtainX() int          { return max(0, 320-c.PanelTick*8) }
