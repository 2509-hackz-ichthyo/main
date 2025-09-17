package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// MakeMoveRequest represents the request body for makeMove
type MakeMoveRequest struct {
	Action string `json:"action"`
	UserId string `json:"userId"`
	RoomId string `json:"roomId"`
	Row    int    `json:"row"`
	Col    int    `json:"col"`
	Color  int    `json:"color"`
}

// GameState represents the current game state (simplified - no board state)
type GameState struct {
	RoomId        string `json:"roomId"`
	TurnNumber    int    `json:"turnNumber"`
	CurrentPlayer string `json:"currentPlayer"`
	NextColor     int    `json:"nextColor"`
	GamePhase     string `json:"gamePhase"`
	Winner        string `json:"winner,omitempty"`
}

// GameUpdateResponse represents the response sent to players
type GameUpdateResponse struct {
	Type      string    `json:"type"`
	GameState GameState `json:"gameState"`
}

// PiecePlacedResponse represents the response when a piece is placed
type PiecePlacedResponse struct {
	Type       string `json:"type"`   // "piecePlaced"
	UserId     string `json:"userId"` // 配置したプレイヤー
	Row        int    `json:"row"`
	Col        int    `json:"col"`
	Color      int    `json:"color"`
	NextPlayer string `json:"nextPlayer"` // 次のターンのプレイヤー
	NextColor  int    `json:"nextColor"`  // 次に配置する色
	GamePhase  string `json:"gamePhase"`  // "PLAYING" or "FINISHED"
	Winner     string `json:"winner,omitempty"`
}

// GameRoom represents room metadata
type GameRoom struct {
	RoomId      string `json:"roomId"`
	Status      string `json:"status"`
	Player1Id   string `json:"player1Id"`
	Player2Id   string `json:"player2Id"`
	PlayerCount int    `json:"playerCount"`
}

// Player represents a player in a room
type Player struct {
	UserId       string `json:"userId"`
	PlayerRole   string `json:"playerRole"`
	ConnectionId string `json:"connectionId"`
}

func handleGameMove(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	connectionId := request.RequestContext.ConnectionID
	fmt.Printf("Game move request from connection: %s\n", connectionId)

	// Parse request body
	var moveRequest MakeMoveRequest
	if err := json.Unmarshal([]byte(request.Body), &moveRequest); err != nil {
		fmt.Printf("Error parsing request body: %v\n", err)
		return events.APIGatewayProxyResponse{StatusCode: 400}, err
	}

	// Validate required fields
	if moveRequest.UserId == "" || moveRequest.RoomId == "" {
		fmt.Printf("UserId and RoomId are required\n")
		return events.APIGatewayProxyResponse{StatusCode: 400}, fmt.Errorf("userId and roomId are required")
	}

	// Initialize AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		fmt.Printf("Error creating AWS session: %v\n", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	dynamo := dynamodb.New(sess)
	apiGW := apigatewaymanagementapi.New(sess, &aws.Config{
		Endpoint: aws.String(fmt.Sprintf("https://%s.execute-api.%s.amazonaws.com/%s",
			request.RequestContext.APIID,
			os.Getenv("AWS_REGION"),
			request.RequestContext.Stage)),
	})

	tableName := os.Getenv("DYNAMODB_TABLE_NAME")
	if tableName == "" {
		tableName = "game-service"
	}

	// Get current game state
	currentGameState, err := getCurrentGameState(dynamo, tableName, moveRequest.RoomId)
	if err != nil {
		fmt.Printf("Error getting current game state: %v\n", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	// Validate turn
	if currentGameState.CurrentPlayer != moveRequest.UserId {
		fmt.Printf("Not player's turn: %s, current: %s\n", moveRequest.UserId, currentGameState.CurrentPlayer)
		return events.APIGatewayProxyResponse{StatusCode: 400}, fmt.Errorf("not your turn")
	}

	// Simple validation: check if position is valid (0-7)
	if moveRequest.Row < 0 || moveRequest.Row >= 8 || moveRequest.Col < 0 || moveRequest.Col >= 8 {
		fmt.Printf("Invalid position: row=%d, col=%d\n", moveRequest.Row, moveRequest.Col)
		return events.APIGatewayProxyResponse{StatusCode: 400}, fmt.Errorf("invalid position")
	}

	// Get room info for next player
	room, err := getRoomInfo(dynamo, tableName, moveRequest.RoomId)
	if err != nil {
		fmt.Printf("Error getting room info: %v\n", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	// Determine next player
	var nextPlayer string
	if currentGameState.CurrentPlayer == room.Player1Id {
		nextPlayer = room.Player2Id
	} else {
		nextPlayer = room.Player1Id
	}

	// Check if game is finished (simple check - after 60 moves, board is likely full)
	gamePhase := "PLAYING"
	winner := ""
	if currentGameState.TurnNumber >= 60 {
		gamePhase = "FINISHED"
		// Winner will be determined by clients when they report final board states
	}

	// Generate next color (0-255)
	nextColor := generateNextColor()

	// Create new simplified game state
	newGameState := GameState{
		RoomId:        moveRequest.RoomId,
		TurnNumber:    currentGameState.TurnNumber + 1,
		CurrentPlayer: nextPlayer,
		NextColor:     nextColor,
		GamePhase:     gamePhase,
		Winner:        winner,
	}

	// Save new game state to DynamoDB
	err = saveGameState(dynamo, tableName, newGameState)
	if err != nil {
		fmt.Printf("Error saving game state: %v\n", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	// Get players for broadcast
	players, err := getRoomPlayers(dynamo, tableName, moveRequest.RoomId)
	if err != nil {
		fmt.Printf("Error getting room players: %v\n", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	// Broadcast piece placed event to all players
	response := PiecePlacedResponse{
		Type:       "piecePlaced",
		UserId:     moveRequest.UserId,
		Row:        moveRequest.Row,
		Col:        moveRequest.Col,
		Color:      moveRequest.Color,
		NextPlayer: nextPlayer,
		NextColor:  nextColor,
		GamePhase:  gamePhase,
		Winner:     winner,
	}

	for _, player := range players {
		err = sendMessage(apiGW, player.ConnectionId, response)
		if err != nil {
			fmt.Printf("Error sending message to player %s: %v\n", player.UserId, err)
		}
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func getCurrentGameState(dynamo *dynamodb.DynamoDB, tableName, roomId string) (*GameState, error) {
	// Query for the latest game state (highest turn number)
	input := &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {S: aws.String(fmt.Sprintf("ROOM#%s", roomId))},
			":sk": {S: aws.String("TURN#")},
		},
		ScanIndexForward: aws.Bool(false), // Descending order (latest first)
		Limit:            aws.Int64(1),
	}

	result, err := dynamo.Query(input)
	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		// No turns yet, create initial state
		return createInitialGameState(dynamo, tableName, roomId)
	}

	item := result.Items[0]
	gameState := &GameState{}

	if roomIdVal, ok := item["roomId"]; ok && roomIdVal.S != nil {
		gameState.RoomId = *roomIdVal.S
	}
	if turnNumVal, ok := item["turnNumber"]; ok && turnNumVal.N != nil {
		if num, err := strconv.Atoi(*turnNumVal.N); err == nil {
			gameState.TurnNumber = num
		}
	}
	if currentPlayerVal, ok := item["currentPlayer"]; ok && currentPlayerVal.S != nil {
		gameState.CurrentPlayer = *currentPlayerVal.S
	}
	if nextColorVal, ok := item["nextColor"]; ok && nextColorVal.N != nil {
		if color, err := strconv.Atoi(*nextColorVal.N); err == nil {
			gameState.NextColor = color
		}
	}
	if gamePhaseVal, ok := item["gamePhase"]; ok && gamePhaseVal.S != nil {
		gameState.GamePhase = *gamePhaseVal.S
	}
	if winnerVal, ok := item["winner"]; ok && winnerVal.S != nil {
		gameState.Winner = *winnerVal.S
	}

	// Board state is now managed by clients - no longer stored server-side

	return gameState, nil
}

func createInitialGameState(dynamo *dynamodb.DynamoDB, tableName, roomId string) (*GameState, error) {
	// Get room info to determine first player
	room, err := getRoomInfo(dynamo, tableName, roomId)
	if err != nil {
		return nil, err
	}

	// Initial game state without board (clients handle initial board setup)
	gameState := &GameState{
		RoomId:        roomId,
		TurnNumber:    0,
		CurrentPlayer: room.Player1Id, // Player1 always goes first
		NextColor:     64,             // Start with middle gray color
		GamePhase:     "PLAYING",
	}

	// Save initial state
	err = saveGameState(dynamo, tableName, *gameState)
	if err != nil {
		return nil, err
	}

	return gameState, nil
}

func getRoomInfo(dynamo *dynamodb.DynamoDB, tableName, roomId string) (*GameRoom, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {S: aws.String(fmt.Sprintf("ROOM#%s", roomId))},
			"SK": {S: aws.String("METADATA")},
		},
	}

	result, err := dynamo.GetItem(input)
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, fmt.Errorf("room not found")
	}

	room := &GameRoom{}
	if roomIdVal, ok := result.Item["roomId"]; ok && roomIdVal.S != nil {
		room.RoomId = *roomIdVal.S
	}
	if statusVal, ok := result.Item["status"]; ok && statusVal.S != nil {
		room.Status = *statusVal.S
	}
	if player1Val, ok := result.Item["player1Id"]; ok && player1Val.S != nil {
		room.Player1Id = *player1Val.S
	}
	if player2Val, ok := result.Item["player2Id"]; ok && player2Val.S != nil {
		room.Player2Id = *player2Val.S
	}
	if playerCountVal, ok := result.Item["playerCount"]; ok && playerCountVal.N != nil {
		if count, err := strconv.Atoi(*playerCountVal.N); err == nil {
			room.PlayerCount = count
		}
	}

	return room, nil
}

func getRoomPlayers(dynamo *dynamodb.DynamoDB, tableName, roomId string) ([]Player, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {S: aws.String(fmt.Sprintf("ROOM#%s", roomId))},
			":sk": {S: aws.String("PLAYER#")},
		},
	}

	result, err := dynamo.Query(input)
	if err != nil {
		return nil, err
	}

	var players []Player
	for _, item := range result.Items {
		player := Player{}
		if userIdVal, ok := item["userId"]; ok && userIdVal.S != nil {
			player.UserId = *userIdVal.S
		}
		if roleVal, ok := item["playerRole"]; ok && roleVal.S != nil {
			player.PlayerRole = *roleVal.S
		}
		if connIdVal, ok := item["connectionId"]; ok && connIdVal.S != nil {
			player.ConnectionId = *connIdVal.S
		}
		players = append(players, player)
	}

	return players, nil
}

// Board validation and game logic moved to client-side
// Server now only handles turn management and basic position validation

func generateNextColor() int {
	// Generate a random color between 0-255
	// For simplicity, using timestamp modulo
	return int(time.Now().UnixNano() % 256)
}

func saveGameState(dynamo *dynamodb.DynamoDB, tableName string, gameState GameState) error {
	now := time.Now().Format(time.RFC3339)

	// Simplified game state without board data
	item := map[string]*dynamodb.AttributeValue{
		"PK":            {S: aws.String(fmt.Sprintf("ROOM#%s", gameState.RoomId))},
		"SK":            {S: aws.String(fmt.Sprintf("TURN#%06d", gameState.TurnNumber))},
		"roomId":        {S: aws.String(gameState.RoomId)},
		"turnNumber":    {N: aws.String(strconv.Itoa(gameState.TurnNumber))},
		"currentPlayer": {S: aws.String(gameState.CurrentPlayer)},
		"nextColor":     {N: aws.String(strconv.Itoa(gameState.NextColor))},
		"gamePhase":     {S: aws.String(gameState.GamePhase)},
		"createdAt":     {S: aws.String(now)},
	}

	if gameState.Winner != "" {
		item["winner"] = &dynamodb.AttributeValue{S: aws.String(gameState.Winner)}
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	}

	_, err := dynamo.PutItem(input)
	return err
}

func sendMessage(apiGW *apigatewaymanagementapi.ApiGatewayManagementApi, connectionId string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	input := &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: aws.String(connectionId),
		Data:         data,
	}

	_, err = apiGW.PostToConnection(input)
	return err
}

func main() {
	lambda.Start(handleGameMove)
}
