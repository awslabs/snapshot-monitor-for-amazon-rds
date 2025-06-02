package notifications

import (
	"context"
	"fmt"
	"os"

	"rds-backup-monitor/lambda/storage"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type SNSClient interface {
	Publish(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

func contains(slice []string, str string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}

func ProcessSnapshotChanges(ctx context.Context, filteredSnapshots []storage.SnapshotInfo,
	processedSnapshots map[string]string, statusesToMonitor []string,
	region string, snsClient SNSClient, ddbClient storage.DDBClient) error {

	var statusChanges []SnapshotStatusChange
	var snapshotsToUpdate []storage.SnapshotInfo

	for _, snapshot := range filteredSnapshots {
		currentStatus := snapshot.Status
		previousStatus, exists := processedSnapshots[snapshot.SnapshotID]

		if contains(statusesToMonitor, currentStatus) {
			fmt.Printf("Checking snapshot %s in region %s\n", snapshot.SnapshotID, region)

			if !exists || previousStatus != string(currentStatus) {
				statusChanges = append(statusChanges, SnapshotStatusChange{
					SnapshotID:     snapshot.SnapshotID,
					CurrentStatus:  string(currentStatus),
					PreviousStatus: previousStatus,
					DBInstance:     snapshot.SnapshotID,
					Region:         region,
				})
				snapshotsToUpdate = append(snapshotsToUpdate, snapshot)
			}
		}
	}

	if len(statusChanges) > 0 {
		message := formatAggregatedMessage(statusChanges)

		_, err := snsClient.Publish(ctx, &sns.PublishInput{
			TopicArn: aws.String(os.Getenv("SNS_TOPIC_ARN")),
			Message:  aws.String(message),
		})

		if err != nil {
			return fmt.Errorf("unable to publish SNS message: %v", err)
		}

		// Update all snapshot states in a single batch operation
		err = storage.BatchUpdateSnapshotStates(ctx, ddbClient, region, snapshotsToUpdate)
		if err != nil {
			return fmt.Errorf("failed to batch update snapshot states: %v", err)
		}
	}

	return nil
}
