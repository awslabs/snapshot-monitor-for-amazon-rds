package notifications

type SnapshotStatusChange struct {
	SnapshotID     string
	CurrentStatus  string
	PreviousStatus string
	DBInstance     string
	Region         string
}
