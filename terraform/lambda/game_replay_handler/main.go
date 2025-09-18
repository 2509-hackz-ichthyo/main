package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// GameArchive represents the archived game data structure
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
		TableName: aws.String(tableName),
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
	
	return gameArchive, nil
}

func main() {
	lambda.Start(handleRequest)
}