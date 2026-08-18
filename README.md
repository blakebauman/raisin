# Raisin

Developer-first email API on Go, Amazon SES, EKS, and Postgres.

**Repo:** [blakebauman/raisin](https://github.com/blakebauman/raisin) · **Domain:** [raisin.run](https://raisin.run)

## Stack

- **API / workers**: Go (chi, pgx, Asynq)
- **Jobs**: Asynq (Redis) for sends/webhooks; **SQS** for SES events
- **Console**: Next.js + Better Auth (app.raisin.run)
- **Mail**: Amazon SES v2 (Mailpit locally) + SMTP relay (smtp.raisin.run)
- **Data**: Postgres + Redis
- **Platform extras**: Automations, dedicated IPs + warmup, multi-region SES, OAuth apps, CLI, MCP, visual template editor, BIMI, domain claiming, inbound receive
- **Deploy**: Terraform + Helm on EKS

## Hosts

| Host | Service |
|------|---------|
| `api.raisin.run` | Public REST API |
| `app.raisin.run` | Console |
| `t.raisin.run` | Open/click tracking |
| `smtp.raisin.run` | SMTP relay |

## Quick start (local)

```bash
cp .env.example .env
docker compose up -d                 # postgres, redis, mailpit, localstack
make api                             # :18080
make worker                          # :18081 + Asynq (+ SQS if configured)
make smtp                            # :2525

cd apps/console && pnpm install && pnpm dev   # :3000 — landing `/`, console `/login` → `/overview`
```

Or run the Go apps + console in Docker:

```bash
make compose-apps   # docker compose --profile apps up -d --build
```

Local ports (remapped to avoid host conflicts): API `18080`, worker/tracking `18081`, Postgres `5433`, Mailpit UI `8026`, LocalStack `4566`. Console defaults to **`3001`** via `make console` when `3000` is occupied.

```bash
make smoke   # end-to-end API check (API+worker must be running)
make seed    # rich demo data for UI review (contacts, emails, broadcasts, …)
make console # Next.js on :3001 — login → “Continue with seeded demo team”
```

Demo API key: `ra_demo_00000000000000000000000000000000`

Migrations: `001_init.sql`, `002_better_auth.sql`, and `003_platform_extras.sql` (automations, IPs, OAuth, BIMI/claims, inbound harden). Compose applies them on first boot; `make migrate` applies all against `DATABASE_URL`.

### CLI / MCP

```bash
go run ./apps/cli --api-key "$RAISIN_API_KEY" emails list
pnpm --filter @raisin-run/mcp-server start   # RAISIN_API_KEY required
```

### Domain verify (local)

With `SENDER_DRIVER=mailpit`, `POST /domains/{id}/verify` marks the domain verified immediately (stub identity). With SES, it polls Amazon for DKIM/sending status.

### SES events (local)

LocalStack init creates the SNS topic + SQS queue and subscribes them. Set:

```bash
SQS_EVENTS_QUEUE_URL=http://localhost:4566/000000000000/raisin-ses-events
AWS_ENDPOINT_URL=http://localhost:4566
```

Compose `apps` profile wires this on the worker. For Mailpit-only local sends, the worker still marks delivery via the send path; use `POST` to tracking `/test/events/` in test mode to synthesize opens/clicks.

### Console env

| Var | Purpose |
|-----|---------|
| `RAISIN_API_URL` | Server-side proxy → API (compose: `http://api:8080`) |
| `NEXT_PUBLIC_API_URL` | Fallback API URL for server routes |
| `JWT_SECRET` | Must match API for demo/provision tokens |
| `BETTER_AUTH_SECRET` / `BETTER_AUTH_URL` | Better Auth |
| `DATABASE_URL` | Console Better Auth Postgres |

Browser dashboard calls go through `/api/proxy/*` so the API origin stays server-side.

### Webhook signatures

Header: `Raisin-Signature: t=<unix>,v1=<hex>` over `${t}.${rawBody}` (HMAC-SHA256).

```ts
import { verifyWebhookSignature } from "@raisin-run/sdk";
await verifyWebhookSignature(secret, header, rawBody);
```

```go
raisin.VerifyWebhookSignature(secret, header, body, 0)
```

```python
from raisin import verify_webhook_signature
verify_webhook_signature(secret, header, body)
```

## SDKs

```ts
import { Raisin } from "@raisin-run/sdk";
const raisin = new Raisin("ra_…");
await raisin.emails.send({
  from: "Acme <hi@acme.com>",
  to: "user@example.com",
  subject: "Hello",
  html: "<p>Hi</p>",
});
```

```go
client := raisin.NewClient("ra_…")
client.Emails.Send(ctx, &raisin.SendEmailRequest{...})
```

```python
from raisin import Raisin
Raisin("ra_…").emails.send(from_="Acme <hi@acme.com>", to="u@x.com", subject="Hi", html="<p>Hi</p>")
```

## Deploy (AWS)

```bash
cd deploy/terraform
cp terraform.tfvars.example terraform.tfvars   # set db_password via TF_VAR_db_password
terraform init && terraform apply

# Install chart with IRSA + SQS wiring — see deploy/helm/raisin/VALUES.md
```

Private subnets use a NAT gateway; EKS gets a managed node group + OIDC/IRSA role for api/worker (S3, SQS, SES).

## License

Proprietary — see [LICENSE](./LICENSE).
