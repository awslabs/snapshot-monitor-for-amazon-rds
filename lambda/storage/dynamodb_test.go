package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
)

type mockDynamoDBClient struct {
	queryOutput   *dynamodb.QueryOutput
	putItemOutput *dynamodb.PutItemOutput
	queryErr      error
	putItemErr    error
}

func (m *mockDynamoDBClient) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	return m.queryOutput, nil
}

func (m *mockDynamoDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	if m.putItemErr != nil {
		return nil, m.putItemErr
	}
	return m.putItemOutput, nil
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

func TestUpdateSnapshotState(t *testing.T) {
	ctx := context.Background()
	region := "us-west-2"
	now := time.Now()

	tests := []struct {
		name     string
		client   *mockDynamoDBClient
		snapshot SnapshotInfo
		wantErr  bool
	}{
		{
			name: "successfully updates snapshot state",
			client: &mockDynamoDBClient{
				putItemOutput: &dynamodb.PutItemOutput{},
			},
			snapshot: SnapshotInfo{
				SnapshotID:   "snap-1",
				SnapshotType: "instance",
				CreateTime:   now,
				Status:       "available",
			},
			wantErr: false,
		},
		{
			name: "handles DynamoDB error",
			client: &mockDynamoDBClient{
				putItemErr: fmt.Errorf("DynamoDB error"),
			},
			snapshot: SnapshotInfo{
				SnapshotID:   "snap-1",
				SnapshotType: "instance",
				CreateTime:   now,
				Status:       "available",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UpdateSnapshotState(ctx, tt.client, region, tt.snapshot)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
