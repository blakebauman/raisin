terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "project" {
  type    = string
  default = "raisin"
}

variable "environment" {
  type    = string
  default = "prod"
}

variable "domain" {
  type    = string
  default = "raisin.run"
}

variable "inbound_webhook_url" {
  type    = string
  default = "https://api.raisin.run/inbound/ses"
}

variable "db_password" {
  type        = string
  sensitive   = true
  description = "Postgres master password for RDS"
}

locals {
  name = "${var.project}-${var.environment}"
  tags = {
    Project     = var.project
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

module "networking" {
  source = "./modules/networking"
  name   = local.name
  tags   = local.tags
}

module "eks" {
  source     = "./modules/eks"
  name       = local.name
  vpc_id     = module.networking.vpc_id
  subnet_ids = module.networking.private_subnet_ids
  tags       = local.tags
}

module "rds" {
  source             = "./modules/rds"
  name               = local.name
  vpc_id             = module.networking.vpc_id
  subnet_ids         = module.networking.private_subnet_ids
  security_group_ids = [module.networking.db_sg_id]
  db_password        = var.db_password
  tags               = local.tags
}

module "redis" {
  source             = "./modules/redis"
  name               = local.name
  subnet_ids         = module.networking.private_subnet_ids
  security_group_ids = [module.networking.redis_sg_id]
  tags               = local.tags
}

module "s3" {
  source = "./modules/s3"
  name   = local.name
  tags   = local.tags
}

module "ses" {
  source              = "./modules/ses"
  name                = local.name
  domain              = var.domain
  inbound_bucket      = module.s3.inbound_bucket
  inbound_bucket_arn  = module.s3.inbound_arn
  inbound_webhook_url = var.inbound_webhook_url
  tags                = local.tags
}

# IRSA permissions for api/worker pods (S3 attachments, SQS events, SES send)
data "aws_iam_policy_document" "app" {
  statement {
    sid = "S3Attachments"
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
      "s3:ListBucket",
    ]
    resources = [
      module.s3.attachments_arn,
      "${module.s3.attachments_arn}/*",
      module.s3.inbound_arn,
      "${module.s3.inbound_arn}/*",
    ]
  }
  statement {
    sid = "SQSEvents"
    actions = [
      "sqs:ReceiveMessage",
      "sqs:DeleteMessage",
      "sqs:GetQueueAttributes",
      "sqs:ChangeMessageVisibility",
    ]
    resources = [module.ses.sqs_queue_arn]
  }
  statement {
    sid = "SES"
    actions = [
      "ses:SendEmail",
      "ses:SendRawEmail",
      "ses:GetEmailIdentity",
      "ses:CreateEmailIdentity",
      "ses:DeleteEmailIdentity",
    ]
    resources = ["*"]
  }
}

resource "aws_iam_role_policy" "app" {
  name   = "${local.name}-raisin-app"
  role   = module.eks.app_role_name
  policy = data.aws_iam_policy_document.app.json
}

output "eks_cluster_name" {
  value = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  value = module.eks.cluster_endpoint
}

output "rds_endpoint" {
  value     = module.rds.endpoint
  sensitive = true
}

output "redis_endpoint" {
  value = module.redis.endpoint
}

output "s3_bucket" {
  value = module.s3.bucket
}

output "s3_inbound_bucket" {
  value = module.s3.inbound_bucket
}

output "ses_configuration_set" {
  value = module.ses.configuration_set
}

output "sqs_events_queue_url" {
  value = module.ses.sqs_queue_url
}

output "app_irsa_role_arn" {
  value       = module.eks.app_role_arn
  description = "Annotate raisin-api / raisin-worker ServiceAccounts with this role ARN"
}
