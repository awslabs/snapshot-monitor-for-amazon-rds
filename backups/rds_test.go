package backups

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/stretchr/testify/assert"
)

type mockRDSClient struct {
	describeDBSnapshotsOutput *rds.DescribeDBSnapshotsOutput
	describeDBClustersOutput  *rds.DescribeDBClusterSnapshotsOutput
	err                       error
}

func (m *mockRDSClient) DescribeDBSnapshots(ctx context.Context, params *rds.DescribeDBSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSnapshotsOutput, error) {
	return m.describeDBSnapshotsOutput, m.err
}

func (m *mockRDSClient) DescribeDBClusterSnapshots(ctx context.Context, params *rds.DescribeDBClusterSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClusterSnapshotsOutput, error) {
	return m.describeDBClustersOutput, m.err
}

func TestGetFilteredSnapshots(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)

	tests := []struct {
		name      string
		client    RDSClient
		wantCount int
		wantErr   bool
	}{
		{
			name: "successfully retrieves snapshots",
			client: &mockRDSClient{
				describeDBSnapshotsOutput: &rds.DescribeDBSnapshotsOutput{
					DBSnapshots: []types.DBSnapshot{
						{
							DBSnapshotIdentifier: aws.String("test-1"),
							SnapshotCreateTime:   &now,
						},
					},
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "handles error from AWS",
			client: &mockRDSClient{
				err: fmt.Errorf("AWS error"),
			},
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshots, err := GetFilteredSnapshots(ctx, tt.client, sevenDaysAgo)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, len(snapshots))
			}
		})
	}
}

func TestGetFilteredClusterSnapshots(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)

	tests := []struct {
		name      string
		client    *mockRDSClient
		wantCount int
		wantErr   bool
	}{
		{
			name: "successfully retrieves cluster snapshots",
			client: &mockRDSClient{
				describeDBClustersOutput: &rds.DescribeDBClusterSnapshotsOutput{
					DBClusterSnapshots: []types.DBClusterSnapshot{
						{
							DBClusterSnapshotIdentifier: aws.String("cluster-1"),
							SnapshotCreateTime:          &now,
						},
					},
				},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "handles error from AWS",
			client: &mockRDSClient{
				err: fmt.Errorf("AWS error"),
			},
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshots, err := GetFilteredClusterSnapshots(ctx, tt.client, sevenDaysAgo)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, len(snapshots))
			}
		})
	}
}
