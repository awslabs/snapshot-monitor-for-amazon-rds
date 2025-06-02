# Snapshot Monitor for Amazon RDS

This CDK app implements a monitor for new snapshots in Amazon RDS. It uses an EventBridge schedule to periodically check for snapshots of specific statuses across specified regions and sends notifications via SNS. By default the monitor checks for new **available** and **failed** snapshots, but you can edit this to just check for a specific status (e.g. failed snapshots).

## Features

- Configurable EventBridge schedule (default: every 10 minutes)
- Monitors multiple regions
- SNS notifications for failed snapshots

## Architecture

- EventBridge rule triggers a Lambda function on a schedule
- Lambda function checks for matching RDS snapshots using the `describe-db-snapshots` API
- Matching (e.g. failed) snapshots trigger SNS notifications

The high-level architecture is shown below:

![High-level architecture](/images/architecture.png)

## Requirements

- [Go 1.18+](https://go.dev/doc/install)
- [AWS CDK v2](https://docs.aws.amazon.com/cdk/v2/guide/getting-started.html)
- AWS credentials configured with appropriate permissions

## Usage

1. Install dependencies:

```
go mod download
```

2. Deploy the stack:

```
 cdk deploy -c notification_email=<email address to receive snapshot summary report>
```

3. To customize the deployment, use CDK context parameters:

```
cdk deploy -c notification_email=your-email@example.com -c schedule_expression="rate(30 minutes)" -c status_to_monitor=failed,available
```

Or modify the `cdk.json` file:

```json
{
  "app": "go mod download && go run main.go",
  "context": {
    "notification_email": "your-email@example.com",
    "status_to_monitor": ["available", "failed"],
    "schedule_expression": "rate(10 minutes)"
  }
}
```

## Configuration

- `notification_email`: Email address to receive snapshot notifications
- `schedule_expression`: Cron or rate expression for the EventBridge rule (default: "rate(10 minutes)")
- `status_to_monitor`: List of snapshot statuses to monitor (default: ["available", "failed"])
- `Regions`: List of AWS regions to monitor (default: all enabled regions)


## Testing

Unit tests can be run from the root of the project with go test:

```bash
go test ./...
```