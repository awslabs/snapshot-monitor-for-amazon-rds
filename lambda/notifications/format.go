package notifications

import (
	"fmt"
	"strings"
)

func formatAggregatedMessage(changes []SnapshotStatusChange) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("RDS Snapshot Status Update Summary (%d changes)\n\n", len(changes)))

	changesByRegion := make(map[string][]SnapshotStatusChange)
	for _, change := range changes {
		changesByRegion[change.Region] = append(changesByRegion[change.Region], change)
	}

	for region, regionChanges := range changesByRegion {
		builder.WriteString(fmt.Sprintf("Region: %s\n", region))
		builder.WriteString("----------------------------------------\n")

		for _, change := range regionChanges {
			var statusTransition string
			if change.PreviousStatus == "" {
				statusTransition = fmt.Sprintf("New snapshot - Status: %s", change.CurrentStatus)
			} else {
				statusTransition = fmt.Sprintf("Status changed from %s to %s",
					change.PreviousStatus, change.CurrentStatus)
			}

			builder.WriteString(fmt.Sprintf("Snapshot: %s\n", change.SnapshotID))
			builder.WriteString(fmt.Sprintf("DB Instance: %s\n", change.DBInstance))
			builder.WriteString(fmt.Sprintf("Status: %s\n\n", statusTransition))
		}
	}

	return builder.String()
}
