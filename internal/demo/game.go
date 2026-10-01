// Package demo composes native artwork with shared DCK scrolling, sprites,
// repeating backgrounds, palette conversion and automatically decoded music.
package demo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"io"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/olivierh59500/democonstructionkit/composite"
	"github.com/olivierh59500/democonstructionkit/scrolling"
	"github.com/olivierh59500/democonstructionkit/sound"
	playback "github.com/olivierh59500/democonstructionkit/sound/ebiten"
	"github.com/olivierh59500/democonstructionkit/sprites"
	"github.com/olivierh59500/go-dom-barbarian2/assets"
	"github.com/olivierh59500/go-dom-barbarian2/internal/source"
)

const Width, Height, FPS = 320, 200, 50

type Game struct {
	clock        *source.Clock
	images       []*ebiten.Image
	stage        *ebiten.Image
	font         [60][8]*ebiten.Image
	balls        [10]image.Rectangle
	stream       *scrolling.SliceStream
	scroll       *scrolling.Scrolling
	background   *composite.Background
	slots        *sprites.ImageSlots
	palette      *composite.IndexedPalette
	colors       [16]color.NRGBA
	baseColors   [16]color.NRGBA
	visual       *sound.Stream
	player       *playback.Player
	pcm          [960 * 8]byte
	closed       bool
	destination  *ebiten.Image
	upper, lower [4]*ebiten.Image
}

func resource(name string) ([]byte, error) { return assets.Files.ReadFile("original/" + name) }
func NewGame(mute bool) (_ *Game, err error) {
	g := &Game{}
	defer func() {
		if err != nil {
			g.Close()
		}
	}()
	load := func(name string) (*ebiten.Image, error) {
		b, e := resource(name)
		if e != nil {
			return nil, e
		}
		im, _, e := image.Decode(bytes.NewReader(b))
		if e != nil {
			return nil, e
		}
		texture := ebiten.NewImageFromImage(im)
		g.images = append(g.images, texture)
		return texture, nil
	}
	atlas, err := load("font.png")
	if err != nil {
		return nil, err
	}
	if _, err = load("background.png"); err != nil {
		return nil, err
	}
	if _, err = load("balls.png"); err != nil {
		return nil, err
	}
	for glyph := 0; glyph < 60; glyph++ {
		for slice := 0; slice < 8; slice++ {
			x, y := glyph%10*32+slice*4, glyph/10*20
			g.font[glyph][slice] = atlas.SubImage(image.Rect(x, y, x+4, y+20)).(*ebiten.Image)
		}
	}
	for i := range g.balls {
		g.balls[i] = image.Rect(i*32, 0, i*32+32, 32)
	}
	names := []string{"entrance-y.bin", "entrance-frames.bin", "scroll-y.bin", "profiles.bin"}
	var data [4][]byte
	for i, name := range names {
		data[i], err = resource(name)
		if err != nil {
			return nil, err
		}
	}
	g.clock, err = source.NewClock(data[0], data[1], data[2], data[3])
	if err != nil {
		return nil, err
	}
	message, err := resource("message.bin")
	if err != nil {
		return nil, err
	}
	var tokens []scrolling.SliceToken
	for _, glyph := range source.Words(message) {
		if glyph < 0 {
			break
		}
		if glyph >= 60 {
			return nil, fmt.Errorf("invalid native glyph %d", glyph)
		}
		tokens = append(tokens, scrolling.SliceToken{Glyph: glyph, Width: 32})
	}
	g.stream, err = scrolling.NewSliceStream(scrolling.SliceStreamConfig{Tokens: tokens, Capacity: 80, SliceWidth: 4, Repeat: true, Initial: scrolling.DNASlice{Glyph: -1}})
	if err != nil {
		return nil, err
	}
	g.scroll, err = scrolling.New(scrolling.Config{GlyphWindow: &scrolling.GlyphWindowConfig{Count: 80, Advance: 4, Glyph: func(slot int) scrolling.Glyph {
		entry := g.stream.Slices()[(g.stream.Head()+slot)%80]
		glyph := scrolling.Glyph{Advance: 4, Y: float64(g.clock.ColumnY(slot / 4))}
		if entry.Glyph >= 0 {
			glyph.Image = g.font[entry.Glyph][entry.Slice]
		}
		return glyph
	}}})
	if err != nil {
		return nil, err
	}
	g.background, err = composite.NewBackground(composite.BackgroundConfig{PeriodY: 50, Filter: ebiten.FilterNearest})
	if err != nil {
		return nil, err
	}
	g.slots, err = sprites.NewImageSlots(sprites.ImageSlotsConfig{Images: g.images, MaxSlots: 8, Filter: ebiten.FilterNearest})
	if err != nil {
		return nil, err
	}
	g.stage = ebiten.NewImage(Width, Height)
	words, err := resource("palette.bin")
	if err != nil {
		return nil, err
	}
	for i := range g.colors {
		w := binary.BigEndian.Uint16(words[i*2:])
		g.colors[i] = color.NRGBA{R: byte(w>>8&7) * 34, G: byte(w>>4&7) * 34, B: byte(w&7) * 34, A: 255}
	}
	g.baseColors = g.colors
	g.palette, err = composite.NewIndexedPalette(composite.IndexedPaletteConfig{Palette: g.colors[:], Channel: composite.BitplaneRed, Scale: 15, Blend: ebiten.BlendCopy})
	if err != nil {
		return nil, err
	}
	music, err := resource("music.ym")
	if err != nil {
		return nil, err
	}
	g.visual, err = sound.Open("music.ym", music, sound.Options{SampleRate: 48000, BlockFrames: 960, Loop: true})
	if err != nil {
		return nil, err
	}
	if !mute {
		g.player, err = playback.Open(nil, "music.ym", music, sound.Options{SampleRate: 48000, BlockFrames: 960, Loop: true})
		if err != nil {
			return nil, err
		}
	}
	if err = g.render(); err != nil {
		return nil, err
	}
	return g, nil
}
func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return ebiten.Termination
	}
	var volumes [3]int
	if g.clock.Stage == 2 {
		if g.player != nil && !g.player.IsPlaying() {
			g.player.Play()
		}
		if _, err := io.ReadFull(g.visual, g.pcm[:]); err != nil {
			return err
		}
		if registers, ok := g.visual.YMRegisters(); ok {
			for i := range volumes {
				volumes[i] = int(registers[8+i] & 15)
				if registers[8+i]&16 != 0 {
					volumes[i] = 15
				}
			}
		}
	}
	g.clock.Step(volumes)
	if g.clock.Stage == 2 {
		g.stream.Step(1, nil)
	}
	return g.render()
}
func (g *Game) render() error {
	g.stage.Fill(color.RGBA{A: 255})
	var poses [8]sprites.ImageSlot
	count := 0
	if g.clock.Stage == 0 {
		if g.clock.Tick > 0 {
			for i := 0; i < 3; i++ {
				poses[count] = sprites.ImageSlot{Image: 2, Source: g.balls[g.clock.EntryFrame], X: float64(64 + i*80), Y: float64(g.clock.EntryY)}
				count++
			}
		}
	} else if g.clock.Stage == 1 {
		for i := 0; i < 3; i++ {
			poses[count] = sprites.ImageSlot{Image: 2, Source: g.balls[g.clock.EntryFrame], X: float64(64 + i*80), Y: float64(g.clock.EntryY)}
			count++
		}
		for panel := 0; panel <= g.clock.Panel; panel++ {
			x := 0
			if panel == g.clock.Panel {
				x = g.clock.CurtainX()
			}
			poses[count] = sprites.ImageSlot{Image: 1, X: float64(x), Y: float64(panel * 50)}
			count++
		}
	} else {
		g.background.Draw(g.stage, g.images[1], composite.BackgroundPose{Y: -float64(g.clock.BackgroundY)})
		// Preserve the native top scanline at the hardware-scroll wrap boundary.
		if g.clock.Moving && g.clock.BackgroundY == 4 {
			poses[0] = sprites.ImageSlot{Image: 1, Source: image.Rect(0, 8, 320, 9)}
			if err := g.slots.SetSlots(poses[:1]); err != nil {
				return err
			}
			g.slots.Draw(g.stage)
		}
		g.scroll.Draw(g.stage)
		if err := g.scroll.Err(); err != nil {
			return err
		}
		for i := 0; i < 3; i++ {
			poses[count] = sprites.ImageSlot{Image: 2, Source: g.balls[g.clock.BallFrames[i]], X: float64(64 + i*80), Y: float64(g.clock.EntryY)}
			count++
		}
	}
	if err := g.slots.SetSlots(poses[:count]); err != nil {
		return err
	}
	g.slots.Draw(g.stage)
	g.colors = g.baseColors
	if g.clock.Stage == 2 {
		g.colors[6], g.colors[7], g.colors[8] = color.NRGBA{R: 238, A: 255}, color.NRGBA{R: 204, A: 255}, color.NRGBA{R: 170, A: 255}
	}
	if g.clock.Stage == 1 {
		pauseStart := 41
		if g.clock.Panel == 3 {
			pauseStart = 40
		}
		rotation := ((g.clock.Panel*22 + max(0, g.clock.PanelTick-pauseStart)) / 3) % 7
		for i := 9; i < 16; i++ {
			g.colors[i] = g.baseColors[9+(i-9+rotation)%7]
		}
	}
	return g.palette.SetPalette(g.colors[:])
}
func (g *Game) Draw(dst *ebiten.Image) {
	if g.clock.Stage == 1 {
		if g.destination != dst {
			g.destination = dst
			for panel := 1; panel < 4; panel++ {
				g.upper[panel] = dst.SubImage(image.Rect(0, 0, Width, panel*50)).(*ebiten.Image)
				g.lower[panel] = dst.SubImage(image.Rect(0, panel*50, Width, Height)).(*ebiten.Image)
			}
		}
		panel := g.clock.Panel
		if panel > 0 {
			if err := g.palette.SetPalette(g.baseColors[:]); err != nil {
				panic(err)
			}
			if err := g.palette.Draw(g.upper[panel], g.stage); err != nil {
				panic(err)
			}
			if err := g.palette.SetPalette(g.colors[:]); err != nil {
				panic(err)
			}
			if err := g.palette.Draw(g.lower[panel], g.stage); err != nil {
				panic(err)
			}
			return
		}
	}
	if err := g.palette.Draw(dst, g.stage); err != nil {
		panic(err)
	}
}
func (*Game) Layout(int, int) (int, int) { return Width, Height }
func (g *Game) Tick() int                { return g.clock.Tick }
func (g *Game) Close() {
	if g == nil || g.closed {
		return
	}
	g.closed = true
	if g.player != nil {
		g.player.Close()
	}
	if g.visual != nil {
		g.visual.Close()
	}
	if g.scroll != nil {
		g.scroll.Close()
	}
	if g.slots != nil {
		g.slots.Close()
	}
	if g.palette != nil {
		g.palette.Close()
	}
	for _, im := range g.images {
		im.Deallocate()
	}
	if g.stage != nil {
		g.stage.Deallocate()
	}
}
