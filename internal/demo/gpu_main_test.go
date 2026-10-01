//go:build dck_gpu_rendercheck

package demo

import (
	gpu "github.com/olivierh59500/democonstructionkit/fidelity/ebiten/testutil"
	"os"
	"testing"
)

func TestMain(m *testing.M) { os.Exit(gpu.RunGPU(m)) }
