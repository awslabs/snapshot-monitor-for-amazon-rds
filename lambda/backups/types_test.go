package backups

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/stretchr/testify/assert"
)

func TestDBSnapshotWrapper_GetCreateTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		wrapper DBSnapshotWrapper
		want    *time.Time
	}{
		{
			name: "returns create time when set",
			wrapper: DBSnapshotWrapper{
				DBSnapshot: &types.DBSnapshot{
					SnapshotCreateTime: &now,
				},
			},
			want: &now,
		},
		{
			name: "returns nil when create time not set",
			wrapper: DBSnapshotWrapper{
				DBSnapshot: &types.DBSnapshot{},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.wrapper.GetCreateTime()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDBClusterSnapshotWrapper_GetCreateTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		wrapper DBClusterSnapshotWrapper
		want    *time.Time
	}{
		{
			name: "returns create time when set",
			wrapper: DBClusterSnapshotWrapper{
				DBClusterSnapshot: &types.DBClusterSnapshot{
					SnapshotCreateTime: &now,
				},
			},
			want: &now,
		},
		{
			name: "returns nil when create time not set",
			wrapper: DBClusterSnapshotWrapper{
				DBClusterSnapshot: &types.DBClusterSnapshot{},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.wrapper.GetCreateTime()
			assert.Equal(t, tt.want, got)
		})
	}
}
