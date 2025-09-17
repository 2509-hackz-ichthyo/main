package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// CleanupResult represents the result of cleanup operations
type CleanupResult struct {
	InactiveRoomsDeleted     int `json:"inactiveRoomsDeleted"`
	WaitingEntriesDeleted    int `json:"waitingEntriesDeleted"`
	DisconnectedUsersDeleted int `json:"disconnectedUsersDeleted"`
	OldConnectionsDeleted    int `json:"oldConnectionsDeleted"`
}

// DeleteRequest represents a delete operation
type DeleteRequest struct {
	TableName string
	PK        string
	SK        string
}

func handleCleanup(ctx context.Context, event events.CloudWatchEvent) (CleanupResult, error) {
	fmt.Printf("Starting cleanup process at %s\n", time.Now().Format(time.RFC3339))

	// Initialize AWS session and DynamoDB client
	sess, err := session.NewSession()
	if err != nil {
		return CleanupResult{}, fmt.Errorf("error creating AWS session: %v", err)
	}

	dynamo := dynamodb.New(sess)
	result := CleanupResult{}

	// Get table names from environment variables
	gameServiceTable := os.Getenv("GAME_SERVICE_TABLE_NAME")
	if gameServiceTable == "" {
		gameServiceTable = "game-service"
	}

	websocketTable := os.Getenv("WEBSOCKET_TABLE_NAME")
	if websocketTable == "" {
		websocketTable = "websocket-connections"
	}

	// Calculate cutoff times
	now := time.Now()
	roomCutoff := now.Add(-24 * time.Hour).Format(time.RFC3339) // 24 hours ago
	waitingCutoff := now.Add(-30 * time.Minute).Unix()          // 30 minutes ago
	userCutoff := now.Add(-1 * time.Hour).Format(time.RFC3339)  // 1 hour ago

	fmt.Printf("Cutoff times - Rooms: %s, Waiting: %d, Users: %s\n", roomCutoff, waitingCutoff, userCutoff)

	// 1. Clean up inactive rooms (highest priority for storage cost)
	inactiveCount, err := cleanupInactiveRooms(dynamo, gameServiceTable, roomCutoff)
	if err != nil {
		fmt.Printf("Error cleaning up inactive rooms: %v\n", err)
	} else {
		result.InactiveRoomsDeleted = inactiveCount
		fmt.Printf("Deleted %d inactive rooms\n", inactiveCount)
	}

	// 2. Clean up old waiting entries (highest priority for functionality)
	waitingCount, err := cleanupWaitingQueue(dynamo, gameServiceTable, waitingCutoff)
	if err != nil {
		fmt.Printf("Error cleaning up waiting queue: %v\n", err)
	} else {
		result.WaitingEntriesDeleted = waitingCount
		fmt.Printf("Deleted %d waiting entries\n", waitingCount)
	}

	// 3. Clean up disconnected users (medium priority)
	userCount, err := cleanupDisconnectedUsers(dynamo, gameServiceTable, userCutoff)
	if err != nil {
		fmt.Printf("Error cleaning up disconnected users: %v\n", err)
	} else {
		result.DisconnectedUsersDeleted = userCount
		fmt.Printf("Deleted %d disconnected users\n", userCount)
	}

	// 4. Clean up old websocket connections (low priority)
	connectionCount, err := cleanupOldConnections(dynamo, websocketTable, userCutoff)
	if err != nil {
		fmt.Printf("Error cleaning up old connections: %v\n", err)
	} else {
		result.OldConnectionsDeleted = connectionCount
		fmt.Printf("Deleted %d old connections\n", connectionCount)
	}

	fmt.Printf("Cleanup completed: %+v\n", result)
	return result, nil
}

// cleanupInactiveRooms removes finished rooms older than 24 hours
func cleanupInactiveRooms(dynamo *dynamodb.DynamoDB, tableName, cutoffTime string) (int, error) {
	// Find inactive rooms using GSI1
	input := &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :status AND GSI1SK <= :cutoff"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":status": {S: aws.String("ROOM_STATUS#FINISHED")},
			":cutoff": {S: aws.String(cutoffTime)},
		},
	}

	result, err := dynamo.Query(input)
	if err != nil {
		return 0, fmt.Errorf("failed to query inactive rooms: %v", err)
	}

	roomIds := []string{}
	for _, item := range result.Items {
		if pk := item["PK"]; pk != nil && pk.S != nil {
			// Extract roomId from PK (format: ROOM#roomId)
			roomId := *pk.S
			if len(roomId) > 5 && roomId[:5] == "ROOM#" {
				roomIds = append(roomIds, roomId[5:])
			}
		}
	}

	deletedCount := 0
	for _, roomId := range roomIds {
		count, err := deleteRoomData(dynamo, tableName, roomId)
		if err != nil {
			fmt.Printf("Error deleting room %s: %v\n", roomId, err)
			continue
		}
		deletedCount += count
	}

	return deletedCount, nil
}

// deleteRoomData removes all data related to a specific room
func deleteRoomData(dynamo *dynamodb.DynamoDB, tableName, roomId string) (int, error) {
	// Query all items for this room
	input := &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {S: aws.String("ROOM#" + roomId)},
		},
	}

	result, err := dynamo.Query(input)
	if err != nil {
		return 0, err
	}

	// Prepare delete requests
	deleteRequests := []DeleteRequest{}
	for _, item := range result.Items {
		if pk := item["PK"]; pk != nil && pk.S != nil {
			if sk := item["SK"]; sk != nil && sk.S != nil {
				deleteRequests = append(deleteRequests, DeleteRequest{
					TableName: tableName,
					PK:        *pk.S,
					SK:        *sk.S,
				})
			}
		}
	}

	// Execute batch delete
	err = batchDeleteItems(dynamo, deleteRequests)
	if err != nil {
		return 0, err
	}

	return len(deleteRequests), nil
}

// cleanupWaitingQueue removes old waiting entries
func cleanupWaitingQueue(dynamo *dynamodb.DynamoDB, tableName string, cutoffTime int64) (int, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		KeyConditionExpression: aws.String("PK = :pk AND SK <= :cutoff"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk":     {S: aws.String("WAITING_QUEUE")},
			":cutoff": {S: aws.String(strconv.FormatInt(cutoffTime, 10) + "#")},
		},
	}

	result, err := dynamo.Query(input)
	if err != nil {
		return 0, err
	}

	// Prepare delete requests
	deleteRequests := []DeleteRequest{}
	for _, item := range result.Items {
		if pk := item["PK"]; pk != nil && pk.S != nil {
			if sk := item["SK"]; sk != nil && sk.S != nil {
				deleteRequests = append(deleteRequests, DeleteRequest{
					TableName: tableName,
					PK:        *pk.S,
					SK:        *sk.S,
				})
			}
		}
	}

	// Execute batch delete
	err = batchDeleteItems(dynamo, deleteRequests)
	if err != nil {
		return 0, err
	}

	return len(deleteRequests), nil
}

// cleanupDisconnectedUsers removes users disconnected for more than 1 hour
func cleanupDisconnectedUsers(dynamo *dynamodb.DynamoDB, tableName, cutoffTime string) (int, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :status"),
		FilterExpression:       aws.String("lastActiveAt <= :cutoff"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":status": {S: aws.String("USER_STATUS#DISCONNECTED")},
			":cutoff": {S: aws.String(cutoffTime)},
		},
	}

	result, err := dynamo.Query(input)
	if err != nil {
		return 0, err
	}

	// Prepare delete requests
	deleteRequests := []DeleteRequest{}
	for _, item := range result.Items {
		if pk := item["PK"]; pk != nil && pk.S != nil {
			if sk := item["SK"]; sk != nil && sk.S != nil {
				deleteRequests = append(deleteRequests, DeleteRequest{
					TableName: tableName,
					PK:        *pk.S,
					SK:        *sk.S,
				})
			}
		}
	}

	// Execute batch delete
	err = batchDeleteItems(dynamo, deleteRequests)
	if err != nil {
		return 0, err
	}

	return len(deleteRequests), nil
}

// cleanupOldConnections removes old websocket connections
func cleanupOldConnections(dynamo *dynamodb.DynamoDB, tableName, cutoffTime string) (int, error) {
	input := &dynamodb.ScanInput{
		TableName:        aws.String(tableName),
		FilterExpression: aws.String("connectedAt <= :cutoff"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":cutoff": {S: aws.String(cutoffTime)},
		},
	}

	result, err := dynamo.Scan(input)
	if err != nil {
		return 0, err
	}

	// Prepare delete requests for websocket-connections table (uses connectionId as key)
	deleteCount := 0
	for _, item := range result.Items {
		if connectionId := item["connectionId"]; connectionId != nil && connectionId.S != nil {
			deleteInput := &dynamodb.DeleteItemInput{
				TableName: aws.String(tableName),
				Key: map[string]*dynamodb.AttributeValue{
					"connectionId": connectionId,
				},
			}

			_, err := dynamo.DeleteItem(deleteInput)
			if err != nil {
				fmt.Printf("Error deleting connection %s: %v\n", *connectionId.S, err)
				continue
			}
			deleteCount++
		}
	}

	return deleteCount, nil
}

// batchDeleteItems performs batch delete operations with DynamoDB limits
func batchDeleteItems(dynamo *dynamodb.DynamoDB, requests []DeleteRequest) error {
	if len(requests) == 0 {
		return nil
	}

	// Process in batches of 25 (DynamoDB limit)
	for i := 0; i < len(requests); i += 25 {
		end := i + 25
		if end > len(requests) {
			end = len(requests)
		}

		batch := requests[i:end]
		input := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{},
		}

		// Group by table name
		for _, req := range batch {
			if input.RequestItems[req.TableName] == nil {
				input.RequestItems[req.TableName] = []*dynamodb.WriteRequest{}
			}

			input.RequestItems[req.TableName] = append(
				input.RequestItems[req.TableName],
				&dynamodb.WriteRequest{
					DeleteRequest: &dynamodb.DeleteRequest{
						Key: map[string]*dynamodb.AttributeValue{
							"PK": {S: aws.String(req.PK)},
							"SK": {S: aws.String(req.SK)},
						},
					},
				},
			)
		}

		_, err := dynamo.BatchWriteItem(input)
		if err != nil {
			return fmt.Errorf("batch delete failed: %v", err)
		}
	}

	return nil
}

func main() {
	lambda.Start(handleCleanup)
}
