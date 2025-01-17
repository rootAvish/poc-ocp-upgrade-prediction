package sqsconnect

import (
	"github.com/aws/aws-sdk-go/aws"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"go.uber.org/zap"
	"os"
	"strconv"
)

var logger, _ = zap.NewDevelopment()
var sugarLogger = logger.Sugar()
var AWSService *sqs.SQS
var AWSSQSQueueUrl string
var AWSSQSQueueName string

// Create a new SQS queue if not created already to store  the entire raw collection of runtime call stacks
func CreateSQSQueue() {
	queuename := os.Getenv("AWS_SQS_QUEUE_NAME")

	if queuename == "" {
		queuename = "ocp-test-upgrade"
	}

	AWSSQSRegion := os.Getenv("AWS_SQS_REGION")
	if AWSSQSRegion == "" {
		AWSSQSRegion = "us-west-2"
	}
	sess, err := awssession.NewSession(&aws.Config{
		Region: aws.String(AWSSQSRegion)},
	)

	if err != nil {
		sugarLogger.Infof("Could not connect to AWS for creating SQS queue %s : %v", queuename, err)
		return
	}

	AWSService = sqs.New(sess)
	qurl, err := AWSService.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: &queuename,
	})
	if err == nil && *qurl.QueueUrl != "" {
		AWSSQSQueueUrl = *qurl.QueueUrl
	}
	response, err := AWSService.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(queuename),
		Attributes: map[string]*string{
			"DelaySeconds":           aws.String("120"),
			"MessageRetentionPeriod": aws.String("172800"),
		},
	})

	if err != nil {
		sugarLogger.Infof("Could not create SQS queue %s : %v", queuename, err)
	} else {
		AWSSQSQueueUrl = *response.QueueUrl
	}
}

// publish entire raw collection of runtime call stacks
func PublishCallStack(callstackjson string, callstackid int) {
	if AWSService == nil {
		CreateSQSQueue()
	} else {
		_, err := AWSService.SendMessage(&sqs.SendMessageInput{
			DelaySeconds: aws.Int64(20),
			MessageAttributes: map[string]*sqs.MessageAttributeValue{
				"CallStack_ID": &sqs.MessageAttributeValue{
					DataType:    aws.String("String"),
					StringValue: aws.String(strconv.Itoa(callstackid)),
				},
			},
			MessageBody: aws.String(callstackjson),
			QueueUrl:    &AWSSQSQueueUrl,
		})
		if err != nil {
			sugarLogger.Errorf("Failed to put message in queue. Error: %v\n", err)
		}
	}
}
