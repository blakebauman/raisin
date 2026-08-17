#!/usr/bin/env bash
# Emit Helm --set flags from terraform outputs (non-secret infra only).
# Usage:
#   helm upgrade --install raisin ./deploy/helm/raisin -n raisin --create-namespace \
#     $(./scripts/helm-values-from-tf.sh) \
#     --set secrets.DATABASE_URL=... --set secrets.REDIS_URL=... \
#     --set secrets.JWT_SECRET=... --set secrets.BETTER_AUTH_SECRET=...
set -euo pipefail

TF="${1:-$(cd "$(dirname "$0")/.." && pwd)/deploy/terraform}"
out() { terraform -chdir="$TF" output -raw "$1"; }

printf '%s\n' \
  "--set global.irsaRoleArn=$(out app_irsa_role_arn)" \
  "--set config.SQS_EVENTS_QUEUE_URL=$(out sqs_events_queue_url)" \
  "--set config.SES_CONFIGURATION_SET=$(out ses_configuration_set)" \
  "--set config.SES_SNS_TOPIC_ARN=$(out sns_topic_arn)" \
  "--set config.S3_BUCKET=$(out s3_bucket)" \
  "--set config.S3_INBOUND_BUCKET=$(out s3_inbound_bucket)"
