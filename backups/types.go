package backups

import (
	"time"

	rdsTypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
)

type SnapshotFilter interface {
	GetCreateTime() *time.Time
}

type DBSnapshotWrapper struct {
	*rdsTypes.DBSnapshot
}

type DBClusterSnapshotWrapper struct {
	*rdsTypes.DBClusterSnapshot
}
