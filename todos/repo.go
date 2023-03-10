package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var todoKey = "T"
var userKey = "U"
var keySep = "#"

type Todo struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Title     string `json:"title"`
	Timestamp int64  `json:"timestamp"`
}

func (t *Todo) String() string {
	str, _ := json.Marshal(t)
	return string(str)
}

type Repo struct {
	tableName string
	svc       *dynamodb.DynamoDB
}

func ToAWSKey(parts []string) string {
	return strings.Join(parts, keySep)
}

func (r *Repo) Init() error {
	tableName, tableNameExists := os.LookupEnv("TABLE_NAME")
	if !tableNameExists {
		return fmt.Errorf("env variable not set: TABLE_NAME")
	}
	r.tableName = tableName
	fmt.Printf("Initializing Repo with tableName=%s\n", r.tableName)

	dynamodbEndpoint, dynamodbEndpointExists := os.LookupEnv("AWS_DYNAMODB_ENDPOINT")
	awsConfig := aws.NewConfig()
	if dynamodbEndpointExists && len(dynamodbEndpoint) > 0 {
		fmt.Printf("Using custom DynamoDB address: %s\n", dynamodbEndpoint)
		awsConfig = awsConfig.WithEndpoint(dynamodbEndpoint)
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            *awsConfig,
	}))
	r.svc = dynamodb.New(sess)
	return r.CreateTable()
}

func (r *Repo) InsertTodo(todo *Todo) error {
	_, err := r.svc.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item: map[string]*dynamodb.AttributeValue{
			"PK": {
				S: aws.String(ToAWSKey([]string{userKey, todo.UserID})),
			},
			"SK": {
				S: aws.String(ToAWSKey([]string{todoKey, strconv.FormatInt(todo.Timestamp, 10), todo.ID})),
			},
			"title": {
				S: aws.String(todo.Title),
			},
		},
	})

	return err
}

func (r *Repo) ListUserTodos(userId string) ([]*Todo, error) {
	fmt.Println("Key", ToAWSKey([]string{userKey, userId}))
	output, err := r.svc.Query(&dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":key": {
				S: aws.String(ToAWSKey([]string{userKey, userId})),
			},
		},
		KeyConditionExpression: aws.String("PK = :key"),
		ProjectionExpression:   aws.String("SK,title"),
		TableName:              aws.String(r.tableName),
	})
	if err != nil {
		return nil, err
	}

	todos := make([]*Todo, 0, 100)
	len := 0
	for _, item := range output.Items {
		len += 1
		splits := strings.Split(*item["SK"].S, keySep)
		timestamp, err := strconv.ParseInt(splits[1], 10, 64)
		if err != nil {
			return nil, err
		}
		todos = append(todos, &Todo{
			ID:        splits[2],
			UserID:    userId,
			Title:     *item["title"].S,
			Timestamp: timestamp,
		})
	}

	return todos[:len], nil
}

func (r *Repo) CreateTable() error {
	tableExists := false
	err := r.svc.ListTablesPages(&dynamodb.ListTablesInput{}, func(lto *dynamodb.ListTablesOutput, b bool) bool {
		for _, curTableName := range lto.TableNames {
			if *curTableName == r.tableName {
				tableExists = true
				return false
			}
		}

		return true
	})
	if err != nil {
		return err
	}

	if tableExists {
		fmt.Println("Table already exists", r.tableName)
		return nil
	}

	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("PK"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("SK"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("PK"),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("SK"),
				KeyType:       aws.String("RANGE"),
			},
		},
		BillingMode: aws.String("PAY_PER_REQUEST"),
		TableName:   aws.String(r.tableName),
	}

	_, err = r.svc.CreateTable(input)
	if err != nil {
		fmt.Println("Created the table", r.tableName)
	}
	return err
}
