package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func handleConnect(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Get connection ID from request context
	connectionId := request.RequestContext.ConnectionID

	fmt.Printf("New WebSocket connection: %s\n", connectionId)

	// Initialize AWS session and DynamoDB client
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		fmt.Printf("Error creating AWS session: %v\n", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	dynamo := dynamodb.New(sess)

	// Get table name from environment variable
	tableName := os.Getenv("DYNAMODB_TABLE_NAME")
	if tableName == "" {
		tableName = "websocket-connections" // fallback
	}

	// Save connection ID to DynamoDB
	now := time.Now().Format(time.RFC3339)

	input := &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]*dynamodb.AttributeValue{
			"connectionId": {
				S: aws.String(connectionId),
			},
			"connectedAt": {
				S: aws.String(now),
			},
			"lastActiveAt": {
				S: aws.String(now),
			},
		},
	}

	_, err = dynamo.PutItem(input)
	if err != nil {
		fmt.Printf("Error saving connection to DynamoDB: %v\n", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	fmt.Printf("Connection %s saved to DynamoDB successfully\n", connectionId)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Connected",
	}, nil
}

func main() {
	lambda.Start(handleConnect)
}
