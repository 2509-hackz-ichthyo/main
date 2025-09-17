package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Move は対局の1手を表す
type Move struct {
	Row   int   // 行位置 (0-7)
	Col   int   // 列位置 (0-7)
	Color uint8 // 色値 (0-255)
}

// GameData は対局データ全体を表す
type GameData struct {
	Moves []Move // 対局の手順
}

// parseGameData は文字列形式の対局データを構造化されたGameDataに変換する
func parseGameData(data string) (*GameData, error) {
	gameData := &GameData{}
	lines := strings.Split(strings.TrimSpace(data), "\n")

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue // 空行をスキップ
		}

		parts := strings.Fields(line)
		if len(parts) != 3 {
			return nil, fmt.Errorf("line %d: expected 3 fields, got %d", lineNum+1, len(parts))
		}

		row, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid row value '%s': %v", lineNum+1, parts[0], err)
		}

		col, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid col value '%s': %v", lineNum+1, parts[1], err)
		}

		colorInt, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid color value '%s': %v", lineNum+1, parts[2], err)
		}

		// 範囲チェック
		if row < 0 || row > 7 {
			return nil, fmt.Errorf("line %d: row %d out of range (0-7)", lineNum+1, row)
		}
		if col < 0 || col > 7 {
			return nil, fmt.Errorf("line %d: col %d out of range (0-7)", lineNum+1, col)
		}
		if colorInt < 0 || colorInt > 255 {
			return nil, fmt.Errorf("line %d: color %d out of range (0-255)", lineNum+1, colorInt)
		}

		move := Move{
			Row:   row,
			Col:   col,
			Color: uint8(colorInt),
		}

		gameData.Moves = append(gameData.Moves, move)
	}

	return gameData, nil
}

// Square はコマを最大1つまで保持できるボードのマスを表す（gameパッケージ互換）
type Square struct {
	Piece *Piece // 空の場合はnil、そうでなければコマを格納
}

// Piece は0-255の色を持つゲームの駒を表す（gameパッケージ互換）
type Piece struct {
	Color uint8 // 0-255: 0-127が黒側、128-255が白側
}

// Board は8x8のリバーシ盤を表す（gameパッケージ互換）
type Board struct {
	Squares [8][8]Square
}

// IsEmpty はマスが空かどうかを返す
func (s *Square) IsEmpty() bool {
	return s.Piece == nil
}

// ApplyToBoard は指定した手数までをボードに適用する
func (g *GameData) ApplyToBoard(board *Board, moveIndex int) error {
	if moveIndex < 0 {
		return fmt.Errorf("moveIndex cannot be negative")
	}
	if moveIndex >= len(g.Moves) {
		return fmt.Errorf("moveIndex %d exceeds available moves (%d)", moveIndex, len(g.Moves))
	}

	// ボードをクリア
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			board.Squares[row][col].Piece = nil
		}
	}

	// 指定したインデックスまでの手を適用
	for i := 0; i <= moveIndex; i++ {
		move := g.Moves[i]
		board.Squares[move.Row][move.Col].Piece = &Piece{Color: move.Color}
	}

	return nil
}

// Validate は対局データの整合性をチェックする
func (g *GameData) Validate() error {
	for i, move := range g.Moves {
		if move.Row < 0 || move.Row > 7 {
			return fmt.Errorf("move %d: row %d out of range (0-7)", i, move.Row)
		}
		if move.Col < 0 || move.Col > 7 {
			return fmt.Errorf("move %d: col %d out of range (0-7)", i, move.Col)
		}
		// Color uint8 は自動的に 0-255 の範囲になる
	}
	return nil
}

// GetMoveCount は対局の手数を返す
func (g *GameData) GetMoveCount() int {
	return len(g.Moves)
}

// GetMove は指定したインデックスの手を返す
func (g *GameData) GetMove(index int) (Move, error) {
	if index < 0 || index >= len(g.Moves) {
		return Move{}, fmt.Errorf("index %d out of range (0-%d)", index, len(g.Moves)-1)
	}
	return g.Moves[index], nil
}

// GetColorRange は対局データに含まれる色値の最小・最大値を返す
func (g *GameData) GetColorRange() (min, max uint8) {
	if len(g.Moves) == 0 {
		return 0, 0
	}

	min = g.Moves[0].Color
	max = g.Moves[0].Color

	for _, move := range g.Moves {
		if move.Color < min {
			min = move.Color
		}
		if move.Color > max {
			max = move.Color
		}
	}

	return min, max
}
