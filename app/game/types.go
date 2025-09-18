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

	// WebSocket エンドポイント
	WebSocketURL = "wss://amsdtyktxi.execute-api.ap-northeast-1.amazonaws.com/production"
)

// ゲーム状態（WebSocket接続含む）
type GameState int

const (
	GameStateDisconnected GameState = iota // WebSocket未接続
	GameStateConnecting                    // WebSocket接続中
	GameStateWaiting                       // マッチメイキング待機中
	GameStateInGame                        // ゲーム中
	GameStateError                         // エラー状態
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

	// WebSocket関連
	State        GameState     // 現在のゲーム状態
	WSConnection *WSConnection // WebSocket接続
	PlayerID     string        // プレイヤーID
	RoomID       string        // ルームID
	PlayerRole   string        // プレイヤーロール ("black" または "white")
	IsOnline     bool          // オンラインモードフラグ
	ErrorMessage string        // エラーメッセージ

	// 対局記録関連
	GameRecord *GameRecord // 対局記録

	// デバッグモード関連
	DebugMode bool // デバッグモードフラグ
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

// GameMove は1手の記録を表す
type GameMove struct {
	TurnNumber int    // ターン番号
	Player     string // プレイヤー("黒"/"白")
	Row        int    // 行座標(0-7)
	Col        int    // 列座標(0-7)
	Color      uint8  // 配置したコマの色(0-255)
	Timestamp  string // 配置時刻(RFC3339形式)
}

// GameRecord は対局記録を表す
type GameRecord struct {
	GameID    string     // ゲームID
	Player1ID string     // プレイヤー1のID
	Player2ID string     // プレイヤー2のID(オフライン時は"CPU")
	StartTime string     // ゲーム開始時刻
	EndTime   string     // ゲーム終了時刻
	Winner    string     // 勝者("黒"/"白"/"引き分け")
	Moves     []GameMove // 全ての手の記録
	GameMode  string     // ゲームモード("online"/"offline")
	TurnCount int        // 総ターン数
}
