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

	// エラーレスポンスのハンドリング（Type フィールドが空の場合）
	if message.Type == "" && message.Action == "" {
		// JSON構造からエラーメッセージを検出
		if dataMap, ok := message.Data.(map[string]interface{}); ok {
			if errMsg, exists := dataMap["message"]; exists {
				log.Printf("Server error message: %v", errMsg)
				if errStr, ok := errMsg.(string); ok && errStr == "Internal server error" {
					g.State = GameStateError
					g.ErrorMessage = "サーバー内部エラーが発生しました。再試行してください。"
					return
				}
			}

			// connectionIdが含まれている場合は、サーバーからのエラー応答
			if _, exists := dataMap["connectionId"]; exists {
				log.Printf("Received error response from server: %+v", dataMap)
				g.State = GameStateError
				g.ErrorMessage = "サーバーとの通信中にエラーが発生しました"
				return
			}
		}
	}

	switch message.Type {
	case "matchFound":
		log.Printf("Match found! Room: %s, Role: %s", message.RoomID, message.Role)
		g.RoomID = message.RoomID
		g.PlayerRole = message.Role
		g.State = GameStateInGame

		// プレイヤーロールに基づいてターンを設定
		if message.Role == "PLAYER1" || message.Role == "black" {
			g.CurrentTurn = true
			log.Printf("You are PLAYER1 (先手)")
		} else {
			g.CurrentTurn = false
			log.Printf("You are PLAYER2 (後手)")
		}

		// ゲーム開始時に初期盤面をリセット（必要に応じて）
		g.GameOver = false
		g.Winner = ""

	case "waiting":
		log.Printf("Waiting for opponent...")
		g.State = GameStateWaiting
		g.ErrorMessage = "新しいプレイヤーの到着を待っています..."

	case "gameUpdate":
		log.Printf("Game update received: %+v", message.Data)
		g.handleGameUpdate(message)

	case "piecePlaced":
		log.Printf("Piece placed: user=%s, row=%d, col=%d, color=%d", message.UserID, message.Row, message.Col, message.Color)
		g.handlePiecePlaced(message)

	case "opponentMove":
		log.Printf("Opponent move: x=%d, y=%d, color=%d", message.X, message.Y, message.Color)
		g.handleOpponentMove(message.X, message.Y, message.Color)

	case "error":
		log.Printf("Server error: %+v", message.Data)
		g.State = GameStateError
		g.ErrorMessage = "サーバーエラーが発生しました"

	default:
		log.Printf("Unknown message type: %s", message.Type)
		// メッセージ全体をデバッグ情報として出力
		log.Printf("Full message details: Action=%s, Type=%s, Data=%+v", message.Action, message.Type, message.Data)
	}
}

// piecePlacedメッセージを処理（新しい統一方式）
func (g *Game) handlePiecePlaced(message WSMessage) {
	// 座標変換（サーバーはrow/col、クライアントはx/y）
	x := message.Row
	y := message.Col
	color := uint8(message.Color)

	if x < 0 || x >= BoardSize || y < 0 || y >= BoardSize {
		log.Printf("Invalid piece placement coordinates: x=%d, y=%d", x, y)
		return
	}

	// 自分の手か相手の手かをチェック
	isMyMove := (message.UserID == g.PlayerID)
	
	if isMyMove {
		log.Printf("Confirmed my piece placement at (%d, %d) with color %d", x, y, color)
	} else {
		log.Printf("Processing opponent's piece placement at (%d, %d) with color %d", x, y, color)
		// 相手の手の場合、同じplacePieceロジックを適用
		g.NextColor = color
		success := g.placePiece(x, y)
		if !success {
			log.Printf("Failed to apply opponent's move - board state may be inconsistent")
		}
	}

	// ターン情報を更新
	if message.NextPlayer != "" {
		g.CurrentTurn = (message.NextPlayer == g.PlayerID)
		g.NextColor = uint8(message.NextColor)
		log.Printf("Next turn: %s (my turn: %v), next color: %d", message.NextPlayer, g.CurrentTurn, message.NextColor)
	}

	// ゲーム終了判定
	if message.GamePhase == "FINISHED" {
		g.GameOver = true
		if message.Winner != "" {
			g.Winner = message.Winner
			if message.Winner == g.PlayerID {
				log.Printf("You won!")
			} else {
				log.Printf("Opponent won!")
			}
		}
	}
}

// 相手プレイヤーのコマ配置を処理（旧方式 - 互換性のため残す）
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
	if message.GameState == nil {
		log.Printf("No game state data in update message")
		return
	}

	gameState := message.GameState
	log.Printf("Synchronizing game state: Turn %d, Current player: %s, Next color: %d",
		gameState.TurnNumber, gameState.CurrentPlayer, gameState.NextColor)

	// 盤面状態を同期
	if len(gameState.BoardState) == BoardSize {
		for x := 0; x < BoardSize; x++ {
			if len(gameState.BoardState[x]) == BoardSize {
				for y := 0; y < BoardSize; y++ {
					color := gameState.BoardState[x][y]
					if color != 0 {
						// サーバーからの盤面データを反映
						g.Board.Squares[x][y].Piece = &Piece{Color: uint8(color)}
					} else {
						// 空のマスは nil にする
						g.Board.Squares[x][y].Piece = nil
					}
				}
			}
		}
		log.Printf("Board state synchronized")
	}

	// ターン管理を同期
	g.NextColor = uint8(gameState.NextColor)

	// 現在のプレイヤーがターンかどうかを判定
	g.CurrentTurn = (gameState.CurrentPlayer == g.PlayerID)
	if g.CurrentTurn {
		log.Printf("It's your turn! Next color: %d", g.NextColor)
	} else {
		log.Printf("Waiting for opponent's turn. Next color: %d", g.NextColor)
	}

	// ゲーム終了判定
	if gameState.GamePhase == "FINISHED" {
		g.GameOver = true
		if gameState.Winner != "" {
			g.Winner = gameState.Winner
			if gameState.Winner == g.PlayerID {
				log.Printf("You won!")
			} else {
				log.Printf("Opponent won!")
			}
		}
	}
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
			// オンラインモードでは自分のターンか確認
			if g.IsOnline && !g.CurrentTurn {
				log.Printf("Not your turn!")
				return nil
			}
			
			success := g.placePiece(boardX, boardY)
			if success {
				// オンラインモードでは、サーバーにコマ配置を通知
				if g.IsOnline && g.State == GameStateInGame && g.WSConnection != nil {
					err := g.WSConnection.MakeMove(g.PlayerID, g.RoomID, boardX, boardY, g.NextColor)
					if err != nil {
						log.Printf("Failed to send move to server: %v", err)
					}
				}
			}
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
