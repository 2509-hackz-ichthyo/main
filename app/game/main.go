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
	// 黒側 (0-127) のコマ
	board.Squares[3][3].Piece = &Piece{Color: 63} // 黒のコマ
	board.Squares[4][4].Piece = &Piece{Color: 63} // 黒のコマ

	// 白側 (128-255) のコマ
	board.Squares[3][4].Piece = &Piece{Color: 191} // 白のコマ
	board.Squares[4][3].Piece = &Piece{Color: 191} // 白のコマ

	return board
}

// Game はゲームの状態を表す
type Game struct {
	Board       *Board
	CurrentTurn bool  // trueが黒 (0-127)、falseが白 (128-255)
	NextColor   uint8 // 次に配置するコマの色
	Rand        *rand.Rand
}

const (
	BoardSize   = 8
	CellSize    = 60
	BoardOffset = 50
)

func (g *Game) Update() error {
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
	// シンプルな色マッピング：緑チャンネルに値を使用し、
	// 黒側/白側でコントラストを作成
	if c < 128 {
		// 黒側：暗い色
		return color.RGBA{R: c, G: c, B: c, A: 255}
	} else {
		// 白側：明るい色
		adjusted := c - 128
		return color.RGBA{R: 128 + adjusted/2, G: 128 + adjusted/2, B: 128 + adjusted/2, A: 255}
	}
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

	// UI情報を描画
	currentPlayer := g.getCurrentPlayerSide()
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

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 800, 600
}

// NewGame は新しいゲームインスタンスを作成する
func NewGame() *Game {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	game := &Game{
		Board:       NewBoard(),
		CurrentTurn: true, // 黒から開始
		Rand:        r,
	}

	// 最初の手の色を生成
	game.generateNextColor()

	return game
}

// generateNextColor は現在のターンに基づいて次のコマの色を生成する
func (g *Game) generateNextColor() {
	if g.CurrentTurn {
		// 黒側: 0-127
		g.NextColor = uint8(g.Rand.Intn(128))
	} else {
		// 白側: 128-255
		g.NextColor = uint8(128 + g.Rand.Intn(128))
	}
}

// isBlackSide は色が黒側 (0-127) に属するかを返す
func isBlackSide(color uint8) bool {
	return color < 128
}

// isWhiteSide は色が白側 (128-255) に属するかを返す
func isWhiteSide(color uint8) bool {
	return color >= 128
}

// switchTurn はターンを切り替えて次の色を生成する
func (g *Game) switchTurn() {
	g.CurrentTurn = !g.CurrentTurn
	g.generateNextColor()
}

// getCurrentPlayerSide は現在のプレイヤーの側を文字列で返す
func (g *Game) getCurrentPlayerSide() string {
	if g.CurrentTurn {
		return "黒"
	}
	return "白"
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
func (g *Game) findFlankingPieces(x, y int, color uint8) [][]Position {
	if !g.Board.Squares[x][y].IsEmpty() {
		return nil
	}

	var flankingLines [][]Position
	currentSide := isBlackSide(color)

	for _, dir := range directions {
		var line []Position
		nx, ny := x+dir.dx, y+dir.dy

		// 反対側のコマを探す
		for isValidPosition(nx, ny) && !g.Board.Squares[nx][ny].IsEmpty() {
			piece := g.Board.Squares[nx][ny].Piece
			if isBlackSide(piece.Color) == currentSide {
				// 同じ側のコマを発見、挟むコマがあればこのラインは有効
				if len(line) > 0 {
					flankingLines = append(flankingLines, line)
				}
				break
			} else {
				// 反対側のコマ、潜在的な挟みラインに追加
				line = append(line, Position{nx, ny})
			}
			nx, ny = nx+dir.dx, ny+dir.dy
		}
	}

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

	flankingLines := g.findFlankingPieces(x, y, g.NextColor)
	return len(flankingLines) > 0
}

// placePiece は(x, y)にコマを置き、ルールに従って色変更を適用する
func (g *Game) placePiece(x, y int) bool {
	if !g.isValidMove(x, y) {
		return false
	}

	// 新しいコマを配置
	g.Board.Squares[x][y].Piece = &Piece{Color: g.NextColor}

	// すべての挟みラインを見つける
	flankingLines := g.findFlankingPieces(x, y, g.NextColor)

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

	// ターンを切り替えて次の色を生成
	g.switchTurn()
	return true
}

func main() {
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("アンミカリバーシ")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
