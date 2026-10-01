// Package source preserves the intro's native pixel layout and animation order.
package source

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

func ExportAssets(memory []byte, directory string) error {
	if len(memory) < 0x48540 || binary.BigEndian.Uint32(memory[0x1d31a:]) != 0x1d350 {
		return fmt.Errorf("source: incomplete unpacked intro memory")
	}
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}
	for _, part := range []struct {
		name       string
		start, end int
	}{
		{"font.bin", 0x1d350, 0x1d350 + 19200},
		{"background.bin", 0x27420, 0x27420 + 8000},
		{"balls.bin", 0x29384, 0x29384 + 5120},
		{"palette.bin", 0x27400, 0x27420},
		{"message.bin", 0x25ac8, 0x26f98},
		{"profiles.bin", 0x26f9e, 0x273fe},
		{"scroll-y.bin", 0x2594e, 0x25ac8},
		{"entrance-y.bin", 0x258e2, 0x2594e},
		{"entrance-frames.bin", 0x25872, 0x258e2},
	} {
		if err := os.WriteFile(filepath.Join(directory, part.name), memory[part.start:part.end], 0644); err != nil {
			return err
		}
	}
	for _, part := range []struct {
		name                 string
		start, width, height int
		transparent          bool
	}{
		{"font.png", 0x1d350, 320, 120, true}, {"background.png", 0x27420, 320, 50, false}, {"balls.png", 0x29384, 320, 32, true},
	} {
		img := image.NewNRGBA(image.Rect(0, 0, part.width, part.height))
		for y := 0; y < part.height; y++ {
			for x := 0; x < part.width; x++ {
				at := part.start + y*(part.width/2) + (x/16)*8
				var index byte
				for plane := 0; plane < 4; plane++ {
					index |= byte(binary.BigEndian.Uint16(memory[at+plane*2:])>>uint(15-x%16)&1) << plane
				}
				alpha := byte(255)
				if part.transparent && index == 0 {
					alpha = 0
				}
				img.SetNRGBA(x, y, color.NRGBA{R: index * 17, A: alpha})
			}
		}
		file, err := os.Create(filepath.Join(directory, part.name))
		if err != nil {
			return err
		}
		err = png.Encode(file, img)
		closeErr := file.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
