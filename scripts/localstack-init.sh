#!/bin/sh
set -e
awslocal s3 mb s3://raisin-attachments || true
awslocal s3 mb s3://raisin-inbound || true
awslocal sqs create-queue --queue-name raisin-ses-events || true
awslocal sns create-topic --name raisin-ses-events || true
awslocal sns create-topic --name raisin-inbound || true

TOPIC_ARN=$(awslocal sns list-topics --query "Topics[?contains(TopicArn, 'raisin-ses-events')].TopicArn" --output text | head -n1)
QUEUE_URL=$(awslocal sqs get-queue-url --queue-name raisin-ses-events --query QueueUrl --output text)
QUEUE_ARN=$(awslocal sqs get-queue-attributes --queue-url "$QUEUE_URL" --attribute-names QueueArn --query Attributes.QueueArn --output text)

awslocal sns subscribe \
  --topic-arn "$TOPIC_ARN" \
  --protocol sqs \
  --notification-endpoint "$QUEUE_ARN" \
  --attributes '{"RawMessageDelivery":"true"}' || true

awslocal sqs set-queue-attributes --queue-url "$QUEUE_URL" --attributes "{\"Policy\":\"{\\\"Version\\\":\\\"2012-10-17\\\",\\\"Statement\\\":[{\\\"Effect\\\":\\\"Allow\\\",\\\"Principal\\\":{\\\"Service\\\":\\\"sns.amazonaws.com\\\"},\\\"Action\\\":\\\"sqs:SendMessage\\\",\\\"Resource\\\":\\\"$QUEUE_ARN\\\",\\\"Condition\\\":{\\\"ArnEquals\\\":{\\\"aws:SourceArn\\\":\\\"$TOPIC_ARN\\\"}}}]}\"}" || true

echo "LocalStack init complete"
echo "SQS_EVENTS_QUEUE_URL=$QUEUE_URL"
echo "SNS_TOPIC_ARN=$TOPIC_ARN"
echo "S3_INBOUND_BUCKET=raisin-inbound"
