# Feeding Terraform into Helm
#
#   helm upgrade --install raisin ./deploy/helm/raisin -n raisin --create-namespace \
#     --set global.irsaRoleArn="$(terraform -chdir=deploy/terraform output -raw app_irsa_role_arn)" \
#     --set config.SQS_EVENTS_QUEUE_URL="$(terraform -chdir=deploy/terraform output -raw sqs_events_queue_url)" \
#     --set config.SES_CONFIGURATION_SET="$(terraform -chdir=deploy/terraform output -raw ses_configuration_set)" \
#     --set config.S3_BUCKET="$(terraform -chdir=deploy/terraform output -raw s3_bucket)" \
#     --set config.S3_INBOUND_BUCKET="$(terraform -chdir=deploy/terraform output -raw s3_inbound_bucket)" \
#     --set secrets.DATABASE_URL="postgres://raisin:...@.../raisin" \
#     --set secrets.REDIS_URL="redis://..." \
#     --set secrets.JWT_SECRET="..." \
#     --set secrets.BETTER_AUTH_SECRET="..."
#
# ServiceAccounts raisin-api / raisin-worker are annotated for IRSA when global.irsaRoleArn is set.
