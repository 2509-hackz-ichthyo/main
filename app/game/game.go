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

		// WebSocket関連の初期化
		State:        GameStateDisconnected,
		PlayerID:     GeneratePlayerID(),
		IsOnline:     true, // オンラインモードを有効にする
		WSConnection: NewWebSocketConnection(),
	}

	// 最初の手の色を生成
	game.generateNextColor()

	// フォントを初期化
	if err := game.initializeFont(); err != nil {
		log.Printf("Failed to initialize font: %v", err)
	}

	// WebSocket接続を開始
	game.initializeWebSocket()

	return game
}

// WebSocket接続を初期化
func (g *Game) initializeWebSocket() {
	if !g.IsOnline {
		return
	}

	log.Printf("Initializing WebSocket connection for player: %s", g.PlayerID)

	// WebSocketイベントハンドラーを設定
	g.WSConnection.SetOnConnect(func() {
		log.Printf("WebSocket connected successfully")
		g.State = GameStateConnecting

		// マッチメイキング要求を送信
		if err := g.WSConnection.JoinGame(g.PlayerID); err != nil {
			log.Printf("Failed to join game: %v", err)
			g.State = GameStateError
			g.ErrorMessage = "マッチメイキングに失敗しました"
		} else {
			g.State = GameStateWaiting
		}
	})

	g.WSConnection.SetOnMessage(func(message WSMessage) {
		g.handleWebSocketMessage(message)
	})

	g.WSConnection.SetOnError(func(err error) {
		log.Printf("WebSocket error: %v", err)
		g.State = GameStateError
		g.ErrorMessage = "サーバーとの接続に失敗しました"
	})

	// WebSocketサーバーに接続
	if err := g.WSConnection.Connect(WebSocketURL); err != nil {
		log.Printf("Failed to connect WebSocket: %v", err)
		g.State = GameStateError
		g.ErrorMessage = "サーバーに接続できませんでした"
	}
}

// WebSocketメッセージを処理
func (g *Game) handleWebSocketMessage(message WSMessage) {
	log.Printf("Handling WebSocket message: %+v", message)

	switch message.Type {
	case "matchFound":
		log.Printf("Match found! Room: %s, Role: %s", message.RoomID, message.Role)
		g.RoomID = message.RoomID
		g.PlayerRole = message.Role
		g.State = GameStateInGame

		// プレイヤーロールに基づいてターンを設定
		if message.Role == "black" {
			g.CurrentTurn = true
			log.Printf("You are playing as BLACK (先手)")
		} else {
			g.CurrentTurn = false
			log.Printf("You are playing as WHITE (後手)")
		}

		// ゲーム開始時に初期盤面をリセット（必要に応じて）
		g.GameOver = false
		g.Winner = ""

	case "gameUpdate":
		log.Printf("Game update received: %+v", message.Data)
		g.handleGameUpdate(message)

	case "opponentMove":
		log.Printf("Opponent move: x=%d, y=%d, color=%d", message.X, message.Y, message.Color)
		g.handleOpponentMove(message.X, message.Y, message.Color)

	case "error":
		log.Printf("Server error: %+v", message.Data)
		g.State = GameStateError
		g.ErrorMessage = "サーバーエラーが発生しました"

	default:
		log.Printf("Unknown message type: %s", message.Type)
	}
}

// 相手プレイヤーのコマ配置を処理
func (g *Game) handleOpponentMove(x, y int, color uint8) {
	if x < 0 || x >= BoardSize || y < 0 || y >= BoardSize {
		log.Printf("Invalid opponent move coordinates: x=%d, y=%d", x, y)
		return
	}

	// 相手のコマを配置（バリデーションは省略、サーバーで検証済みと仮定）
	g.Board.Squares[x][y].Piece = &Piece{Color: color}

	// ターンを切り替え
	g.switchTurn()

	log.Printf("Opponent placed piece at (%d, %d) with color %d", x, y, color)
}

// ゲーム状態更新を処理
func (g *Game) handleGameUpdate(message WSMessage) {
	// ここで盤面の同期やゲーム終了判定などを処理
	// 現在は基本ログ出力のみ
	log.Printf("Game state synchronization: %+v", message.Data)
}

func (g *Game) Update() error {
	// ゲーム終了後は入力を受け付けない
	if g.GameOver {
		return nil
	}

	// オンラインモードでゲーム中でない場合は、ゲーム操作を受け付けない
	if g.IsOnline && g.State != GameStateInGame {
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
