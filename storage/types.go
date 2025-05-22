package storage

import "time"

type SnapshotInfo struct {
	SnapshotID   string
	SnapshotType string
	CreateTime   time.Time
	Status       string
}
