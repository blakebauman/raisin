# Feeding Terraform into Helm
#
# Non-secret wiring (preferred):
#   helm upgrade --install raisin ./deploy/helm/raisin -n raisin --create-namespace \
#     $(./scripts/helm-values-from-tf.sh) \
#     --set secrets.DATABASE_URL="postgres://raisin:...@.../raisin" \
#     --set secrets.REDIS_URL="redis://..." \
#     --set secrets.JWT_SECRET="..." \
#     --set secrets.BETTER_AUTH_SECRET="..." \
#     --set api.ingress.tls.clusterIssuer=letsencrypt-prod \
#     --set worker.ingress.tls.clusterIssuer=letsencrypt-prod \
#     --set console.ingress.tls.clusterIssuer=letsencrypt-prod \
#     --set smtp.tls.enabled=true --set smtp.tls.secretName=raisin-smtp-tls
#
# Or expand the helper output and set secrets separately. Prefer ExternalSecrets /
# SealedSecrets for DATABASE_URL / JWT / Better Auth in production — do not commit them.
#
# Manual equivalent:
#   --set global.irsaRoleArn="$(terraform -chdir=deploy/terraform output -raw app_irsa_role_arn)" \
#   --set config.SQS_EVENTS_QUEUE_URL="$(terraform -chdir=deploy/terraform output -raw sqs_events_queue_url)" \
#   --set config.SES_CONFIGURATION_SET="$(terraform -chdir=deploy/terraform output -raw ses_configuration_set)" \
#   --set config.SES_SNS_TOPIC_ARN="$(terraform -chdir=deploy/terraform output -raw sns_topic_arn)" \
#   --set config.S3_BUCKET="$(terraform -chdir=deploy/terraform output -raw s3_bucket)" \
#   --set config.S3_INBOUND_BUCKET="$(terraform -chdir=deploy/terraform output -raw s3_inbound_bucket)"
#
# Migrations: api pods run an initContainer (api.migrateInit) with scripts/migrate.sh
# (versioned via schema_migrations). Keep Helm copies in sync:
#   make helm-sync
# Optional: --set migrate.enabled=true for a post-install Job as well.
#
# SMTP STARTTLS: create a Secret with tls.crt/tls.key, then --set smtp.tls.enabled=true.
#
# Console images bake NEXT_PUBLIC_API_URL at build time (CI defaults to https://api.raisin.run).
# Runtime RAISIN_API_URL (Helm) is used by the server-side /api/proxy.
#
# ServiceAccounts raisin-api / raisin-worker are annotated for IRSA when global.irsaRoleArn is set.
