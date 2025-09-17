package main

import (
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Game struct {
	gameData    *GameData // パースした対局データ
	board       *Board    // 現在のボード状態
	currentMove int       // 現在の手数
	timer       float64   // 経過時間（秒）
	interval    float64   // コマ配置間隔（3秒）
	isPlaying   bool      // 再生中フラグ
}

const (
	BoardSize   = 8
	CellSize    = 60
	BoardOffset = 50
)

// colorToRGB は0-255の色値をRGBに変換する
func colorToRGB(c uint8) color.RGBA {
	return color.RGBA{R: c, G: c, B: c, A: 255}
}

// NewGame は新しいゲームインスタンスを作成する
func NewGame() *Game {
	// mockPlayDataをパース
	gameData, err := parseGameData(mockPlayData)
	if err != nil {
		fmt.Printf("ゲーム初期化エラー: %v\n", err)
		return nil
	}

	// 空のボードを初期化
	board := &Board{}

	game := &Game{
		gameData:    gameData,
		board:       board,
		currentMove: -1,   // まだ何も置いていない状態
		timer:       0,    // タイマー初期化
		interval:    1.0,  // 1秒間隔
		isPlaying:   true, // 自動再生開始
	}

	fmt.Printf("ゲーム初期化完了。総手数: %d\n", gameData.GetMoveCount())
	return game
}

func (g *Game) Update() error {
	if !g.isPlaying {
		return nil
	}

	// タイマーを更新（Ebitenは60FPS、1フレーム = 1/60秒）
	g.timer += 1.0 / 60.0

	// 3秒経過したかチェック
	if g.timer >= g.interval {
		g.timer = 0 // タイマーリセット
		g.placeNextPiece()
	}

	return nil
}

// placeNextPiece は次のコマを配置する
func (g *Game) placeNextPiece() {
	// 次の手があるかチェック
	if g.currentMove+1 >= g.gameData.GetMoveCount() {
		fmt.Println("対局終了")
		g.isPlaying = false
		return
	}

	// 次の手を取得して配置
	g.currentMove++
	move, err := g.gameData.GetMove(g.currentMove)
	if err != nil {
		fmt.Printf("手の取得エラー: %v\n", err)
		return
	}

	// ボードに反映
	g.board.Squares[move.Row][move.Col].Piece = &Piece{Color: move.Color}

	fmt.Printf("手%d: (%d,%d) 色=%d\n", g.currentMove+1, move.Row, move.Col, move.Color)
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
	g.drawPieces(screen)
}

// drawPieces はボード上のコマを描画する
func (g *Game) drawPieces(screen *ebiten.Image) {
	for row := 0; row < BoardSize; row++ {
		for col := 0; col < BoardSize; col++ {
			square := &g.board.Squares[row][col]
			if !square.IsEmpty() {
				// コマの位置を計算
				centerX := float32(BoardOffset + col*CellSize + CellSize/2)
				centerY := float32(BoardOffset + row*CellSize + CellSize/2)
				radius := float32(CellSize/2 - 4) // 少し余白を残す

				// 色を取得してRGBAに変換
				pieceColor := colorToRGB(square.Piece.Color)

				// 円を描画
				vector.DrawFilledCircle(screen, centerX, centerY, radius, pieceColor, false)

				// 縁を描画（見やすくするため）
				vector.StrokeCircle(screen, centerX, centerY, radius, 1, color.Black, false)
			}
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 800, 600
}

func main() {
	fmt.Println("=== リバーシ対局再生プレーヤー開始 ===")

	// ゲームインスタンスを作成
	game := NewGame()
	if game == nil {
		log.Fatal("ゲームの初期化に失敗しました")
	}

	fmt.Println("=== Ebiten Game Start ===")
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Reversi Player - Auto Play")
	if err := ebiten.RunGame(game); err != nil {
		fmt.Printf("Ebitenエラー: %v\n", err)
		log.Fatal(err)
	}
}
