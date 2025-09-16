package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	// クロスオリジン WebSocket 接続の検証関数: デバック用に全部許可
	CheckOrigin: func(r *http.Request) bool { return true },
}

func wsHandler(c *gin.Context) {
	// ハンドシェイク
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer conn.Close()

	// WebSocket セッション本体: 接続が切れるまでメッセージを読み取り、エコー返信する
	for {
		mt, msg, err := conn.ReadMessage() // mt: message type
		if err != nil {
			log.Println("read:", err)
			break
		}
		if err := conn.WriteMessage(mt, msg); err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) { c.String(200, "Hello, World!") })
	r.GET("/api/ws", wsHandler)
	r.Run(":3000")
}
