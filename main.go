package main

import (
	"context"
	"log"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/account"
	accountTypes "github.com/aws/aws-sdk-go-v2/service/account/types"
	"github.com/aws/jsii-runtime-go"

	"rds-backup-monitor/pkg/rds_backup_monitor"
)

func main() {
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	aws_account_client := account.NewFromConfig(cfg)
	available_regions, err := aws_account_client.ListRegions(ctx, &account.ListRegionsInput{
		RegionOptStatusContains: []accountTypes.RegionOptStatus{
			accountTypes.RegionOptStatusEnabled,
			accountTypes.RegionOptStatusEnabledByDefault,
		},
	})
	if err != nil {
		log.Fatalf("unable to list regions, %v", err)
	}

	app := awscdk.NewApp(&awscdk.AppProps{})

	regions := []string{}
	for _, region := range available_regions.Regions {
		regions = append(regions, *region.RegionName)
	}

	// Get configuration from context
	email := app.Node().TryGetContext(jsii.String("notification_email")).(string)
	if email == "" {
		log.Fatalf("unable to get notification email from context")
	}
	
	// Get status configuration from context or use defaults
	var status []string
	statusContext := app.Node().TryGetContext(jsii.String("status_to_monitor"))
	if statusContext != nil {
		if statusArray, ok := statusContext.([]interface{}); ok {
			for _, s := range statusArray {
				if str, ok := s.(string); ok {
					status = append(status, str)
				}
			}
		}
	}
	// Use defaults if not provided
	if len(status) == 0 {
		status = []string{"available", "failed"}
	}
	
	// Get schedule from context or use default
	scheduleExpression := "rate(10 minutes)"
	scheduleContext := app.Node().TryGetContext(jsii.String("schedule_expression"))
	if scheduleContext != nil {
		if scheduleStr, ok := scheduleContext.(string); ok && scheduleStr != "" {
			scheduleExpression = scheduleStr
		}
	}

	// Get snapshot age days from context or use default
	snapshotAgeDays := "7"
	snapshotAgeContext := app.Node().TryGetContext(jsii.String("snapshot_age_days"))
	if snapshotAgeContext != nil {
		if snapshotAgeStr, ok := snapshotAgeContext.(string); ok && snapshotAgeStr != "" {
			snapshotAgeDays = snapshotAgeStr
		}
	}

	rds_backup_monitor.NewRdsBackupMonitorStack(app, "RdsBackupMonitorStack", &rds_backup_monitor.RdsBackupMonitorStackProps{
		//Change based on your desired monitor frequency. Maximum granularity is 1 minute.
		//See: https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-scheduled-rule-pattern.html#eb-rate-expressions
		ScheduleExpression: jsii.String(scheduleExpression),
		Regions:            &regions,
		Status:             &status,
		NotificationEmail:  jsii.String(email),
		SnapshotAgeDays:    jsii.String(snapshotAgeDays),
	})

	app.Synth(nil)
}
