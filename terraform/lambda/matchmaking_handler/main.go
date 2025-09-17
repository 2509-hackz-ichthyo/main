package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// JoinGameRequest represents the request body for joinGame
type JoinGameRequest struct {
	Action string `json:"action"`
	UserId string `json:"userId"`
}

// MatchFoundResponse represents the response when a match is found
type MatchFoundResponse struct {
	Type     string `json:"type"`
	RoomId   string `json:"roomId"`
	Role     string `json:"role"`
	OpponentId string `json:"opponentId"`
}

// WaitingResponse represents the response when waiting for a match
type WaitingResponse struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func handleMatchmaking(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Get connection ID from request context
	connectionId := request.RequestContext.ConnectionID
	
	fmt.Printf("Matchmaking request from connection: %s\n", connectionId)

	// Parse request body
	var joinRequest JoinGameRequest
	if err := json.Unmarshal([]byte(request.Body), &joinRequest); err != nil {
		fmt.Printf("Error parsing request body: %v\n", err)
		return events.APIGatewayProxyResponse{StatusCode: 400}, err
	}

	if joinRequest.UserId == "" {
		fmt.Printf("UserId is required\n")
		return events.APIGatewayProxyResponse{StatusCode: 400}, fmt.Errorf("userId is required")
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
		tableName = "websocket-connections"
	}

	// Check for waiting players
	waitingPlayer, err := findWaitingPlayer(dynamo, tableName)
	if err != nil {
		fmt.Printf("Error finding waiting player: %v\n", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	if waitingPlayer != nil {
		// Match found - create room and notify both players
		roomId := generateRoomId()
		
		err = createGameRoom(dynamo, tableName, roomId, waitingPlayer.UserId, joinRequest.UserId, waitingPlayer.ConnectionId, connectionId)
		if err != nil {
			fmt.Printf("Error creating game room: %v\n", err)
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}

		// Notify player 1 (waiting player)
		player1Response := MatchFoundResponse{
			Type:       "matchFound",
			RoomId:     roomId,
			Role:       "PLAYER1",
			OpponentId: joinRequest.UserId,
		}
		err = sendMessage(apiGW, waitingPlayer.ConnectionId, player1Response)
		if err != nil {
			fmt.Printf("Error sending message to player 1: %v\n", err)
		}

		// Notify player 2 (current player)
		player2Response := MatchFoundResponse{
			Type:       "matchFound",
			RoomId:     roomId,
			Role:       "PLAYER2",
			OpponentId: waitingPlayer.UserId,
		}
		err = sendMessage(apiGW, connectionId, player2Response)
		if err != nil {
			fmt.Printf("Error sending message to player 2: %v\n", err)
		}

		fmt.Printf("Match created: Room %s with players %s and %s\n", roomId, waitingPlayer.UserId, joinRequest.UserId)
	} else {
		// No waiting player - add to waiting queue
		err = addToWaitingQueue(dynamo, tableName, joinRequest.UserId, connectionId)
		if err != nil {
			fmt.Printf("Error adding to waiting queue: %v\n", err)
			return events.APIGatewayProxyResponse{StatusCode: 500}, err
		}

		// Notify player they are waiting
		waitingResponse := WaitingResponse{
			Type:    "waiting",
			Message: "Waiting for opponent...",
		}
		err = sendMessage(apiGW, connectionId, waitingResponse)
		if err != nil {
			fmt.Printf("Error sending waiting message: %v\n", err)
		}

		fmt.Printf("Player %s added to waiting queue\n", joinRequest.UserId)
	}

	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

type WaitingPlayer struct {
	UserId       string
	ConnectionId string
	Timestamp    int64
}

func findWaitingPlayer(dynamo *dynamodb.DynamoDB, tableName string) (*WaitingPlayer, error) {
	// Query waiting queue (PK = "WAITING_QUEUE")
	input := &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {S: aws.String("WAITING_QUEUE")},
		},
		Limit:            aws.Int64(1),
		ScanIndexForward: aws.Bool(true), // Oldest first
	}

	result, err := dynamo.Query(input)
	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, nil // No waiting players
	}

	item := result.Items[0]
	
	// Parse the waiting player data
	player := &WaitingPlayer{}
	if userId, ok := item["userId"]; ok && userId.S != nil {
		player.UserId = *userId.S
	}
	if connectionId, ok := item["connectionId"]; ok && connectionId.S != nil {
		player.ConnectionId = *connectionId.S
	}
	if timestamp, ok := item["timestamp"]; ok && timestamp.N != nil {
		// Parse timestamp if needed
	}

	// Remove from waiting queue
	deleteInput := &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"PK": {S: aws.String("WAITING_QUEUE")},
			"SK": item["SK"],
		},
	}
	_, err = dynamo.DeleteItem(deleteInput)
	if err != nil {
		fmt.Printf("Error removing from waiting queue: %v\n", err)
	}

	return player, nil
}

func addToWaitingQueue(dynamo *dynamodb.DynamoDB, tableName, userId, connectionId string) error {
	now := time.Now()
	timestamp := now.Unix()
	
	// Use timestamp#userId as sort key for ordering
	sortKey := fmt.Sprintf("%d#%s", timestamp, userId)

	input := &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]*dynamodb.AttributeValue{
			"PK":           {S: aws.String("WAITING_QUEUE")},
			"SK":           {S: aws.String(sortKey)},
			"userId":       {S: aws.String(userId)},
			"connectionId": {S: aws.String(connectionId)},
			"timestamp":    {N: aws.String(fmt.Sprintf("%d", timestamp))},
			"waitingSince": {S: aws.String(now.Format(time.RFC3339))},
		},
	}

	_, err := dynamo.PutItem(input)
	return err
}

func createGameRoom(dynamo *dynamodb.DynamoDB, tableName, roomId, player1Id, player2Id, player1ConnId, player2ConnId string) error {
	now := time.Now().Format(time.RFC3339)

	// Create room metadata
	roomInput := &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]*dynamodb.AttributeValue{
			"PK":           {S: aws.String(fmt.Sprintf("ROOM#%s", roomId))},
			"SK":           {S: aws.String("METADATA")},
			"roomId":       {S: aws.String(roomId)},
			"status":       {S: aws.String("WAITING")},
			"playerCount":  {N: aws.String("2")},
			"player1Id":    {S: aws.String(player1Id)},
			"player2Id":    {S: aws.String(player2Id)},
			"createdAt":    {S: aws.String(now)},
			"updatedAt":    {S: aws.String(now)},
		},
	}

	_, err := dynamo.PutItem(roomInput)
	if err != nil {
		return err
	}

	// Add player 1
	player1Input := &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]*dynamodb.AttributeValue{
			"PK":           {S: aws.String(fmt.Sprintf("ROOM#%s", roomId))},
			"SK":           {S: aws.String(fmt.Sprintf("PLAYER#%s", player1Id))},
			"userId":       {S: aws.String(player1Id)},
			"roomId":       {S: aws.String(roomId)},
			"playerRole":   {S: aws.String("PLAYER1")},
			"connectionId": {S: aws.String(player1ConnId)},
			"joinedAt":     {S: aws.String(now)},
		},
	}

	_, err = dynamo.PutItem(player1Input)
	if err != nil {
		return err
	}

	// Add player 2
	player2Input := &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]*dynamodb.AttributeValue{
			"PK":           {S: aws.String(fmt.Sprintf("ROOM#%s", roomId))},
			"SK":           {S: aws.String(fmt.Sprintf("PLAYER#%s", player2Id))},
			"userId":       {S: aws.String(player2Id)},
			"roomId":       {S: aws.String(roomId)},
			"playerRole":   {S: aws.String("PLAYER2")},
			"connectionId": {S: aws.String(player2ConnId)},
			"joinedAt":     {S: aws.String(now)},
		},
	}

	_, err = dynamo.PutItem(player2Input)
	return err
}

func generateRoomId() string {
	return fmt.Sprintf("room_%d", time.Now().UnixNano())
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
	lambda.Start(handleMatchmaking)
}