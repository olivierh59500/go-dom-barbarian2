// Command extract exports artwork and tables from unpacked Atari memory.
package main

import (
	"flag"
	"github.com/olivierh59500/go-dom-barbarian2/internal/source"
	"log"
	"os"
)

func main() {
	input := flag.String("input", "", "unpacked native memory image")
	output := flag.String("assets", "assets/original", "asset directory")
	flag.Parse()
	b, err := os.ReadFile(*input)
	if err != nil {
		log.Fatal(err)
	}
	if err = source.ExportAssets(b, *output); err != nil {
		log.Fatal(err)
	}
}
