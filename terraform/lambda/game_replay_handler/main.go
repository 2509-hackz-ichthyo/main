package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// GameArchive represents the archived game data structure
type GameArchive struct {
	GameId      string `json:"gameId"`
	RoomId      string `json:"roomId"`
	Player1Id   string `json:"player1Id"`
	Player2Id   string `json:"player2Id"`
	Winner      string `json:"winner"`
	GamePhase   string `json:"gamePhase"`
	EndTime     string `json:"endTime"`
	GameData    string `json:"gameData"`    // Whitespace形式
	DecodedData string `json:"decodedData"` // 10進数形式（新規追加）
}

// APIResponse represents the REST API response structure
type APIResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// RandomGameResponse represents the response body for /replay/random endpoint
type RandomGameResponse struct {
	Success bool         `json:"success"`
	Data    *GameArchive `json:"data,omitempty"`
	Message string       `json:"message,omitempty"`
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// CORS headers
	headers := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type",
		"Content-Type":                 "application/json",
	}

	// Handle OPTIONS request for CORS preflight
	if request.HTTPMethod == "OPTIONS" {
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers:    headers,
			Body:       "",
		}, nil
	}

	// Only handle GET requests
	if request.HTTPMethod != "GET" {
		response := RandomGameResponse{
			Success: false,
			Message: "Method not allowed",
		}
		body, _ := json.Marshal(response)
		return events.APIGatewayProxyResponse{
			StatusCode: 405,
			Headers:    headers,
			Body:       string(body),
		}, nil
	}

	// Get random game from archive
	gameArchive, err := getRandomGame()
	if err != nil {
		fmt.Printf("Error getting random game: %v\n", err)
		response := RandomGameResponse{
			Success: false,
			Message: "Failed to get random game",
		}
		body, _ := json.Marshal(response)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers:    headers,
			Body:       string(body),
		}, nil
	}

	if gameArchive == nil {
		response := RandomGameResponse{
			Success: false,
			Message: "No archived games found",
		}
		body, _ := json.Marshal(response)
		return events.APIGatewayProxyResponse{
			StatusCode: 404,
			Headers:    headers,
			Body:       string(body),
		}, nil
	}

	// Return successful response
	response := RandomGameResponse{
		Success: true,
		Data:    gameArchive,
	}
	body, _ := json.Marshal(response)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    headers,
		Body:       string(body),
	}, nil
}

func getRandomGame() (*GameArchive, error) {
	// Initialize AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating AWS session: %v", err)
	}

	dynamo := dynamodb.New(sess)

	// Get table name from environment variable
	tableName := os.Getenv("GAME_ARCHIVE_TABLE")
	if tableName == "" {
		tableName = "game-archive"
	}

	fmt.Printf("Scanning table: %s\n", tableName)

	// Scan the game-archive table to get all games
	scanInput := &dynamodb.ScanInput{
		TableName:        aws.String(tableName),
		FilterExpression: aws.String("SK = :sk"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":sk": {S: aws.String("ARCHIVE")},
		},
	}

	result, err := dynamo.Scan(scanInput)
	if err != nil {
		return nil, fmt.Errorf("error scanning DynamoDB table: %v", err)
	}

	if len(result.Items) == 0 {
		return nil, nil // No games found
	}

	fmt.Printf("Found %d archived games\n", len(result.Items))

	// Select a random game
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(result.Items))
	selectedItem := result.Items[randomIndex]

	// Parse the selected item into GameArchive struct
	gameArchive := &GameArchive{}

	if gameId, ok := selectedItem["gameId"]; ok && gameId.S != nil {
		gameArchive.GameId = *gameId.S
	}
	if roomId, ok := selectedItem["roomId"]; ok && roomId.S != nil {
		gameArchive.RoomId = *roomId.S
	}
	if player1Id, ok := selectedItem["player1Id"]; ok && player1Id.S != nil {
		gameArchive.Player1Id = *player1Id.S
	}
	if player2Id, ok := selectedItem["player2Id"]; ok && player2Id.S != nil {
		gameArchive.Player2Id = *player2Id.S
	}
	if winner, ok := selectedItem["winner"]; ok && winner.S != nil {
		gameArchive.Winner = *winner.S
	}
	if gamePhase, ok := selectedItem["gamePhase"]; ok && gamePhase.S != nil {
		gameArchive.GamePhase = *gamePhase.S
	}
	if endTime, ok := selectedItem["endTime"]; ok && endTime.S != nil {
		gameArchive.EndTime = *endTime.S
	}
	if gameData, ok := selectedItem["gameData"]; ok && gameData.S != nil {
		gameArchive.GameData = *gameData.S
	}

	fmt.Printf("Selected game: %s (winner: %s)\n", gameArchive.GameId, gameArchive.Winner)

	// Whitespace→10進数変換を実行
	if gameArchive.GameData != "" {
		decodedData, err := convertWhitespaceToDecimal(gameArchive.GameData)
		if err != nil {
			fmt.Printf("Warning: Failed to decode whitespace data: %v\n", err)
			// エラーでも処理続行（Whitespace形式のまま返却）
		} else {
			gameArchive.DecodedData = decodedData
			fmt.Printf("Successfully converted whitespace to decimal format\n")
		}
	}

	return gameArchive, nil
}

// convertWhitespaceToDecimal はWhitespace形式データを10進数形式に変換する
func convertWhitespaceToDecimal(whitespaceData string) (string, error) {
	// 固定IPアドレスを使用（ECS Fargateへのアクセス）
	apiURL := "http://18.181.38.132:3000"
	fmt.Printf("Converting whitespace to decimal using API: %s\n", apiURL)
	fmt.Printf("Input whitespace data length: %d, first 100 chars: %q\n", len(whitespaceData), whitespaceData[:min(100, len(whitespaceData))])

	if whitespaceData == "" {
		return "", fmt.Errorf("empty whitespace data")
	}

	// Split whitespace data into sentences (3 lines each)
	// Each sentence follows pattern: SSS{4bit}LSSS{4bit}LSSS{8bit}L
	lines := strings.Split(whitespaceData, "\n")

	// Filter out empty lines
	var validLines []string
	for _, line := range lines {
		if line != "" {
			validLines = append(validLines, line)
		}
	}

	fmt.Printf("Total lines after filtering: %d\n", len(validLines))

	// Group into sentences (3 lines each)
	var sentences []string
	for i := 0; i < len(validLines); i += 3 {
		if i+2 < len(validLines) {
			sentence := validLines[i] + "\n" + validLines[i+1] + "\n" + validLines[i+2] + "\n"
			sentences = append(sentences, sentence)
			fmt.Printf("Created sentence %d: %q\n", len(sentences), sentence[:min(50, len(sentence))])
		} else {
			fmt.Printf("Warning: Incomplete sentence at line %d, remaining lines: %d\n", i, len(validLines)-i)
		}
	}

	if len(sentences) == 0 {
		return "", fmt.Errorf("no valid whitespace sentences found in data")
	}

	fmt.Printf("Total sentences created: %d\n", len(sentences))

	// Prepare the request payload
	reqBody := map[string]interface{}{
		"command_type": "WhitespaceToDecimal",
		"payload":      sentences, // Array of sentences instead of single string
	}

	fmt.Printf("Request body prepared with %d sentences\n", len(sentences))

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}
	fmt.Printf("JSON request prepared (length: %d)\n", len(jsonData))

	resp, err := http.Post(apiURL+"/v1/decode", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to call decode API: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Response status code: %d\n", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		// レスポンスボディを読んでエラー内容を確認
		bodyBytes := make([]byte, 1024)
		n, _ := resp.Body.Read(bodyBytes)
		return "", fmt.Errorf("decode API returned status code: %d, body: %s", resp.StatusCode, string(bodyBytes[:n]))
	}

	// Decode APIレスポンス形式に合わせて解析
	var apiResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}
	fmt.Printf("Decode API response keys: %v\n", getMapKeys(apiResp))

	// result_decimalsフィールドから結果を取得（配列形式）
	if resultArray, ok := apiResp["result_decimals"].([]interface{}); ok && len(resultArray) > 0 {
		// 配列の各要素を文字列として連結
		var decimalLines []string
		for _, item := range resultArray {
			if line, ok := item.(string); ok {
				decimalLines = append(decimalLines, line)
			}
		}
		result := strings.Join(decimalLines, "\n")
		fmt.Printf("Converted decimal data (lines: %d): %q\n", len(decimalLines), result[:min(200, len(result))])
		return result, nil
	}

	return "", fmt.Errorf("empty or invalid decode result, full response keys: %v", getMapKeys(apiResp))
}

// helper function to get map keys for debugging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	lambda.Start(handleRequest)
}
