package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Game struct{}


const (
	BoardSize   = 8
	CellSize    = 60
	BoardOffset = 50
)

// colorToRGB は0-255の色値をRGBに変換する
func colorToRGB(c uint8) color.RGBA {
	return color.RGBA{R: c, G: c, B: c, A: 255}
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// 画面を薄いグレーでクリア
	screen.Fill(color.RGBA{240, 240, 240, 255})

	// ボードのグリッドを描画
	for i := 0; i <= BoardSize; i++ {
		x := float32(BoardOffset + i*CellSize)
		y1 := float32(BoardOffset)
		y2 := float32(BoardOffset + BoardSize*CellSize)

		// 縦線
		vector.StrokeLine(screen, x, y1, x, y2, 2, color.Black, false)

		// 横線
		y := float32(BoardOffset + i*CellSize)
		x1 := float32(BoardOffset)
		x2 := float32(BoardOffset + BoardSize*CellSize)
		vector.StrokeLine(screen, x1, y, x2, y, 2, color.Black, false)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 800, 600
}

func main() {
	// パーサーのテスト
	gameData, err := parseGameData(mockPlayData)
	if err != nil {
		log.Printf("パーサーエラー: %v", err)
	} else {
		log.Printf("対局データを正常に解析しました。手数: %d", gameData.GetMoveCount())
		
		// 最初の5手を表示
		for i := 0; i < 5 && i < gameData.GetMoveCount(); i++ {
			move, _ := gameData.GetMove(i)
			log.Printf("手%d: row=%d, col=%d, color=%d", i+1, move.Row, move.Col, move.Color)
		}
		
		// 色の範囲を表示
		minColor, maxColor := gameData.GetColorRange()
		log.Printf("色の範囲: %d - %d", minColor, maxColor)
	}

	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Hello, World!")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
