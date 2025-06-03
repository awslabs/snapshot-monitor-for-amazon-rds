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
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error)
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

// BatchUpdateSnapshotStates updates multiple snapshot states at once using BatchWriteItem
func BatchUpdateSnapshotStates(ctx context.Context, ddbClient DDBClient, region string, snapshots []SnapshotInfo, snapshotAgeDays int) error {
	if len(snapshots) == 0 {
		return nil
	}

	// DynamoDB BatchWriteItem can process up to 25 items at once
	const batchSize = 25
	expirationTime := time.Now().Add(time.Duration(snapshotAgeDays) * 24 * time.Hour)
	tableName := os.Getenv("DYNAMODB_TABLE_NAME")

	// Process snapshots in batches of 25
	for i := 0; i < len(snapshots); i += batchSize {
		end := i + batchSize
		if end > len(snapshots) {
			end = len(snapshots)
		}

		batch := snapshots[i:end]
		writeRequests := make([]ddbTypes.WriteRequest, len(batch))

		for j, snapshot := range batch {
			writeRequests[j] = ddbTypes.WriteRequest{
				PutRequest: &ddbTypes.PutRequest{
					Item: map[string]ddbTypes.AttributeValue{
						"pk":     &ddbTypes.AttributeValueMemberS{Value: region},
						"sk":     &ddbTypes.AttributeValueMemberS{Value: snapshot.SnapshotID},
						"status": &ddbTypes.AttributeValueMemberS{Value: snapshot.Status},
						"ttl":    &ddbTypes.AttributeValueMemberN{Value: fmt.Sprintf("%d", expirationTime.Unix())},
					},
				},
			}
		}

		_, err := ddbClient.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]ddbTypes.WriteRequest{
				tableName: writeRequests,
			},
		})

		if err != nil {
			return fmt.Errorf("unable to batch update snapshot states in DynamoDB for region %s: %v", region, err)
		}
	}

	return nil
}
