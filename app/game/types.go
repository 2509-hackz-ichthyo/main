package main

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// 定数定義
const (
	BoardSize   = 8
	CellSize    = 60
	BoardOffset = 50
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

// Game はゲームの状態を表す
type Game struct {
	Board       *Board
	CurrentTurn bool  // trueが黒 (0-127)、falseが白 (128-255)
	NextColor   uint8 // 次に配置するコマの色
	Rand        *rand.Rand
	GameOver    bool             // ゲーム終了フラグ
	Winner      string           // 勝者（"黒" または "白"）
	FontFace    *text.GoTextFace // 勝利メッセージ用フォント
}

// Position はボード上の位置を表す
type Position struct {
	X, Y int
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
