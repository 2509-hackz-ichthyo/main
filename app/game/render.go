package main

import (
	"bytes"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// colorToRGB は0-255の色値をRGBに変換する
func colorToRGB(c uint8) color.RGBA {
	return color.RGBA{R: c, G: c, B: c, A: 255}
}

// initializeFont はフォントを初期化する
func (g *Game) initializeFont() error {
	fontSource, err := text.NewGoTextFaceSource(bytes.NewReader(fontData))
	if err != nil {
		return fmt.Errorf("failed to create font source: %v", err)
	}

	g.FontFace = &text.GoTextFace{
		Source: fontSource,
		Size:   32,
	}

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

	// コマを描画
	for x := 0; x < BoardSize; x++ {
		for y := 0; y < BoardSize; y++ {
			if !g.Board.Squares[x][y].IsEmpty() {
				piece := g.Board.Squares[x][y].Piece
				pieceColor := colorToRGB(piece.Color)

				centerX := float32(BoardOffset + x*CellSize + CellSize/2)
				centerY := float32(BoardOffset + y*CellSize + CellSize/2)
				radius := float32(CellSize/2 - 4)

				vector.DrawFilledCircle(screen, centerX, centerY, radius, pieceColor, false)
				vector.StrokeCircle(screen, centerX, centerY, radius, 2, color.Black, false)
			}
		}
	}

	// ゲーム終了時の勝利メッセージ表示
	if g.GameOver && g.FontFace != nil {
		// 画面中央に勝利メッセージを表示
		winMessage := fmt.Sprintf("どちらかというと %s の勝利！", g.Winner)

		// テキストの大きさを測定
		textWidth, textHeight := text.Measure(winMessage, g.FontFace, 0)

		// メッセージを画面中央に配置
		screenWidth, screenHeight := 800, 600
		messageX := float64(screenWidth/2) - textWidth/2
		messageY := float64(screenHeight/2) - textHeight/2

		// 背景の四角形を描画（見やすくするため）
		bgPadding := float32(20)
		bgX := float32(messageX) - bgPadding
		bgY := float32(messageY) - bgPadding
		bgWidth := float32(textWidth) + bgPadding*2
		bgHeight := float32(textHeight) + bgPadding*2

		vector.DrawFilledRect(screen, bgX, bgY, bgWidth, bgHeight, color.RGBA{255, 255, 255, 240}, false)
		vector.StrokeRect(screen, bgX, bgY, bgWidth, bgHeight, 3, color.Black, false)

		// テキストを描画
		textOptions := &text.DrawOptions{}
		textOptions.GeoM.Translate(messageX, messageY)
		textOptions.ColorScale.ScaleWithColor(color.Black)
		text.Draw(screen, winMessage, g.FontFace, textOptions)
	} else {
		// 次の色のプレビューを描画
		nextColorRGB := colorToRGB(g.NextColor)
		previewX := float32(BoardOffset + BoardSize*CellSize + 40)
		previewY := float32(BoardOffset + 60)
		vector.DrawFilledCircle(screen, previewX, previewY, 20, nextColorRGB, false)
		vector.StrokeCircle(screen, previewX, previewY, 20, 2, color.Black, false)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 800, 600
}
