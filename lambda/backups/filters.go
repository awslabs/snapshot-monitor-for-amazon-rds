package backups

import (
	"context"
	"rds-backup-monitor/lambda/storage"
	"time"
)

func filterSnapshots[T SnapshotFilter](snapshots []T, cutoffTime time.Time) []T {
	var filtered []T
	for _, snapshot := range snapshots {
		if createTime := snapshot.GetCreateTime(); createTime != nil && createTime.After(cutoffTime) {
			filtered = append(filtered, snapshot)
		}
	}
	return filtered
}

func getFilteredSnapshotsGeneric[T SnapshotFilter](
	ctx context.Context,
	nextPage func(context.Context) ([]T, error),
	hasMore func() bool,
	cutoffTime time.Time,
) ([]T, error) {
	var allSnapshots []T

	for hasMore() {
		snapshots, err := nextPage(ctx)
		if err != nil {
			return nil, err
		}
		allSnapshots = append(allSnapshots, snapshots...)
	}

	return filterSnapshots(allSnapshots, cutoffTime), nil
}

func ProcessSnapshots(instanceSnapshots []DBSnapshotWrapper, clusterSnapshots []DBClusterSnapshotWrapper) []storage.SnapshotInfo {
	var results []storage.SnapshotInfo

	for _, snapshot := range instanceSnapshots {
		results = append(results, storage.SnapshotInfo{
			SnapshotID:   *snapshot.DBSnapshotIdentifier,
			SnapshotType: "instance",
			CreateTime:   *snapshot.SnapshotCreateTime,
			Status:       string(*snapshot.Status),
		})
	}

	for _, snapshot := range clusterSnapshots {
		results = append(results, storage.SnapshotInfo{
			SnapshotID:   *snapshot.DBClusterSnapshotIdentifier,
			SnapshotType: "cluster",
			CreateTime:   *snapshot.SnapshotCreateTime,
			Status:       string(*snapshot.Status),
		})
	}

	return results
}
