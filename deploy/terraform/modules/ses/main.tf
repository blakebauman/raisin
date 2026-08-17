variable "name" { type = string }
variable "domain" { type = string }
variable "tags" { type = map(string) }
variable "inbound_bucket" {
  type        = string
  description = "S3 bucket for SES receipt rule (raw MIME)"
}
variable "inbound_bucket_arn" {
  type = string
}
variable "inbound_webhook_url" {
  type        = string
  description = "HTTPS endpoint for SES inbound SNS (POST /inbound/ses)"
  default     = "https://api.raisin.run/inbound/ses"
}

data "aws_caller_identity" "current" {}

resource "aws_sesv2_configuration_set" "main" {
  configuration_set_name = "${var.name}-events"
}

resource "aws_sns_topic" "ses_events" {
  name = "${var.name}-ses-events"
  tags = var.tags
}

resource "aws_sqs_queue" "ses_events" {
  name = "${var.name}-ses-events"
  tags = var.tags
}

resource "aws_sns_topic_subscription" "ses_to_sqs" {
  topic_arn = aws_sns_topic.ses_events.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.ses_events.arn
}

resource "aws_sesv2_configuration_set_event_destination" "sns" {
  configuration_set_name = aws_sesv2_configuration_set.main.configuration_set_name
  event_destination_name = "sns"
  event_destination {
    enabled              = true
    matching_event_types = ["SEND", "REJECT", "BOUNCE", "COMPLAINT", "DELIVERY", "OPEN", "CLICK", "DELIVERY_DELAY"]
    sns_destination {
      topic_arn = aws_sns_topic.ses_events.arn
    }
  }
}

resource "aws_sqs_queue_policy" "ses_events" {
  queue_url = aws_sqs_queue.ses_events.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "sns.amazonaws.com" }
      Action    = "sqs:SendMessage"
      Resource  = aws_sqs_queue.ses_events.arn
      Condition = { ArnEquals = { "aws:SourceArn" = aws_sns_topic.ses_events.arn } }
    }]
  })
}

# Inbound: SNS → API + S3 store
resource "aws_sns_topic" "inbound" {
  name = "${var.name}-inbound"
  tags = var.tags
}

resource "aws_sns_topic_subscription" "inbound_api" {
  topic_arn = aws_sns_topic.inbound.arn
  protocol  = "https"
  endpoint  = var.inbound_webhook_url
}

resource "aws_s3_bucket_policy" "inbound_ses" {
  bucket = var.inbound_bucket
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "AllowSESPuts"
      Effect    = "Allow"
      Principal = { Service = "ses.amazonaws.com" }
      Action    = "s3:PutObject"
      Resource  = "${var.inbound_bucket_arn}/*"
      Condition = {
        StringEquals = {
          "AWS:SourceAccount" = data.aws_caller_identity.current.account_id
        }
      }
    }]
  })
}

resource "aws_ses_receipt_rule_set" "main" {
  rule_set_name = "${var.name}-inbound"
}

resource "aws_ses_active_receipt_rule_set" "main" {
  rule_set_name = aws_ses_receipt_rule_set.main.rule_set_name
}

# Empty recipients = catch-all for domains verified in this SES account.
# Omit the attribute rather than [] — AWS provider treats empty lists inconsistently.
resource "aws_ses_receipt_rule" "inbound" {
  name          = "${var.name}-store"
  rule_set_name = aws_ses_receipt_rule_set.main.rule_set_name
  enabled       = true
  scan_enabled  = true

  s3_action {
    bucket_name       = var.inbound_bucket
    object_key_prefix = "ses/"
    position          = 1
  }

  sns_action {
    topic_arn = aws_sns_topic.inbound.arn
    position  = 2
  }

  depends_on = [aws_s3_bucket_policy.inbound_ses]
}

output "configuration_set" { value = aws_sesv2_configuration_set.main.configuration_set_name }
output "sqs_queue_url" { value = aws_sqs_queue.ses_events.url }
output "sqs_queue_arn" { value = aws_sqs_queue.ses_events.arn }
output "sns_topic_arn" { value = aws_sns_topic.ses_events.arn }
output "inbound_sns_topic_arn" { value = aws_sns_topic.inbound.arn }
output "receipt_rule_set" { value = aws_ses_receipt_rule_set.main.rule_set_name }
