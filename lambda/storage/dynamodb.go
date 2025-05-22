package storage

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DDBClient interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

func GetProcessedSnapshots(ctx context.Context, ddbClient DDBClient, region string) (map[string]string, error) {
	processedSnapshots := make(map[string]string)
	var lastEvaluatedKey map[string]ddbTypes.AttributeValue

	for {
		input := &dynamodb.QueryInput{
			TableName:              aws.String(os.Getenv("DYNAMODB_TABLE_NAME")),
			KeyConditionExpression: aws.String("pk = :region"),
			ExpressionAttributeValues: map[string]ddbTypes.AttributeValue{
				":region": &ddbTypes.AttributeValueMemberS{Value: region},
			},
		}

		if lastEvaluatedKey != nil {
			input.ExclusiveStartKey = lastEvaluatedKey
		}

		result, err := ddbClient.Query(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("unable to query snapshots from DynamoDB in region %s: %v", region, err)
		}

		for _, item := range result.Items {
			snapshotID := item["sk"].(*ddbTypes.AttributeValueMemberS).Value
			status := item["status"].(*ddbTypes.AttributeValueMemberS).Value
			processedSnapshots[snapshotID] = status
		}

		lastEvaluatedKey = result.LastEvaluatedKey
		if lastEvaluatedKey == nil {
			break // No more items to fetch
		}
	}

	return processedSnapshots, nil
}

func UpdateSnapshotState(ctx context.Context, ddbClient DDBClient, region string, snapshot SnapshotInfo) error {
	expirationTime := time.Now().Add(7 * 24 * time.Hour)
	_, err := ddbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("DYNAMODB_TABLE_NAME")),
		Item: map[string]ddbTypes.AttributeValue{
			"pk":     &ddbTypes.AttributeValueMemberS{Value: region},
			"sk":     &ddbTypes.AttributeValueMemberS{Value: snapshot.SnapshotID},
			"status": &ddbTypes.AttributeValueMemberS{Value: snapshot.Status},
			"ttl":    &ddbTypes.AttributeValueMemberN{Value: fmt.Sprintf("%d", expirationTime.Unix())},
		},
	})
	if err != nil {
		return fmt.Errorf("unable to update snapshot state in DynamoDB in region %s: %v", region, err)
	}
	return nil
}
