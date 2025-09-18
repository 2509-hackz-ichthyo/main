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

	// オンラインモードでゲーム中でない場合は、状態画面を表示
	if g.IsOnline && g.State != GameStateInGame {
		g.drawStatusScreen(screen)
		return
	}

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
		// 現在の手番を表示
		g.drawCurrentTurnInfo(screen)

		// 次のコマのプレビューを描画（右側に移動）
		g.drawNextPiecePreview(screen)
	}
}

// drawStatusScreen は接続状態に応じた画面を描画する
func (g *Game) drawStatusScreen(screen *ebiten.Image) {
	if g.FontFace == nil {
		return
	}

	var message string
	var bgColor color.RGBA

	switch g.State {
	case GameStateDisconnected:
		message = "サーバーとの接続が切断されました"
		bgColor = color.RGBA{220, 220, 220, 255}
	case GameStateConnecting:
		message = "サーバーに接続中..."
		bgColor = color.RGBA{200, 230, 255, 255}
	case GameStateWaiting:
		message = "新しいプレイヤーの到着を待っています..."
		bgColor = color.RGBA{255, 248, 200, 255}
	case GameStateError:
		if g.ErrorMessage != "" {
			message = g.ErrorMessage
		} else {
			message = "エラーが発生しました"
		}
		bgColor = color.RGBA{255, 200, 200, 255}
	default:
		message = "接続中..."
		bgColor = color.RGBA{240, 240, 240, 255}
	}

	// 背景色を設定
	screen.Fill(bgColor)

	// プレイヤー情報を表示
	if g.PlayerID != "" {
		playerInfo := fmt.Sprintf("プレイヤーID: %s", g.PlayerID)
		g.drawCenteredText(screen, playerInfo, 0, -50, color.RGBA{100, 100, 100, 255})
	}

	// メインメッセージを表示
	g.drawCenteredText(screen, message, 0, 0, color.Black)

	// 状態に応じた追加情報
	if g.State == GameStateWaiting {
		subMessage := "しばらくお待ちください"
		g.drawCenteredText(screen, subMessage, 0, 50, color.RGBA{120, 120, 120, 255})
	}
}

// drawCenteredText は画面中央にテキストを描画する（オフセット付き）
func (g *Game) drawCenteredText(screen *ebiten.Image, message string, offsetX, offsetY int, textColor color.Color) {
	if g.FontFace == nil {
		return
	}

	// テキストの大きさを測定
	textWidth, textHeight := text.Measure(message, g.FontFace, 0)

	// 画面中央に配置（オフセット付き）
	screenWidth, screenHeight := 800, 600
	messageX := float64(screenWidth/2) - textWidth/2 + float64(offsetX)
	messageY := float64(screenHeight/2) - textHeight/2 + float64(offsetY)

	// テキストを描画
	textOptions := &text.DrawOptions{}
	textOptions.GeoM.Translate(messageX, messageY)
	textOptions.ColorScale.ScaleWithColor(textColor)
	text.Draw(screen, message, g.FontFace, textOptions)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 800, 600
}

// drawCurrentTurnInfo は現在の手番情報を表示する
func (g *Game) drawCurrentTurnInfo(screen *ebiten.Image) {
	if g.FontFace == nil {
		return
	}

	var turnMessage string
	var messageColor color.Color

	if g.IsOnline {
		// オンラインモードの場合
		if g.CurrentTurn {
			turnMessage = "あなたの手番です"
			messageColor = color.RGBA{0, 120, 0, 255} // 緑色
		} else {
			turnMessage = "相手の手番です"
			messageColor = color.RGBA{120, 120, 120, 255} // グレー色
		}
	} else {
		// オフラインモードの場合
		currentPlayer := g.getCurrentPlayer()
		turnMessage = fmt.Sprintf("%s の手番", currentPlayer)
		messageColor = color.Black
	}

	// テキストを描画
	textOptions := &text.DrawOptions{}
	textOptions.GeoM.Translate(10, 0)
	textOptions.ColorScale.ScaleWithColor(messageColor)
	text.Draw(screen, turnMessage, g.FontFace, textOptions)
}

// drawNextPiecePreview は次のコマのプレビューを枠付きで表示する
func (g *Game) drawNextPiecePreview(screen *ebiten.Image) {
	if g.FontFace == nil {
		return
	}

	// 次のコマの表示位置（右側に移動）
	previewX := float32(BoardOffset + BoardSize*CellSize + 80)
	previewY := float32(BoardOffset + 80)

	// 枠のサイズと位置
	boxWidth := float32(100)
	boxHeight := float32(80)
	boxX := previewX - boxWidth/2
	boxY := previewY - 30

	// 枠を描画（背景色付き）
	vector.DrawFilledRect(screen, boxX, boxY, boxWidth, boxHeight, color.RGBA{250, 250, 250, 255}, false)
	vector.StrokeRect(screen, boxX, boxY, boxWidth, boxHeight, 2, color.Black, false)

	// ラベルテキスト「次のコマ」を描画
	labelText := "次のコマ"
	labelWidth, _ := text.Measure(labelText, g.FontFace, 0)
	labelX := float64(previewX) - labelWidth/2
	labelY := float64(boxY+20) - 80

	textOptions := &text.DrawOptions{}
	textOptions.GeoM.Translate(labelX, labelY)
	textOptions.ColorScale.ScaleWithColor(color.Black)
	text.Draw(screen, labelText, g.FontFace, textOptions)

	// 次のコマの色を描画
	nextColorRGB := colorToRGB(g.NextColor)
	vector.DrawFilledCircle(screen, previewX, previewY+10, 20, nextColorRGB, false)
	vector.StrokeCircle(screen, previewX, previewY+10, 20, 2, color.Black, false)
}
