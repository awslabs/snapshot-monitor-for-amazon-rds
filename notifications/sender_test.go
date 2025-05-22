package notifications

import (
	"context"
	"fmt"
	"rds-backup-monitor/lambda/storage"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/stretchr/testify/assert"
)

type mockSNSClient struct {
	publishOutput *sns.PublishOutput
	err           error
}

func (m *mockSNSClient) Publish(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error) {
	return m.publishOutput, m.err
}

type mockDynamoDBClient struct {
	err error
}

func (m *mockDynamoDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return &dynamodb.PutItemOutput{}, m.err
}

func (m *mockDynamoDBClient) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return &dynamodb.QueryOutput{}, m.err
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		str      string
		expected bool
	}{
		{
			name:     "string exists in slice",
			slice:    []string{"a", "b", "c"},
			str:      "b",
			expected: true,
		},
		{
			name:     "string does not exist in slice",
			slice:    []string{"a", "b", "c"},
			str:      "d",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			str:      "a",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.str)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProcessSnapshotChanges(t *testing.T) {
	ctx := context.Background()
	region := "us-west-2"
	statusesToMonitor := []string{"available", "error"}

	tests := []struct {
		name               string
		filteredSnapshots  []storage.SnapshotInfo
		processedSnapshots map[string]string
		snsErr             error
		ddbErr             error
		wantErr            bool
	}{
		{
			name: "successfully processes new snapshots",
			filteredSnapshots: []storage.SnapshotInfo{
				{
					SnapshotID: "snap-1",
					Status:     "available",
				},
			},
			processedSnapshots: map[string]string{},
			snsErr:             nil,
			ddbErr:             nil,
			wantErr:            false,
		},
		{
			name: "successfully processes status changes",
			filteredSnapshots: []storage.SnapshotInfo{
				{
					SnapshotID: "snap-1",
					Status:     "error",
				},
			},
			processedSnapshots: map[string]string{
				"snap-1": "available",
			},
			snsErr:  nil,
			ddbErr:  nil,
			wantErr: false,
		},
		{
			name: "handles SNS error",
			filteredSnapshots: []storage.SnapshotInfo{
				{
					SnapshotID: "snap-1",
					Status:     "available",
				},
			},
			processedSnapshots: map[string]string{},
			snsErr:             fmt.Errorf("SNS error"),
			ddbErr:             nil,
			wantErr:            true,
		},
		{
			name: "handles DynamoDB error",
			filteredSnapshots: []storage.SnapshotInfo{
				{
					SnapshotID: "snap-1",
					Status:     "available",
				},
			},
			processedSnapshots: map[string]string{},
			snsErr:             nil,
			ddbErr:             fmt.Errorf("DynamoDB error"),
			wantErr:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snsClient := &mockSNSClient{
				publishOutput: &sns.PublishOutput{},
				err:           tt.snsErr,
			}
			ddbClient := &mockDynamoDBClient{
				err: tt.ddbErr,
			}

			err := ProcessSnapshotChanges(ctx, tt.filteredSnapshots, tt.processedSnapshots,
				statusesToMonitor, region, snsClient, ddbClient)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
