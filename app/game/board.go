package main

import (
	"fmt"
	"log"
)

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

// placePiece は(x, y)にコマを置き、ルールに従って色変更を適用する
func (g *Game) placePiece(x, y int) bool {
	// オンラインモードでは、自分のターンでない場合は配置できない
	if g.IsOnline && g.State == GameStateInGame {
		// 自分のターンかどうかチェック
		if (g.PlayerRole == "black" && !g.CurrentTurn) || (g.PlayerRole == "white" && g.CurrentTurn) {
			return false
		}
	}

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

	// 手を記録
	g.recordMove(x, y, g.NextColor)

	// 注意: WebSocket送信は呼び出し元（game.go）で管理
	// ここでは盤面処理のみを行う

	// 盤面満杯の終了判定（オンライン・オフライン共通）
	if g.isBoardFull() {
		g.GameOver = true
		g.calculateWinner()

		// 対局記録を完了
		g.finishGameRecord()

		// オンラインモードの場合は、サーバーに終了を通知
		if g.IsOnline && g.State == GameStateInGame && g.WSConnection != nil {
			err := g.WSConnection.FinishGame(g.PlayerID, g.RoomID, g.Winner)
			if err != nil {
				log.Printf("Failed to send game finish notification: %v", err)
			} else {
				log.Printf("Game finished! Winner: %s - sent to server", g.Winner)
			}
		}
	} else if !g.IsOnline {
		// ローカルモードでのみターンを切り替え
		g.switchTurn()
	}

	return true
}
