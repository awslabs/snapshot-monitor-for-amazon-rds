package types

type Configuration struct {
	Regions            []string
	StatusesToMonitor  []string
	ScheduleExpression string
	SnapshotAgeDays    int
}
