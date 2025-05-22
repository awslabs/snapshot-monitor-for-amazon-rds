package notifications

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatAggregatedMessage(t *testing.T) {
	tests := []struct {
		name    string
		changes []SnapshotStatusChange
		want    string
	}{
		{
			name: "formats single status change",
			changes: []SnapshotStatusChange{
				{
					SnapshotID:     "snap-1",
					CurrentStatus:  "available",
					PreviousStatus: "creating",
					DBInstance:     "db-1",
					Region:         "us-west-2",
				},
			},
			want: "RDS Snapshot Status Update Summary (1 changes)\n\n" +
				"Region: us-west-2\n" +
				"----------------------------------------\n" +
				"Snapshot: snap-1\n" +
				"DB Instance: db-1\n" +
				"Status: Status changed from creating to available\n\n",
		},
		{
			name: "formats multiple status changes",
			changes: []SnapshotStatusChange{
				{
					SnapshotID:     "snap-1",
					CurrentStatus:  "available",
					PreviousStatus: "creating",
					DBInstance:     "db-1",
					Region:         "us-west-2",
				},
				{
					SnapshotID:     "snap-2",
					CurrentStatus:  "error",
					PreviousStatus: "available",
					DBInstance:     "db-2",
					Region:         "us-west-2",
				},
			},
			want: "RDS Snapshot Status Update Summary (2 changes)\n\n" +
				"Region: us-west-2\n" +
				"----------------------------------------\n" +
				"Snapshot: snap-1\n" +
				"DB Instance: db-1\n" +
				"Status: Status changed from creating to available\n\n" +
				"Snapshot: snap-2\n" +
				"DB Instance: db-2\n" +
				"Status: Status changed from available to error\n\n",
		},
		{
			name:    "handles empty changes",
			changes: []SnapshotStatusChange{},
			want:    "RDS Snapshot Status Update Summary (0 changes)\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAggregatedMessage(tt.changes)
			assert.Equal(t, tt.want, got)
		})
	}
}
