package main

import (
	"context"
	"fmt"
	"os"
	"rds-backup-monitor/lambda/backups"
	"rds-backup-monitor/lambda/notifications"
	"rds-backup-monitor/lambda/storage"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

func handler(ctx context.Context) error {
	defaultConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("unable to load SDK config: %v", err)
	}
	ddbClient := dynamodb.NewFromConfig(defaultConfig)
	snsClient := sns.NewFromConfig(defaultConfig)

	regions := strings.Split(os.Getenv("REGIONS"), ",")
	if len(regions) == 0 {
		return fmt.Errorf("no regions provided")
	}
	for i, region := range regions {
		fmt.Printf("Monitoring Region %d: %s\n", i, region)
	}

	statusesToMonitor := strings.Split(os.Getenv("STATUS"), ",")
	if len(statusesToMonitor) == 0 {
		return fmt.Errorf("no statuses provided to monitor")
	}
	for i, status := range statusesToMonitor {
		fmt.Printf("Monitoring Status %d: %s\n", i, status)
	}

	for _, region := range regions {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
		if err != nil {
			return fmt.Errorf("unable to load SDK config for region %s: %v", region, err)
		}

		rdsClient := rds.NewFromConfig(cfg)
		sevenDaysAgo := time.Now().AddDate(0, 0, -7)

		// Get existing processed snapshots from DynamoDB
		processedSnapshots, err := storage.GetProcessedSnapshots(ctx, ddbClient, region)
		if err != nil {
			return fmt.Errorf("unable to get processed snapshots from DynamoDB in region %s: %v", region, err)
		}

		// Get instance snapshots in last 7 days
		snapshots, err := backups.GetFilteredSnapshots(ctx, rdsClient, sevenDaysAgo)
		if err != nil {
			return fmt.Errorf("unable to describe DB snapshots in region %s: %v", region, err)
		}

		// Get cluster snapshots in last 7 days
		clusterSnapshots, err := backups.GetFilteredClusterSnapshots(ctx, rdsClient, sevenDaysAgo)
		if err != nil {
			return fmt.Errorf("unable to describe DB cluster snapshots in region %s: %v", region, err)
		}

		// Compare with DynamoDB state and send summary report
		filteredSnapshots := backups.ProcessSnapshots(snapshots, clusterSnapshots)
		err = notifications.ProcessSnapshotChanges(ctx, filteredSnapshots, processedSnapshots, statusesToMonitor, region, snsClient, ddbClient)
		if err != nil {
			return fmt.Errorf("unable to process snapshots in region %s: %v", region, err)
		}
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
