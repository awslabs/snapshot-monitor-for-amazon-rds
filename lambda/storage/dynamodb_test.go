package storage

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
)

type mockDynamoDBClient struct {
	queryOutput        *dynamodb.QueryOutput
	batchWriteOutput   *dynamodb.BatchWriteItemOutput
	queryErr           error
	batchWriteItemErr  error
	capturedBatchWrite *dynamodb.BatchWriteItemInput
}

func (m *mockDynamoDBClient) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	return m.queryOutput, nil
}

func (m *mockDynamoDBClient) BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
	m.capturedBatchWrite = params
	if m.batchWriteItemErr != nil {
		return nil, m.batchWriteItemErr
	}
	return m.batchWriteOutput, nil
}

func TestGetProcessedSnapshots(t *testing.T) {
	ctx := context.Background()
	region := "us-west-2"

	tests := []struct {
		name           string
		client         DDBClient
		expectedResult map[string]string
		wantErr        bool
	}{
		{
			name: "successfully retrieves processed snapshots",
			client: &mockDynamoDBClient{
				queryOutput: &dynamodb.QueryOutput{
					Items: []map[string]types.AttributeValue{
						{
							"sk": &types.AttributeValueMemberS{
								Value: "snap-1",
							},
							"status": &types.AttributeValueMemberS{
								Value: "available",
							},
						},
						{
							"sk": &types.AttributeValueMemberS{
								Value: "snap-2",
							},
							"status": &types.AttributeValueMemberS{
								Value: "creating",
							},
						},
					},
				},
			},
			expectedResult: map[string]string{
				"snap-1": "available",
				"snap-2": "creating",
			},
			wantErr: false,
		},
		{
			name: "handles empty result",
			client: &mockDynamoDBClient{
				queryOutput: &dynamodb.QueryOutput{
					Items: []map[string]types.AttributeValue{},
				},
			},
			expectedResult: map[string]string{},
			wantErr:        false,
		},
		{
			name: "handles DynamoDB error",
			client: &mockDynamoDBClient{
				queryErr: fmt.Errorf("DynamoDB error"),
			},
			expectedResult: nil,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetProcessedSnapshots(ctx, tt.client, region)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestBatchUpdateSnapshotStates(t *testing.T) {
	ctx := context.Background()
	region := "us-west-2"
	now := time.Now()

	// Set environment variables for tests
	t.Setenv("DYNAMODB_TABLE_NAME", "test-table")
	t.Setenv("SCHEDULE_EXPRESSION", "rate(10 minutes)")
	t.Setenv("SNAPSHOT_AGE_DAYS", "7")

	tests := []struct {
		name      string
		client    *mockDynamoDBClient
		snapshots []SnapshotInfo
		wantErr   bool
	}{
		{
			name: "successfully updates multiple snapshots",
			client: &mockDynamoDBClient{
				batchWriteOutput: &dynamodb.BatchWriteItemOutput{},
			},
			snapshots: []SnapshotInfo{
				{
					SnapshotID:   "snap-1",
					SnapshotType: "instance",
					CreateTime:   now,
					Status:       "available",
				},
				{
					SnapshotID:   "snap-2",
					SnapshotType: "cluster",
					CreateTime:   now,
					Status:       "creating",
				},
			},
			wantErr: false,
		},
		{
			name: "handles empty snapshots list",
			client: &mockDynamoDBClient{
				batchWriteOutput: &dynamodb.BatchWriteItemOutput{},
			},
			snapshots: []SnapshotInfo{},
			wantErr:   false,
		},
		{
			name: "handles DynamoDB error",
			client: &mockDynamoDBClient{
				batchWriteItemErr: fmt.Errorf("DynamoDB batch write error"),
			},
			snapshots: []SnapshotInfo{
				{
					SnapshotID:   "snap-1",
					SnapshotType: "instance",
					CreateTime:   now,
					Status:       "available",
				},
			},
			wantErr: true,
		},
		{
			name: "handles large batch (>25 items)",
			client: &mockDynamoDBClient{
				batchWriteOutput: &dynamodb.BatchWriteItemOutput{},
			},
			snapshots: func() []SnapshotInfo {
				snapshots := make([]SnapshotInfo, 30)
				for i := 0; i < 30; i++ {
					snapshots[i] = SnapshotInfo{
						SnapshotID:   fmt.Sprintf("snap-%d", i),
						SnapshotType: "instance",
						CreateTime:   now,
						Status:       "available",
					}
				}
				return snapshots
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset captured batch write for each test
			tt.client.capturedBatchWrite = nil
			snapshotAgeDays, _ := strconv.Atoi(os.Getenv("SNAPSHOT_AGE_DAYS"))

			err := BatchUpdateSnapshotStates(ctx, tt.client, region, tt.snapshots, snapshotAgeDays)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// For non-empty snapshots, verify the batch write was called correctly
				if len(tt.snapshots) > 0 {
					assert.NotNil(t, tt.client.capturedBatchWrite)

					// For large batches, verify it was split correctly
					if len(tt.snapshots) > 25 {
						// Should have been called at least once
						assert.NotEmpty(t, tt.client.capturedBatchWrite.RequestItems)

						// Check that the table name is correct
						tableName := "test-table"
						_, exists := tt.client.capturedBatchWrite.RequestItems[tableName]
						assert.True(t, exists, "Table name should be %s", tableName)
					} else {
						tableName := "test-table"
						requests, exists := tt.client.capturedBatchWrite.RequestItems[tableName]
						assert.True(t, exists, "Table name should be %s", tableName)
						assert.Len(t, requests, len(tt.snapshots))
					}
				}
			}
		})
	}
}
