package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// NewGame は新しいゲームインスタンスを作成する
func NewGame() *Game {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	game := &Game{
		Board:       NewBoard(),
		CurrentTurn: true, // プレイヤー1から開始
		Rand:        r,
	}

	// 最初の手の色を生成
	game.generateNextColor()

	// フォントを初期化
	if err := game.initializeFont(); err != nil {
		log.Printf("Failed to initialize font: %v", err)
	}

	return game
}

func (g *Game) Update() error {
	// ゲーム終了後は入力を受け付けない
	if g.GameOver {
		return nil
	}

	// マウスクリックを処理
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()

		// 画面座標をボード座標に変換
		boardX := (mx - BoardOffset) / CellSize
		boardY := (my - BoardOffset) / CellSize

		// クリックがボード範囲内かチェックしてコマを配置
		if boardX >= 0 && boardX < BoardSize && boardY >= 0 && boardY < BoardSize {
			g.placePiece(boardX, boardY)
		}
	}

	return nil
}

// generateNextColor はランダムに次のコマの色を生成する
func (g *Game) generateNextColor() {
	// 0-255の全範囲からランダムに色を生成
	g.NextColor = uint8(g.Rand.Intn(256))
}

// switchTurn はターンを切り替えて次の色を生成する
func (g *Game) switchTurn() {
	g.CurrentTurn = !g.CurrentTurn
	g.generateNextColor()
}

// getCurrentPlayer は現在のプレイヤー番号を文字列で返す
func (g *Game) getCurrentPlayer() string {
	if g.CurrentTurn {
		return "プレイヤー1"
	}
	return "プレイヤー2"
}

// calculateWinner は全駒の色の平均値から勝者を決定する
func (g *Game) calculateWinner() {
	var colorSum int
	var pieceCount int

	for x := 0; x < BoardSize; x++ {
		for y := 0; y < BoardSize; y++ {
			if !g.Board.Squares[x][y].IsEmpty() {
				colorSum += int(g.Board.Squares[x][y].Piece.Color)
				pieceCount++
			}
		}
	}

	if pieceCount == 0 {
		g.Winner = "引き分け"
		return
	}

	average := float64(colorSum) / float64(pieceCount)

	// 平均値が127.5未満なら黒、以上なら白の勝利
	if average < 127.5 {
		g.Winner = "黒"
	} else {
		g.Winner = "白"
	}
}
