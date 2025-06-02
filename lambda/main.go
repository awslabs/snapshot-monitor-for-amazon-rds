package main

import (
	"context"
	"fmt"
	"os"
	"rds-backup-monitor/lambda/backups"
	"rds-backup-monitor/lambda/notifications"
	"rds-backup-monitor/lambda/storage"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

type Configuration struct {
	Regions          []string
	StatusesToMonitor []string
	ScheduleExpression string
	SnapshotAgeDays   int
}

var (
	ddbClient *dynamodb.Client
	snsClient *sns.Client
	appConfig Configuration
)

func init() {
	defaultConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config: %v", err))
	}

	ddbClient = dynamodb.NewFromConfig(defaultConfig)
	snsClient = sns.NewFromConfig(defaultConfig)

	// Get snapshot age from environment or use default
	snapshotAgeDays := 7 // Default to 7 days
	if ageStr := os.Getenv("SNAPSHOT_AGE_DAYS"); ageStr != "" {
		if age, err := strconv.Atoi(ageStr); err == nil && age > 0 {
			snapshotAgeDays = age
		}
	}

	// Initialize application configuration
	appConfig = Configuration{
		Regions:           strings.Split(os.Getenv("REGIONS"), ","),
		StatusesToMonitor: strings.Split(os.Getenv("STATUS"), ","),
		ScheduleExpression: os.Getenv("SCHEDULE_EXPRESSION"),
		SnapshotAgeDays:   snapshotAgeDays,
	}

	// Validate configuration
	if len(appConfig.Regions) == 0 {
		panic("no regions provided in configuration")
	}
	if len(appConfig.StatusesToMonitor) == 0 {
		panic("no statuses provided to monitor in configuration")
	}
	if appConfig.ScheduleExpression == "" {
		appConfig.ScheduleExpression = "rate(10 minutes)" // Default schedule
	}
}

func handler(ctx context.Context) error {
	// Log configuration
	for i, region := range appConfig.Regions {
		fmt.Printf("Monitoring Region %d: %s\n", i, region)
	}

	for i, status := range appConfig.StatusesToMonitor {
		fmt.Printf("Monitoring Status %d: %s\n", i, status)
	}

	fmt.Printf("Schedule Expression: %s\n", appConfig.ScheduleExpression)
	fmt.Printf("Snapshot Age: %d days\n", appConfig.SnapshotAgeDays)

	for _, region := range appConfig.Regions {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
		if err != nil {
			return fmt.Errorf("unable to load SDK config for region %s: %v", region, err)
		}

		rdsClient := rds.NewFromConfig(cfg)
		cutoffDate := time.Now().AddDate(0, 0, -appConfig.SnapshotAgeDays)

		// Get existing processed snapshots from DynamoDB
		processedSnapshots, err := storage.GetProcessedSnapshots(ctx, ddbClient, region)
		if err != nil {
			return fmt.Errorf("unable to get processed snapshots from DynamoDB in region %s: %v", region, err)
		}

		// Get instance snapshots based on configured age
		snapshots, err := backups.GetFilteredSnapshots(ctx, rdsClient, cutoffDate)
		if err != nil {
			return fmt.Errorf("unable to describe DB snapshots in region %s: %v", region, err)
		}

		// Get cluster snapshots based on configured age
		clusterSnapshots, err := backups.GetFilteredClusterSnapshots(ctx, rdsClient, cutoffDate)
		if err != nil {
			return fmt.Errorf("unable to describe DB cluster snapshots in region %s: %v", region, err)
		}

		// Compare with DynamoDB state and send summary report
		filteredSnapshots := backups.ProcessSnapshots(snapshots, clusterSnapshots)
		err = notifications.ProcessSnapshotChanges(ctx, filteredSnapshots, processedSnapshots, appConfig.StatusesToMonitor, region, snsClient, ddbClient)
		if err != nil {
			return fmt.Errorf("unable to process snapshots in region %s: %v", region, err)
		}
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
