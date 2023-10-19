package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"
)

const (
	PkName    = "gamerID"
	SkName    = "timeStamp"
	TableName = "gamerDetails"

	// GlobalWriteReadCap should be equal or higher than PartitionWriteReadCap
	GlobalWriteReadCap    = 2
	PartitionWriteReadCap = 2
	Location              = "us-east-1" // Update to your AWS region.
)

// loadConfig loads the AWS SDK configuration for the application.
// It sets the AWS region to the specified location.
// It returns an AWS configuration and an error, if any.
func loadConfig() (aws.Config, error) {
	return config.LoadDefaultConfig(context.TODO(), func(opts *config.LoadOptions) error {
		opts.Region = Location
		return nil
	})
}

func createTable(svc *dynamodb.Client) error {
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(TableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String(PkName),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String(SkName),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String(PkName),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String(SkName),
				KeyType:       types.KeyTypeRange,
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(PartitionWriteReadCap),
			WriteCapacityUnits: aws.Int64(GlobalWriteReadCap),
		},
	}

	_, err := svc.CreateTable(context.TODO(), input)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Programming.Errors.html
// https://github.com/aws/aws-sdk-go-v2/blob/main/service/dynamodb/types/errors.go
func HandleDynamoDBError(err error) error {
	var ae smithy.APIError
	if errors.As(err, &ae) {
		switch ae.ErrorCode() {
		case "ResourceInUseException":
			log.Println("Table already exists:", TableName)
			return nil // Handle or return a custom error if needed
		case "LimitExceededException":
			return fmt.Errorf("limit exceeded: %s", ae.ErrorMessage())
		case "InternalServerError":
			return fmt.Errorf("internal Server Error: %s", ae.ErrorMessage())
		case "ProvisionedThroughputExceededException":
			return fmt.Errorf("provisioned throughput exceeded: %s", ae.ErrorMessage())
		case "ResourceNotFoundException":
			return fmt.Errorf("resource not found: %s", ae.ErrorMessage())
		case "ConditionalCheckFailedException":
			return fmt.Errorf("conditional check failed: %s", ae.ErrorMessage())
		default:
			// Check for DNS issues
			if strings.Contains(err.Error(), "no such host") {
				return fmt.Errorf("dns resolution error. Please check your internet connection or AWS endpoint configuration")
			}
			return err // Return the raw error if it's of an unexpected type
		}
	}
	return err
}

func waitForTableCreation(svc *dynamodb.Client, tableName string) error {
	for {
		resp, err := svc.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{
			TableName: aws.String(tableName),
		})
		if err != nil {
			return err
		}
		if resp.Table.TableStatus == types.TableStatusActive {
			break
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}

// main function
func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load AWS configuration: %v", err)
	}

	svc := dynamodb.NewFromConfig(cfg)
	log.Println("Attempting to create table...")
	err = createTable(svc)
	if err != nil {
		if strings.Contains(err.Error(), "Table already exists") {
			log.Println("CreateTable:", err) // <-- Specific log for CreateTable
			os.Exit(0)                       // Exit the program gracefully with status code 0
		} else {
			log.Fatalf("Failed to create DynamoDB table: %v", err)
		}
	}

	log.Println("Waiting for table creation...")
	if err := waitForTableCreation(svc, TableName); err != nil {
		log.Println("waitForTableCreation:", err) // <-- Specific log for waitForTableCreation
		log.Fatalf("Failed to wait for table creation: %v", err)
	}

	fmt.Println("Table created successfully!")
}