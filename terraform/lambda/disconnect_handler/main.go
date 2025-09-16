package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func handleDisconnect(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Get connection ID from request context
	connectionId := request.RequestContext.ConnectionID

	fmt.Printf("WebSocket disconnection: %s\n", connectionId)

	// Initialize AWS session and DynamoDB client
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		fmt.Printf("Error creating AWS session: %v\n", err)
		// Return 200 even on error to avoid API Gateway retries
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	dynamo := dynamodb.New(sess)

	// Get table name from environment variable
	tableName := os.Getenv("DYNAMODB_TABLE_NAME")
	if tableName == "" {
		tableName = "websocket-connections" // fallback
	}

	// Delete connection ID from DynamoDB
	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"connectionId": {
				S: aws.String(connectionId),
			},
		},
	}

	_, err = dynamo.DeleteItem(input)
	if err != nil {
		fmt.Printf("Error deleting connection from DynamoDB: %v\n", err)
		// Return 200 even on error to avoid API Gateway retries
		// The connection is already closed, so we can't do much about it
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}

	fmt.Printf("Connection %s deleted from DynamoDB successfully\n", connectionId)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Disconnected",
	}, nil
}

func main() {
	lambda.Start(handleDisconnect)
}
