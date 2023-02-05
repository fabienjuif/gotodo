package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-playground/validator/v10"
)

var Validate = validator.New()
var gen GenID = GenID{}
var TodoRepo = Repo{}

func GetUserID(request *events.APIGatewayProxyRequest) string {
	return request.RequestContext.Identity.CognitoIdentityID
}

type GetTodosResponse struct {
	Data *[]*Todo `json:"data"`
}

func HandlerGetTodos(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userID := GetUserID(&request)
	if len(userID) == 0 {
		fmt.Printf("User ID is not specified")
		return New403Error()
	}

	todos, err := TodoRepo.ListUserTodos(userID)
	if err != nil {
		fmt.Printf("Error while listing todos: %v\n", err)
		return New500Response("error while listing todos")
	}

	responseBody, err := json.Marshal(&GetTodosResponse{&todos})
	if err != nil {
		fmt.Println("Error while creating a response", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return New200Response(&responseBody)
}

func HandlerPostTodo(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userID := GetUserID(&request)
	if len(userID) == 0 {
		fmt.Printf("User ID is not specified")
		return New403Error()
	}

	body := new(TodosPostBody)
	err := json.Unmarshal([]byte(request.Body), &body)
	if err != nil {
		fmt.Printf("Error while parsing JSON: %v\n", err)
		return New400Response("error while parsing JSON body")
	}

	todoID := gen.Get()
	err = TodoRepo.InsertTodo(&Todo{
		ID:        todoID,
		UserID:    userID,
		Title:     *body.Title,
		Timestamp: time.Now().UnixMilli(),
	})
	if err != nil {
		fmt.Printf("Error while creating todo: %v\n", err)
		return New500Response("error while creating todo")
	}

	responseBody, err := json.Marshal(&TodosPostResponse{todoID})
	if err != nil {
		fmt.Println("Error while creating a response", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return New200Response(&responseBody)
}

func HandlerDoneTodo(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, exists := request.PathParameters["id"]
	if !exists {
		return New400Response("path parameter id must be set")
	}

	body := new(TodosPutDoneBody)
	err := json.Unmarshal([]byte(request.Body), &body)
	if err != nil {
		fmt.Printf("Error while parsing JSON: %v\n", err)
		return New400Response("error while parsing JSON body")
	}

	err = Validate.Struct(body)
	if err != nil {
		return MapValidationErrors(err.(validator.ValidationErrors))
	}

	doneStr := "undone"
	if *body.Done {
		doneStr = "done"
	}

	return events.APIGatewayProxyResponse{
		Body:       "Ho! You want to mark a todo as " + doneStr + ": " + id,
		StatusCode: 200,
	}, nil
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	fmt.Println("userId", request.RequestContext.Identity.CognitoIdentityID)

	switch request.HTTPMethod {
	case "GET":
		return HandlerGetTodos(request)
	case "POST":
		return HandlerPostTodo(request)
	case "PUT":
		switch request.RequestContext.Path {
		case "/todos/{id}/done":
			return HandlerDoneTodo(request)
		}
	}

	return New400Response(fmt.Sprintf("Unknown method or path: %s %s", request.HTTPMethod, request.RequestContext.Path))
}

func init() {
	fmt.Println("Initializing ID Generator...")
	err := gen.Init()
	if err != nil {
		fmt.Println("Error while initializing ID Generator")
		panic(err)
	}
	fmt.Println("Initializing Repository...")
	err = TodoRepo.Init()
	if err != nil {
		fmt.Println("Error while initializing Repository")
		panic(err)
	}
}

func main() {
	lambda.Start(handler)
}

type MessageResponse struct {
	Message          string     `json:"message"`
	ValidationErrors *[]*IError `json:"validationErrors"`
}

func New200Response(body *[]byte) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 200,
		Body:       string(*body),
	}, nil
}

func New400Response(message string) (events.APIGatewayProxyResponse, error) {
	body, err := json.Marshal(
		MessageResponse{message, nil},
	)

	if err != nil {
		fmt.Printf("Error while creating a response with message: %s\n", message)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 400,
		Body:       string(body),
	}, nil
}

func New500Response(message string) (events.APIGatewayProxyResponse, error) {
	body, err := json.Marshal(
		MessageResponse{message, nil},
	)

	if err != nil {
		fmt.Printf("Error while creating a response with message: %s\n", message)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 500,
		Body:       string(body),
	}, nil
}

func New403Error() (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		Headers:    map[string]string{"Content-Type": "application/json"},
		StatusCode: 403,
	}, nil
}

type TodosPutDoneBody struct {
	Done *bool `json:"done" validate:"required"`
}

type TodosPostBody struct {
	Title *string `json:"title" validate:"required,min=3,max=100"`
}

type TodosPostResponse struct {
	ID string `json:"id"`
}

func MapValidationErrors(validationErrors validator.ValidationErrors) (events.APIGatewayProxyResponse, error) {
	var errors []*IError
	for _, err := range validationErrors {
		el := IError{
			Field: err.Field(),
			Tag:   err.Tag(),
			Value: err.Param(),
		}
		errors = append(errors, &el)
	}

	message := "validation error"
	body, err := json.Marshal(
		MessageResponse{message, &errors},
	)

	if err != nil {
		fmt.Printf("Error while creating a response with message: %s\n", message)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 400,
		Body:       string(body),
	}, nil
}

type IError struct {
	Field string `json:"field"`
	Tag   string `json:"tag"`
	Value string `json:"value"`
}
