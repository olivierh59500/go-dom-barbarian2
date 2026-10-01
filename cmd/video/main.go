// Command video exports the demo canvas and its own soundtrack with DCK.
package main

import (
	"flag"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/democonstructionkit/video"
	"github.com/olivierh59500/go-dom-barbarian2/internal/demo"
	"log"
	"time"
)

func main() {
	config := video.Config{Output: "recordings/dom-barbarian2.mp4", Title: "Dom Barbarian II Go", Width: demo.Width * 2, Height: demo.Height * 2, FPS: demo.FPS, TPS: demo.FPS, SampleRate: 48000, Duration: 3 * time.Minute, PosterAt: 20 * time.Second}
	config.Flags(flag.CommandLine)
	flag.Parse()
	if config.Duration <= 0 {
		log.Fatal("a looping intro requires a positive recording duration")
	}
	if err := video.Run(config, func() (ebiten.Game, error) { return demo.NewGame(false) }); err != nil {
		log.Fatal(err)
	}
}
