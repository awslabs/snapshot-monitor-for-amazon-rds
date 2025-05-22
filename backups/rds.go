package backups

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rds"
)

type RDSClient interface {
	DescribeDBSnapshots(ctx context.Context, params *rds.DescribeDBSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBSnapshotsOutput, error)
	DescribeDBClusterSnapshots(ctx context.Context, params *rds.DescribeDBClusterSnapshotsInput, optFns ...func(*rds.Options)) (*rds.DescribeDBClusterSnapshotsOutput, error)
}

func (s DBSnapshotWrapper) GetCreateTime() *time.Time {
	return s.SnapshotCreateTime
}

func (s DBClusterSnapshotWrapper) GetCreateTime() *time.Time {
	return s.SnapshotCreateTime
}

func GetFilteredSnapshots(ctx context.Context, rdsClient RDSClient, cutoffTime time.Time) ([]DBSnapshotWrapper, error) {
	paginator := rds.NewDescribeDBSnapshotsPaginator(rdsClient, &rds.DescribeDBSnapshotsInput{})

	return getFilteredSnapshotsGeneric(
		ctx,
		func(ctx context.Context) ([]DBSnapshotWrapper, error) {
			output, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("error getting DB snapshots page: %v", err)
			}
			// Convert to wrapper type
			wrappers := make([]DBSnapshotWrapper, len(output.DBSnapshots))
			for i, snapshot := range output.DBSnapshots {
				snapshotCopy := snapshot
				wrappers[i] = DBSnapshotWrapper{&snapshotCopy}
			}
			return wrappers, nil
		},
		paginator.HasMorePages,
		cutoffTime,
	)
}

func GetFilteredClusterSnapshots(ctx context.Context, rdsClient RDSClient, cutoffTime time.Time) ([]DBClusterSnapshotWrapper, error) {
	paginator := rds.NewDescribeDBClusterSnapshotsPaginator(rdsClient, &rds.DescribeDBClusterSnapshotsInput{})

	return getFilteredSnapshotsGeneric(
		ctx,
		func(ctx context.Context) ([]DBClusterSnapshotWrapper, error) {
			output, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("error getting DB cluster snapshots page: %v", err)
			}

			wrappers := make([]DBClusterSnapshotWrapper, len(output.DBClusterSnapshots))
			for i, snapshot := range output.DBClusterSnapshots {
				snapshotCopy := snapshot
				wrappers[i] = DBClusterSnapshotWrapper{&snapshotCopy}
			}
			return wrappers, nil
		},
		paginator.HasMorePages,
		cutoffTime,
	)
}
