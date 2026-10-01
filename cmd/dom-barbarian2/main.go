// Command dom-barbarian2 runs the native Go/Ebitengine conversion.
package main

import (
	"flag"
	"github.com/hajimehoshi/ebiten/v2"
	capture "github.com/olivierh59500/democonstructionkit/fidelity/ebiten"
	"github.com/olivierh59500/go-dom-barbarian2/internal/demo"
	"log"
)

func main() {
	directory := flag.String("capture", "", "write native PNG frames")
	first := flag.Int("frame", 300, "first capture tick")
	count := flag.Int("frames", 1, "consecutive capture frames (1..1500)")
	mute := flag.Bool("mute", false, "disable device music")
	flag.Parse()
	if *directory != "" {
		if *first < 0 || *count < 1 || *count > 1500 || *first > int(^uint(0)>>1)-*count {
			log.Fatal("invalid capture range")
		}
		frames := make([]int, *count)
		for i := range frames {
			frames[i] = *first + i
		}
		var g *demo.Game
		err := capture.Run(capture.Config{Directory: *directory, Frames: frames, Width: demo.Width, Height: demo.Height}, func() (ebiten.Game, error) { var e error; g, e = demo.NewGame(true); return g, e })
		if g != nil {
			g.Close()
		}
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	g, err := demo.NewGame(*mute)
	if err != nil {
		log.Fatal(err)
	}
	defer g.Close()
	ebiten.SetTPS(demo.FPS)
	ebiten.SetWindowSize(demo.Width*3, demo.Height*3)
	ebiten.SetWindowTitle("Dom Barbarian II Go / The Replicants / 1989")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
