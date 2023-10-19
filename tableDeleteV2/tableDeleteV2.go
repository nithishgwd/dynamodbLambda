// delete_table.go

package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/aws/smithy-go"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	
)

const (
	TableName = "gamerDetails"
	Location  = "us-east-1"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load AWS configuration: %v", err)
	}
	svc := dynamodb.NewFromConfig(cfg)

	if err := deleteTable(svc); err != nil {
		log.Fatalf("Failed to delete DynamoDB table: %v", err)
	}
	fmt.Println("Table deleted successfully!")
}

func loadConfig() (aws.Config, error) {
	return config.LoadDefaultConfig(context.TODO(), func(opts *config.LoadOptions) error {
		opts.Region = Location
		return nil
	})
}

func deleteTable(svc *dynamodb.Client) error {
	input := &dynamodb.DeleteTableInput{
		TableName: aws.String(TableName),
	}

	_, err := svc.DeleteTable(context.TODO(), input)
	return handleDeleteError(err)
}

// The error handling function remains the same as before
// https://aws.github.io/aws-sdk-go-v2/docs/migrating/
// https://aws.github.io/aws-sdk-go-v2/docs/handling-errors/
func handleDeleteError(err error) error {
	// All service API response errors implement the smithy.APIError interface type
	var ae smithy.APIError
	if errors.As(err, &ae) {
		switch ae.ErrorCode() {
		case "TableInUseException":
			return fmt.Errorf("table is in use: %s", ae.ErrorMessage())
		case "TableNotFoundException", "ResourceNotFoundException":
			return fmt.Errorf("table not found: %s", ae.ErrorMessage())
		default:
			return err
		}
	}
	return err
}
