package main

import (
	"encoding/json"
	"fmt"
	"log"
	"syscall/js"
	"time"
)

// WebSocket接続状態
type ConnectionState int

const (
	Disconnected ConnectionState = iota
	Connecting
	Connected
	Error
)

// WebSocketメッセージ構造体
type WSMessage struct {
	Action   string      `json:"action,omitempty"`
	Type     string      `json:"type,omitempty"`
	Data     interface{} `json:"data,omitempty"`
	PlayerID string      `json:"playerId,omitempty"`
	RoomID   string      `json:"roomId,omitempty"`
	Role     string      `json:"role,omitempty"`
	X        int         `json:"x,omitempty"`
	Y        int         `json:"y,omitempty"`
	Color    uint8       `json:"color,omitempty"`
}

// WebSocketコネクション管理
type WSConnection struct {
	websocket js.Value
	state     ConnectionState
	onMessage func(WSMessage)
	onConnect func()
	onError   func(error)
}

// WebSocket接続を作成
func NewWebSocketConnection() *WSConnection {
	return &WSConnection{
		state: Disconnected,
	}
}

// WebSocketサーバーに接続
func (ws *WSConnection) Connect(url string) error {
	if ws.state == Connecting || ws.state == Connected {
		return fmt.Errorf("already connecting or connected")
	}

	ws.state = Connecting

	// JavaScript WebSocket オブジェクトを作成
	websocket := js.Global().Get("WebSocket")
	if websocket.IsUndefined() {
		ws.state = Error
		return fmt.Errorf("WebSocket not supported")
	}

	ws.websocket = websocket.New(url)

	// WebSocket イベントハンドラーを設定
	ws.setupEventHandlers()

	return nil
}

// WebSocketイベントハンドラーの設定
func (ws *WSConnection) setupEventHandlers() {
	// 接続成功時
	ws.websocket.Set("onopen", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Printf("WebSocket connected")
		ws.state = Connected
		if ws.onConnect != nil {
			ws.onConnect()
		}
		return nil
	}))

	// メッセージ受信時
	ws.websocket.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return nil
		}

		event := args[0]
		data := event.Get("data").String()

		log.Printf("WebSocket message received: %s", data)

		var message WSMessage
		if err := json.Unmarshal([]byte(data), &message); err != nil {
			log.Printf("Failed to parse WebSocket message: %v", err)
			return nil
		}

		if ws.onMessage != nil {
			ws.onMessage(message)
		}
		return nil
	}))

	// エラー時
	ws.websocket.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Printf("WebSocket error occurred")
		ws.state = Error
		if ws.onError != nil {
			ws.onError(fmt.Errorf("WebSocket connection error"))
		}
		return nil
	}))

	// 接続終了時
	ws.websocket.Set("onclose", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Printf("WebSocket connection closed")
		ws.state = Disconnected
		return nil
	}))
}

// メッセージを送信
func (ws *WSConnection) SendMessage(message WSMessage) error {
	if ws.state != Connected {
		return fmt.Errorf("WebSocket not connected")
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	log.Printf("WebSocket sending message: %s", string(data))
	ws.websocket.Call("send", string(data))
	return nil
}

// マッチメイキング要求を送信
func (ws *WSConnection) JoinGame(playerID string) error {
	message := WSMessage{
		Action:   "joinGame",
		PlayerID: playerID,
	}
	return ws.SendMessage(message)
}

// コマ配置を送信
func (ws *WSConnection) MakeMove(roomID string, x, y int, color uint8) error {
	message := WSMessage{
		Action: "makeMove",
		RoomID: roomID,
		X:      x,
		Y:      y,
		Color:  color,
	}
	return ws.SendMessage(message)
}

// 接続を閉じる
func (ws *WSConnection) Close() {
	if ws.websocket.IsUndefined() {
		return
	}

	ws.websocket.Call("close")
	ws.state = Disconnected
}

// 接続状態を取得
func (ws *WSConnection) GetState() ConnectionState {
	return ws.state
}

// イベントハンドラーを設定
func (ws *WSConnection) SetOnMessage(handler func(WSMessage)) {
	ws.onMessage = handler
}

func (ws *WSConnection) SetOnConnect(handler func()) {
	ws.onConnect = handler
}

func (ws *WSConnection) SetOnError(handler func(error)) {
	ws.onError = handler
}

// ユニークなプレイヤーIDを生成
func GeneratePlayerID() string {
	return fmt.Sprintf("player_%d", time.Now().UnixNano())
}
