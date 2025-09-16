package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Piece は0-255の色を持つゲームの駒を表す
type Piece struct {
	Color uint8 // 0-255: 0-127が黒側、128-255が白側
}

// Square はコマを最大1つまで保持できるボードのマスを表す
type Square struct {
	Piece *Piece // 空の場合はnil、そうでなければコマを格納
}

// IsEmpty はマスが空かどうかを返す
func (s *Square) IsEmpty() bool {
	return s.Piece == nil
}

// Board は8x8のリバーシ盤を表す
type Board struct {
	Squares [8][8]Square
}

// NewBoard は初期設定済みの新しいボードを作成する
func NewBoard() *Board {
	board := &Board{}

	// 中央に開始時のコマを配置
	board.Squares[3][3].Piece = &Piece{Color: 0}   // 初期コマ（黒色）
	board.Squares[4][4].Piece = &Piece{Color: 0}   // 初期コマ（黒色）
	board.Squares[3][4].Piece = &Piece{Color: 255} // 初期コマ（白色）
	board.Squares[4][3].Piece = &Piece{Color: 255} // 初期コマ（白色）

	return board
}

// Game はゲームの状態を表す
type Game struct {
	Board       *Board
	CurrentTurn bool  // trueが黒 (0-127)、falseが白 (128-255)
	NextColor   uint8 // 次に配置するコマの色
	Rand        *rand.Rand
	GameOver    bool   // ゲーム終了フラグ
	Winner      string // 勝者（"黒" または "白"）
}

const (
	BoardSize   = 8
	CellSize    = 60
	BoardOffset = 50
)

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

// colorToRGB は0-255の色値をRGBに変換する
func colorToRGB(c uint8) color.RGBA {
	return color.RGBA{R: c, G: c, B: c, A: 255}
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
	if g.GameOver {
		// 画面中央に勝利メッセージを表示
		winMessage := fmt.Sprintf("どちらかというと %s の勝利！", g.Winner)
		
		// メッセージを画面中央に配置
		screenWidth, screenHeight := 800, 600
		messageX := float64(screenWidth / 2)
		messageY := float64(screenHeight / 2)
		
		// 背景の四角形を描画（見やすくするため）
		bgX := float32(messageX - 150)
		bgY := float32(messageY - 30)
		bgWidth := float32(300)
		bgHeight := float32(60)
		
		vector.DrawFilledRect(screen, bgX, bgY, bgWidth, bgHeight, color.RGBA{255, 255, 255, 200}, false)
		vector.StrokeRect(screen, bgX, bgY, bgWidth, bgHeight, 3, color.Black, false)
		
		// テキストを描画（ebitenutil.DebugPrintAtを使用）
		ebitenutil.DebugPrintAt(screen, winMessage, int(messageX-100), int(messageY-10))
	} else {
		// 通常のUI情報を描画
		currentPlayer := g.getCurrentPlayer()
		nextColorRGB := colorToRGB(g.NextColor)

		infoText := fmt.Sprintf("現在のプレイヤー: %s\n次の色: %d\nクリックでコマを配置",
			currentPlayer, g.NextColor)
		ebitenutil.DebugPrint(screen, infoText)

		// 次の色のプレビューを描画
		previewX := float32(BoardOffset + BoardSize*CellSize + 20)
		previewY := float32(BoardOffset + 60)
		vector.DrawFilledCircle(screen, previewX, previewY, 20, nextColorRGB, false)
		vector.StrokeCircle(screen, previewX, previewY, 20, 2, color.Black, false)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 800, 600
}

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

	return game
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

// Direction はボード上の8方向を表す
type Direction struct {
	dx, dy int
}

var directions = []Direction{
	{-1, -1}, {-1, 0}, {-1, 1},
	{0, -1}, {0, 1},
	{1, -1}, {1, 0}, {1, 1},
}

// isValidPosition は位置がボードの境界内かをチェックする
func isValidPosition(x, y int) bool {
	return x >= 0 && x < 8 && y >= 0 && y < 8
}

// findFlankingPieces は(x, y)にコマを置いた時に挟まれるコマを見つける
func (g *Game) findFlankingPieces(x, y int) [][]Position {
	if !g.Board.Squares[x][y].IsEmpty() {
		return nil
	}

	var flankingLines [][]Position

	for _, dir := range directions {
		var line []Position
		nx, ny := x+dir.dx, y+dir.dy

		// この方向にコマがあるかを探す
		for isValidPosition(nx, ny) && !g.Board.Squares[nx][ny].IsEmpty() {
			// コマを潜在的な挟みラインに追加
			line = append(line, Position{nx, ny})
			nx, ny = nx+dir.dx, ny+dir.dy
		}

		// ラインに少なくとも2つのコマがある場合、挟みが成立
		// （最低でも1つのコマを挟み、もう1つで終端とする）
		if len(line) >= 2 {
			// 最奥のコマは対象から外す
			line = line[:len(line)-1]
			flankingLines = append(flankingLines, line)
		}
	}

	fmt.Println("Found flanking lines:", flankingLines)
	return flankingLines
}

// Position はボード上の位置を表す
type Position struct {
	X, Y int
}

// isValidMove は(x, y)にコマを置くことが有効な手かをチェックする
func (g *Game) isValidMove(x, y int) bool {
	if !isValidPosition(x, y) || !g.Board.Squares[x][y].IsEmpty() {
		return false
	}

	flankingLines := g.findFlankingPieces(x, y)
	return len(flankingLines) > 0
}

// isBoardFull は全てのマスが埋まっているかを判定する
func (g *Game) isBoardFull() bool {
	for x := 0; x < BoardSize; x++ {
		for y := 0; y < BoardSize; y++ {
			if g.Board.Squares[x][y].IsEmpty() {
				return false
			}
		}
	}
	return true
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

// placePiece は(x, y)にコマを置き、ルールに従って色変更を適用する
func (g *Game) placePiece(x, y int) bool {
	if !g.isValidMove(x, y) {
		return false
	}

	// すべての挟みラインを見つける
	flankingLines := g.findFlankingPieces(x, y)
	fmt.Println("Flanking Lines:", flankingLines)

	// 各挟みラインを処理
	for _, line := range flankingLines {
		if len(line) > 0 {
			// 挟んでいるコマを取得
			end := line[len(line)-1]

			a1 := g.NextColor                               // 新しく配置されたコマ
			a2 := g.Board.Squares[end.X][end.Y].Piece.Color // 遠端の挟んでいるコマ

			// 間のすべてのコマに色変更を適用
			for _, pos := range line {
				piece := g.Board.Squares[pos.X][pos.Y].Piece
				b1 := piece.Color

				// 色変更式を適用: c = (a1 + a2 + b1) / 3
				newColor := uint8((uint16(a1) + uint16(a2) + uint16(b1)) / 3)
				piece.Color = newColor
			}
		}
	}

	// 新しいコマを配置
	g.Board.Squares[x][y].Piece = &Piece{Color: g.NextColor}

	// ゲーム終了判定
	if g.isBoardFull() {
		g.GameOver = true
		g.calculateWinner()
	} else {
		// ターンを切り替えて次の色を生成
		g.switchTurn()
	}
	
	return true
}

func main() {
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("アンミカリバーシ")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
