package backups

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	rdsTypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/stretchr/testify/assert"
)

type mockSnapshot struct {
	createTime *time.Time
}

func (m mockSnapshot) GetCreateTime() *time.Time {
	return m.createTime
}

func TestFilterSnapshots(t *testing.T) {
	now := time.Now()
	oldTime := now.Add(-24 * time.Hour)
	tests := []struct {
		name       string
		snapshots  []mockSnapshot
		cutoffTime time.Time
		want       int
	}{
		{
			name: "filters out old snapshots",
			snapshots: []mockSnapshot{
				{createTime: &now},
				{createTime: &oldTime},
			},
			cutoffTime: now.Add(-12 * time.Hour),
			want:       1,
		},
		{
			name: "keeps all recent snapshots",
			snapshots: []mockSnapshot{
				{createTime: &now},
				{createTime: &now},
			},
			cutoffTime: now.Add(-12 * time.Hour),
			want:       2,
		},
		{
			name: "handles nil create times",
			snapshots: []mockSnapshot{
				{createTime: &now},
				{createTime: nil},
			},
			cutoffTime: now.Add(-12 * time.Hour),
			want:       1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterSnapshots(tt.snapshots, tt.cutoffTime)
			assert.Equal(t, tt.want, len(filtered))
		})
	}
}

func TestGetFilteredSnapshotsGeneric(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	tests := []struct {
		name       string
		snapshots  []mockSnapshot
		hasError   bool
		wantCount  int
		cutoffTime time.Time
	}{
		{
			name: "successfully filters snapshots",
			snapshots: []mockSnapshot{
				{createTime: &now},
				{createTime: &now},
			},
			hasError:   false,
			wantCount:  2,
			cutoffTime: now.Add(-24 * time.Hour),
		},
		{
			name:       "handles empty snapshot list",
			snapshots:  []mockSnapshot{},
			hasError:   false,
			wantCount:  0,
			cutoffTime: now.Add(-24 * time.Hour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var page int
			nextPage := func(ctx context.Context) ([]mockSnapshot, error) {
				if page > 0 {
					return nil, nil
				}
				page++
				return tt.snapshots, nil
			}
			hasMore := func() bool {
				return page == 0
			}

			results, err := getFilteredSnapshotsGeneric(ctx, nextPage, hasMore, tt.cutoffTime)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, len(results))
			}
		})
	}
}

func TestProcessSnapshots(t *testing.T) {
	now := time.Now()
	status := "available"

	tests := []struct {
		name              string
		instanceSnapshots []DBSnapshotWrapper
		clusterSnapshots  []DBClusterSnapshotWrapper
		want              int
	}{
		{
			name: "processes both instance and cluster snapshots",
			instanceSnapshots: []DBSnapshotWrapper{
				{
					DBSnapshot: &rdsTypes.DBSnapshot{
						DBSnapshotIdentifier: aws.String("instance-1"),
						SnapshotCreateTime:   &now,
						Status:               aws.String(status),
					},
				},
			},
			clusterSnapshots: []DBClusterSnapshotWrapper{
				{
					DBClusterSnapshot: &rdsTypes.DBClusterSnapshot{
						DBClusterSnapshotIdentifier: aws.String("cluster-1"),
						SnapshotCreateTime:          &now,
						Status:                      aws.String(status),
					},
				},
			},
			want: 2,
		},
		{
			name:              "handles empty inputs",
			instanceSnapshots: []DBSnapshotWrapper{},
			clusterSnapshots:  []DBClusterSnapshotWrapper{},
			want:              0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := ProcessSnapshots(tt.instanceSnapshots, tt.clusterSnapshots)
			assert.Equal(t, tt.want, len(results))
		})
	}
}
