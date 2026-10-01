package demo

import (
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	gpu "github.com/olivierh59500/democonstructionkit/fidelity/ebiten/testutil"
	"github.com/olivierh59500/go-dom-barbarian2/internal/source"
)

func TestDCKLayersMatchOriginalPlanarFrames(t *testing.T) {
	gpu.RequireGPU(t)
	game, err := NewGame(true)
	if err != nil {
		t.Fatal(err)
	}
	defer game.Close()
	game.clock.Stage = 2
	game.clock.EntryY = 84
	game.clock.Tick = source.EntranceTicks + source.CurtainTicks
	checkpoints := map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true, 6: true, 7: true, 8: true, 9: true, 10: true, 160: true, 256: true, 789: true, 1000: true, 1197: true, 1200: true, 1500: true, 2000: true, 2398: true, 2399: true, 2400: true, 2401: true, 2402: true, 2403: true, 2404: true, 2500: true, 2599: true, 2600: true, 3000: true}
	pixels := make([]byte, Width*Height*4)
	for tick := 1; tick <= 3000; tick++ {
		game.clock.Step([3]int{6, 9, 12})
		game.stream.Step(1, nil)
		if err := game.render(); err != nil {
			t.Fatal(err)
		}
		if !checkpoints[tick] {
			continue
		}
		file, err := os.Open(fmt.Sprintf("testdata/%06d.bin.gz", tick))
		if err != nil {
			t.Fatal(err)
		}
		z, err := gzip.NewReader(file)
		if err != nil {
			file.Close()
			t.Fatal(err)
		}
		native, err := io.ReadAll(z)
		z.Close()
		file.Close()
		if err != nil || len(native) != 32000 {
			t.Fatal("incomplete native frame", tick, err)
		}
		game.stage.ReadPixels(pixels)
		mismatch := 0
		first := ""
		for y := 0; y < Height; y++ {
			for x := 0; x < Width; x++ {
				at := y*160 + x/16*8
				var expected byte
				for p := 0; p < 4; p++ {
					expected |= byte(binary.BigEndian.Uint16(native[at+p*2:])>>uint(15-x%16)&1) << p
				}
				actual := pixels[(y*Width+x)*4] / 17
				if expected != actual {
					mismatch++
					if first == "" {
						first = fmt.Sprintf("(%d,%d): got %d, want %d", x, y, actual, expected)
					}
				}
			}
		}
		if mismatch > 0 {
			t.Errorf("native frame %d: %d pixel mismatches; first %s", tick, mismatch, first)
		}
	}
}

func TestEntrancePaletteKeepsCompletedPanelsUnchanged(t *testing.T) {
	gpu.RequireGPU(t)
	game, err := NewGame(true)
	if err != nil {
		t.Fatal(err)
	}
	defer game.Close()
	game.clock.Stage, game.clock.Panel, game.clock.PanelTick = 1, 1, 44
	game.clock.EntryY, game.clock.EntryFrame = 84, 8
	if err := game.render(); err != nil {
		t.Fatal(err)
	}
	output := ebiten.NewImage(Width, Height)
	defer output.Deallocate()
	game.Draw(output)
	indices := make([]byte, Width*Height*4)
	pixels := make([]byte, len(indices))
	game.stage.ReadPixels(indices)
	output.ReadPixels(pixels)
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			at := (y*Width + x) * 4
			index := int(indices[at] / 17)
			want := game.colors[index]
			if y < 50 {
				want = game.baseColors[index]
			}
			if pixels[at] != want.R || pixels[at+1] != want.G || pixels[at+2] != want.B || pixels[at+3] != want.A {
				t.Fatalf("palette split at (%d,%d): got %v, want %v", x, y, pixels[at:at+4], want)
			}
		}
	}
}

func TestBallEntranceStaysInFrontOfCompletedBackdrop(t *testing.T) {
	gpu.RequireGPU(t)
	game, err := NewGame(true)
	if err != nil {
		t.Fatal(err)
	}
	defer game.Close()
	for tick := 0; tick < source.CurtainTicks; tick++ {
		game.clock.Step([3]int{})
	}
	sheet := make([]byte, 320*32*4)
	game.images[2].ReadPixels(sheet)
	stage := make([]byte, Width*Height*4)
	for tick := 0; tick < 53; tick++ {
		game.clock.Step([3]int{})
		if err := game.render(); err != nil {
			t.Fatal(err)
		}
		game.stage.ReadPixels(stage)
		// Every opaque sprite pixel must cover the backdrop, through the full descent.
		for ball := 0; ball < 3; ball++ {
			for y := 0; y < 32; y++ {
				for x := 0; x < 32; x++ {
					sourceAt := (y*320 + game.clock.EntryFrame*32 + x) * 4
					if sheet[sourceAt+3] == 0 {
						continue
					}
					targetAt := ((game.clock.EntryY+y)*Width + 64 + ball*80 + x) * 4
					if stage[targetAt] != sheet[sourceAt] || stage[targetAt+3] != 255 {
						t.Fatalf("ball %d obscured at entrance tick %d pixel (%d,%d)", ball, tick, x, y)
					}
				}
			}
		}
		if stage[3] != 255 {
			t.Fatal("backdrop disappeared during ball entrance")
		}
	}
}
