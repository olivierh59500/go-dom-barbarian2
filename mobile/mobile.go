// Package mobile hosts the same 50 Hz intro as the desktop command.
package mobile

import (
	"log"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	enginemobile "github.com/hajimehoshi/ebiten/v2/mobile"
	"github.com/olivierh59500/go-dom-barbarian2/internal/demo"
)

type host struct {
	game         *demo.Game
	requests     chan int
	lastReport   time.Time
	verification bool
}

var gameHost = &host{requests: make(chan int, 1)}

func init() {
	ebiten.SetTPS(demo.FPS)
	enginemobile.SetGame(gameHost)
}

func (h *host) Update() error {
	if h.game == nil {
		tick := 0
		select {
		case tick = <-h.requests:
			h.verification = true
		default:
		}
		// Android's application context and graphics view are ready here.
		// Constructing the game in init would open audio before Seq.setContext.
		var err error
		h.game, err = demo.NewGame(h.verification && tick > 0)
		if err != nil {
			return err
		}

		for h.game.Tick() < tick {
			if err := h.game.Update(); err != nil {
				return err
			}
		}
	}
	if err := h.game.Update(); err != nil {
		return err
	}
	if h.verification && time.Since(h.lastReport) >= 5*time.Second {
		var memory runtime.MemStats
		runtime.ReadMemStats(&memory)
		log.Printf("dombarbarian2_verify tick=%d tps=%.1f fps=%.1f heap_mib=%.1f", h.game.Tick(), ebiten.ActualTPS(), ebiten.ActualFPS(), float64(memory.HeapAlloc)/(1<<20))
		h.lastReport = time.Now()
	}
	return nil
}

func (h *host) Draw(dst *ebiten.Image) {
	if h.game != nil {
		h.game.Draw(dst)
	}
}

func (*host) Layout(int, int) (int, int) { return demo.Width, demo.Height }

// ConfigureVerification queues an optional startup checkpoint. Positive ticks
// mute device audio; zero retains normal playback. Rendering still uses 50 Hz.
func ConfigureVerification(tick int) bool {
	if tick < 0 || tick > 12000 {
		return false
	}
	select {
	case gameHost.requests <- tick:
		return true
	default:
		return false
	}
}

// Dummy exports an entry point for the mobile binding generator.
func Dummy() {}
