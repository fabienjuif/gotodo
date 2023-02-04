package main

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type MyEvent struct {
	Name string `json:"name"`
}

// func HandleRequest(ctx context.Context, name MyEvent) (string, error) {
func HandleRequest() (string, error) {
	fmt.Println("Handling...")
	return fmt.Sprintf("Hello %s!", "you"), nil
}

func main() {
	svc := CreateDynamodbClient()

	// Create tables
	CreateMovie(svc)

	// Start lambda
	lambda.Start(HandleRequest)
}

func CreateDynamodbClient() *dynamodb.DynamoDB {
	dynamodbEndpoint, dynamodbEndpointExists := os.LookupEnv("AWS_DYNAMODB_ENDPOINT")
	awsConfig := aws.NewConfig()
	if dynamodbEndpointExists && len(dynamodbEndpoint) > 0 {
		awsConfig = awsConfig.WithEndpoint(dynamodbEndpoint)
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            *awsConfig,
	}))

	return dynamodb.New(sess)
}

func CreateMovie(svc *dynamodb.DynamoDB) {
	// Create table Movies
	tableName := "Movies"

	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("Year"),
				AttributeType: aws.String("N"),
			},
			{
				AttributeName: aws.String("Title"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("Year"),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("Title"),
				KeyType:       aws.String("RANGE"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
		TableName: aws.String(tableName),
	}

	_, err := svc.CreateTable(input)
	if err != nil {
		log.Fatalf("Got error calling CreateTable: %s", err)
	}

	fmt.Println("Created the table", tableName)
}
