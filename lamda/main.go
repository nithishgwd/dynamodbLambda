package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

const TableName = "gamerDetails"
const Region = "us-east-1"

var dbClient *dynamodb.Client

type GamerRecord struct {
	GamerID          string `json:"gamerID"`
	TimeStamp        int64  `json:"timeStamp"`
	GamerName        string `json:"gamerName"`
	GamerPhoneNumber string `json:"gamerPhoneNumber"`
	Game             string `json:"game"`
}

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(Region))
	if err != nil {
		log.Fatalf("Error loading AWS SDK configuration: %v", err)
	}

	dbClient = dynamodb.NewFromConfig(cfg)
}

func HandleRequest(ctx context.Context, request Request) (Response, error) {
	switch request.HTTPMethod {
	case "POST":
		return CreateGamer(request)
	case "GET":
		return GetGamerDetails(request)
	default:
		return Response{StatusCode: 400, Body: "Invalid HTTP method"}, nil
	}
}

func CreateGamer(request Request) (Response, error) {
	// Generate a unique gamerID using a UUID
	gamerID := uuid.New().String()

	// Get the current timestamp
	timeStamp := NowUnixTimestamp()
	
	// Parse and validate the request body
	var gamerRecord GamerRecord
	err := json.Unmarshal([]byte(request.Body), &gamerRecord)
	if err != nil {
		return Response{StatusCode: 400, Body: "Invalid request body"}, nil
	}

	// Set the generated gamerID and the provided timeStamp
	gamerRecord.GamerID = gamerID
	gamerRecord.TimeStamp = timeStamp

	// Create an item in the DynamoDB table
	putItemInput := &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item: map[string]types.AttributeValue{
			"gamerID":          &types.AttributeValueMemberS{Value: gamerRecord.GamerID},
			"timeStamp":        &types.AttributeValueMemberN{Value: fmt.Sprint(gamerRecord.TimeStamp)},
			"gamerName":        &types.AttributeValueMemberS{Value: gamerRecord.GamerName},
			"gamerPhoneNumber": &types.AttributeValueMemberS{Value: gamerRecord.GamerPhoneNumber},
			"game":             &types.AttributeValueMemberS{Value: gamerRecord.Game},
		},
	}

	_, err = dbClient.PutItem(context.TODO(), putItemInput)
	if err != nil {
		return Response{StatusCode: 500, Body: "Failed to create gamer"}, nil
	}

	return Response{StatusCode: 201, Body: toJSON(gamerRecord)}, nil
}

func GetGamerDetails(request Request) (Response, error) {
	gamerID := request.PathParameters["gamerID"]

	getItemInput := &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key: map[string]types.AttributeValue{
			"gamerID": &types.AttributeValueMemberS{Value: gamerID},
		},
	}

	result, err := dbClient.GetItem(context.TODO(), getItemInput)
	if err != nil {
		return Response{StatusCode: 500, Body: "Failed to get gamer details"}, nil
	}

	if result.Item == nil {
		return Response{StatusCode: 404, Body: "Gamer not found"}, nil
	}

	var gamerRecord GamerRecord
	err = unmarshalAttributeValue(result.Item, &gamerRecord)
	if err != nil {
		return Response{StatusCode: 500, Body: "Failed to unmarshal gamer details"}, nil
	}

	return Response{StatusCode: 200, Body: toJSON(gamerRecord)}, nil
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func NowUnixTimestamp() int64 {
	return time.Now().Unix()
}

func unmarshalAttributeValue(av map[string]types.AttributeValue, v interface{}) error {
	marshalMap := make(map[string]types.AttributeValue)
	for key, val := range av {
		marshalMap[key] = val
	}

	err := attributevalue.UnmarshalMap(marshalMap, v)
	if err != nil {
		return err
	}

	return nil
}

type Request struct {
	HTTPMethod     string            `json:"httpMethod"`
	Body           string            `json:"body"`
	PathParameters map[string]string `json`
}

type Response struct {
	StatusCode int    `json:"statusCode"`
	Body       string `json:"body"`
}

func main() {
	lambda.Start(HandleRequest)
}
