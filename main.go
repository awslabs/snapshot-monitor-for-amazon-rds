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

	//Changed based on the statuses you would like to monitor.
	// For example, if you would like to just receive alerts for failed snapshots, change to []string{"failed"}
	status := []string{"available", "failed"}
	email := app.Node().TryGetContext(jsii.String("notification_email")).(string)
	if email == "" {
		log.Fatalf("unable to get notification email from context, %v", err)
	}

	rds_backup_monitor.NewRdsBackupMonitorStack(app, "RdsBackupMonitorStack", &rds_backup_monitor.RdsBackupMonitorStackProps{
		//Change based on your desired monitor frequency. Maximum granularity is 1 minute.
		//See: https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-scheduled-rule-pattern.html#eb-rate-expressions
		ScheduleExpression: jsii.String("rate(10 minutes)"),
		Regions:            &regions,
		Status:             &status,
		NotificationEmail:  jsii.String(email),
	})

	app.Synth(nil)
}
