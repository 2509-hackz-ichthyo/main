package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Game struct {
	gameData    *GameData // パースした対局データ
	board       *Board    // 現在のボード状態
	currentMove int       // 現在の手数
	timer       float64   // 経過時間（秒）
	interval    float64   // コマ配置間隔（3秒）
	isPlaying   bool      // 再生中フラグ
	is24Mode    bool      // 24-7モードフラグ
	isLoading   bool      // データ読み込み中フラグ
}

// GameArchive はAPIから返される対局データの構造体
type GameArchive struct {
	GameId    string `json:"gameId"`
	RoomId    string `json:"roomId"`
	Player1Id string `json:"player1Id"`
	Player2Id string `json:"player2Id"`
	Winner    string `json:"winner"`
	GamePhase string `json:"gamePhase"`
	EndTime   string `json:"endTime"`
	GameData  string `json:"gameData"`
}

// RandomGameResponse は /replay/random APIのレスポンス
type RandomGameResponse struct {
	Success bool         `json:"success"`
	Data    *GameArchive `json:"data,omitempty"`
	Message string       `json:"message,omitempty"`
}

const (
	BoardSize   = 8
	CellSize    = 60
	BoardOffset = 50
)

// colorToRGB は0-255の色値をRGBに変換する
func colorToRGB(c uint8) color.RGBA {
	return color.RGBA{R: c, G: c, B: c, A: 255}
}

// checkIs24Mode はURLから24-7モードかどうかを判定する
func checkIs24Mode() bool {
	location := js.Global().Get("location")
	hash := location.Get("hash").String()
	return hash == "#24-7" || hash == "#247"
}

// fetchRandomGameData はランダムな対局データを取得する
func fetchRandomGameData() (*GameArchive, error) {
	// Terraformから取得した実際のAPI URL
	apiURL := "https://ut7hbw3323.execute-api.ap-northeast-1.amazonaws.com/prod/replay/random"
	
	promise := js.Global().Get("fetch").Invoke(apiURL)
	
	// Promise を同期的に待つためのチャネル
	resultCh := make(chan *GameArchive, 1)
	errorCh := make(chan error, 1)
	
	// then ハンドラー
	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		response := args[0]
		
		// レスポンスを JSON として解析
		jsonPromise := response.Call("json")
		jsonPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			data := args[0]
			
			// JavaScriptオブジェクトを Go 構造体に変換
			jsonStr := js.Global().Get("JSON").Call("stringify", data).String()
			
			var resp RandomGameResponse
			if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
				errorCh <- fmt.Errorf("JSON parse error: %v", err)
				return nil
			}
			
			if !resp.Success || resp.Data == nil {
				errorCh <- fmt.Errorf("API error: %s", resp.Message)
				return nil
			}
			
			resultCh <- resp.Data
			return nil
		}))
		
		return nil
	}))
	
	// catch ハンドラー
	promise.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		error := args[0]
		errorCh <- fmt.Errorf("fetch error: %s", error.Get("message").String())
		return nil
	}))
	
	// 結果を待つ
	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errorCh:
		return nil, err
	}
}

// NewGame は新しいゲームインスタンスを作成する
func NewGame() *Game {
	is24Mode := checkIs24Mode()
	
	game := &Game{
		board:       &Board{},
		currentMove: -1,     // まだ何も置いていない状態
		timer:       0,      // タイマー初期化
		interval:    1.0,    // 1秒間隔
		isPlaying:   false,  // データ読み込み完了後に開始
		is24Mode:    is24Mode,
		isLoading:   true,   // 読み込み中
	}

	if is24Mode {
		fmt.Println("24-7モードで開始します")
		// 非同期で対局データを取得
		go func() {
			if err := game.loadNext24Data(); err != nil {
				fmt.Printf("24-7データ読み込みエラー: %v\n", err)
				// エラー時はモックデータで継続
				game.loadMockData()
			}
		}()
	} else {
		fmt.Println("通常モードで開始します")
		game.loadMockData()
	}

	return game
}

// loadMockData はモックデータを読み込む
func (g *Game) loadMockData() {
	gameData, err := parseGameData(mockPlayData)
	if err != nil {
		fmt.Printf("モックデータ解析エラー: %v\n", err)
		return
	}
	
	g.gameData = gameData
	g.isLoading = false
	g.isPlaying = true
	fmt.Printf("モックデータ読み込み完了。総手数: %d\n", gameData.GetMoveCount())
}

// loadNext24Data は24-7モード用の次の対局データを読み込む
func (g *Game) loadNext24Data() error {
	fmt.Println("ランダム対局データを取得中...")
	
	// ランダム対局データを取得
	gameArchive, err := fetchRandomGameData()
	if err != nil {
		return fmt.Errorf("対局データ取得失敗: %v", err)
	}
	
	fmt.Printf("対局データ取得成功: GameID=%s\n", gameArchive.GameId)
	
	// 対局データは既に通常形式なので、直接パース
	fmt.Println("対局データをパース中...")
	gameData, err := parseGameData(gameArchive.GameData)
	if err != nil {
		return fmt.Errorf("パース失敗: %v", err)
	}
	
	// ゲーム状態を更新
	g.gameData = gameData
	g.board = &Board{}  // ボードをリセット
	g.currentMove = -1  // 手数をリセット
	g.timer = 0         // タイマーリセット
	g.isLoading = false
	g.isPlaying = true
	
	fmt.Printf("24-7データ読み込み完了。総手数: %d\n", gameData.GetMoveCount())
	return nil
}

func (g *Game) Update() error {
	// データ読み込み中は更新しない
	if g.isLoading || !g.isPlaying {
		return nil
	}

	// タイマーを更新（Ebitenは60FPS、1フレーム = 1/60秒）
	g.timer += 1.0 / 60.0

	// 1秒経過したかチェック
	if g.timer >= g.interval {
		g.timer = 0 // タイマーリセット
		g.placeNextPiece()
	}

	return nil
}

// placeNextPiece は次のコマを配置する
func (g *Game) placeNextPiece() {
	if g.gameData == nil {
		return
	}

	// 次の手があるかチェック
	if g.currentMove+1 >= g.gameData.GetMoveCount() {
		fmt.Println("対局終了")
		g.isPlaying = false
		
		// 24-7モードの場合、次の対局を自動読み込み
		if g.is24Mode {
			fmt.Println("次の対局を読み込み中...")
			g.isLoading = true
			go func() {
				if err := g.loadNext24Data(); err != nil {
					fmt.Printf("次の対局読み込みエラー: %v\n", err)
					// エラー時は1秒後に再試行
					// TODO: より堅牢なエラーハンドリングを追加
				}
			}()
		}
		return
	}

	// 次の手を取得して配置
	g.currentMove++
	move, err := g.gameData.GetMove(g.currentMove)
	if err != nil {
		fmt.Printf("手の取得エラー: %v\n", err)
		return
	}

	// ボードに反映
	g.board.Squares[move.Row][move.Col].Piece = &Piece{Color: move.Color}

	fmt.Printf("手%d: (%d,%d) 色=%d\n", g.currentMove+1, move.Row, move.Col, move.Color)
}

func (g *Game) Draw(screen *ebiten.Image) {
	// 画面を薄いグレーでクリア
	screen.Fill(color.RGBA{240, 240, 240, 255})

	// データ読み込み中の場合、読み込み中メッセージを表示
	if g.isLoading {
		// TODO: テキスト描画ライブラリを使用してメッセージを表示
		// 現在はコンソールログのみ
		return
	}

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
	g.drawPieces(screen)
}

// drawPieces はボード上のコマを描画する
func (g *Game) drawPieces(screen *ebiten.Image) {
	for row := 0; row < BoardSize; row++ {
		for col := 0; col < BoardSize; col++ {
			square := &g.board.Squares[row][col]
			if !square.IsEmpty() {
				// コマの位置を計算
				centerX := float32(BoardOffset + col*CellSize + CellSize/2)
				centerY := float32(BoardOffset + row*CellSize + CellSize/2)
				radius := float32(CellSize/2 - 4) // 少し余白を残す

				// 色を取得してRGBAに変換
				pieceColor := colorToRGB(square.Piece.Color)

				// 円を描画
				vector.DrawFilledCircle(screen, centerX, centerY, radius, pieceColor, false)

				// 縁を描画（見やすくするため）
				vector.StrokeCircle(screen, centerX, centerY, radius, 1, color.Black, false)
			}
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 800, 600
}

func main() {
	fmt.Println("=== リバーシ対局再生プレーヤー開始 ===")

	// ゲームインスタンスを作成
	game := NewGame()
	if game == nil {
		log.Fatal("ゲームの初期化に失敗しました")
	}

	fmt.Println("=== Ebiten Game Start ===")
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Reversi Player - Auto Play")
	if err := ebiten.RunGame(game); err != nil {
		fmt.Printf("Ebitenエラー: %v\n", err)
		log.Fatal(err)
	}
}
