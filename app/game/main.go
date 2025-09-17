package main

import (
	_ "embed"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed DotGothic16-Regular.ttf
var fontData []byte

func main() {
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("アンミカリバーシ")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
