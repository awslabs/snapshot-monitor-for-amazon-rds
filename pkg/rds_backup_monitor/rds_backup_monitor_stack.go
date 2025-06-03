package rds_backup_monitor

import (
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseventstargets"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssns"
	awscdklambdagoalpha "github.com/aws/aws-cdk-go/awscdklambdagoalpha/v2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type RdsBackupMonitorStackProps struct {
	awscdk.StackProps
	ScheduleExpression *string
	Regions            *[]string
	Status             *[]string
	NotificationEmail  *string
}

func NewRdsBackupMonitorStack(scope constructs.Construct, id string, props *RdsBackupMonitorStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// Create SNS topic for notifications
	topic := awssns.NewTopic(stack, jsii.String("RdsSnapshotTopic"), &awssns.TopicProps{
		DisplayName: jsii.String("RDS Snapshot Notifications"),
	})

	awssns.NewSubscription(stack, jsii.String("RdsSnapshotSubscription"), &awssns.SubscriptionProps{
		Protocol: awssns.SubscriptionProtocol_EMAIL,
		Endpoint: jsii.String(*props.NotificationEmail),
		Topic:    topic,
	})

	// DynamoDB Table to record last checked time
	table := awsdynamodb.NewTableV2(stack, jsii.String("RdsBackupMonitorTable"), &awsdynamodb.TablePropsV2{
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("pk"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("sk"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		TimeToLiveAttribute: jsii.String("ttl"),
		PointInTimeRecovery: jsii.Bool(true),
	})

	lambdaFn := awscdklambdagoalpha.NewGoFunction(stack, jsii.String("RdsBackupMonitorFunction"), &awscdklambdagoalpha.GoFunctionProps{
		Runtime: awslambda.Runtime_PROVIDED_AL2023(),
		Entry:   jsii.String("lambda"),
		Timeout: awscdk.Duration_Seconds(jsii.Number(300)),
		Environment: &map[string]*string{
			"SNS_TOPIC_ARN":        topic.TopicArn(),
			"REGIONS":              jsii.String(strings.Join(*props.Regions, ",")),
			"STATUS":               jsii.String(strings.Join(*props.Status, ",")),
			"DYNAMODB_TABLE_NAME":  table.TableName(),
			"SCHEDULE_EXPRESSION": props.ScheduleExpression,
			"SNAPSHOT_AGE_DAYS":    jsii.String("7"), // Default to 7 days
		},
	})

	// Grant Lambda permission to describe DB snapshots and publish to SNS
	lambdaFn.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("rds:DescribeDBSnapshots", "rds:DescribeDBClusterSnapshots"),
		Resources: jsii.Strings("*"),
	}))
	lambdaFn.Role().AddManagedPolicy(
		awsiam.ManagedPolicy_FromAwsManagedPolicyName(
			jsii.String("service-role/AWSLambdaBasicExecutionRole")))
	topic.GrantPublish(lambdaFn)
	table.GrantReadWriteData(lambdaFn)

	// EventBridge schedule
	scheduleExpression := "rate(10 minutes)"
	if props.ScheduleExpression != nil {
		scheduleExpression = *props.ScheduleExpression
	}

	rule := awsevents.NewRule(stack, jsii.String("RdsBackupMonitorRule"), &awsevents.RuleProps{
		Schedule: awsevents.Schedule_Expression(jsii.String(scheduleExpression)),
	})

	rule.AddTarget(awseventstargets.NewLambdaFunction(lambdaFn, &awseventstargets.LambdaFunctionProps{}))

	return stack
}
