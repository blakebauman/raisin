# Architecture

Raisin is a developer email API: four processes, Postgres, Redis, and Amazon SES. This page is the system as the repo implements it today — not the cheaper launch target (see [Scaling](./scaling.md)).

## Hosts

| Host | Process | Role |
|------|---------|------|
| `api.raisin.run` | `raisin-api` | Public REST API |
| `app.raisin.run` | `raisin-console` | Next.js console + Better Auth |
| `t.raisin.run` | `raisin-worker` | Open / click / unsubscribe HTTP |
| `smtp.raisin.run` | `raisin-smtp` | SMTP relay (`:25` / `:587`) |

Local compose remaps API `18080`, worker `18081`, SMTP `2525`, Postgres `5433`.

## System context

```mermaid
flowchart TB
  subgraph clients [Clients]
    Dev[SDK / CLI / MCP]
    User[Console user]
    Inbox[Recipient]
    Stripe[Stripe]
  end

  subgraph edge [Raisin edge]
    API[api.raisin.run]
    App[app.raisin.run]
    Track[t.raisin.run]
    SMTP[smtp.raisin.run]
  end

  subgraph data [Data plane]
    SES[Amazon SES]
    PG[(Postgres)]
    Redis[(Redis / Asynq)]
    S3[(S3)]
  end

  Dev -->|Bearer ra_| API
  User --> App
  App -->|server /api/proxy| API
  Dev -->|AUTH PLAIN API key| SMTP
  Inbox -->|open / click / unsub| Track
  Stripe -->|/billing/webhook| API
  API --> PG
  API --> Redis
  SMTP --> PG
  SMTP --> Redis
  Track --> PG
  API --> S3
  Redis -->|email:send| SES
  SES -->|SNS / SQS events| Redis
  SES --> Inbox
```

The console never calls the API from the browser with a team key. Dashboard traffic goes through Next.js `/api/proxy/*` so the API origin stays server-side.

## Processes and domains

Four deployables. Everything else is a package behind `raisin-api` or a job on `raisin-worker`.

```mermaid
flowchart LR
  subgraph processes [Processes]
    API[raisin-api]
    Worker[raisin-worker]
    SMTP[raisin-smtp]
    Console[raisin-console]
  end

  subgraph api_surface [API surface]
    Emails[emails]
    Domains[domains / BIMI / claim]
    Audience[contacts / segments / topics]
    Tpl[templates]
    Auto[automations]
    BC[broadcasts]
    WH[webhooks]
    Auth[api-keys / OAuth]
    Bill[billing / usage]
    IPs[ip-pools]
    Supp[suppressions]
  end

  subgraph worker_surface [Worker]
    Jobs[Asynq jobs]
    Pixel[tracking HTTP]
    SQS[SQS SES consumer]
  end

  API --> api_surface
  Worker --> worker_surface
  SMTP -->|same Send path| Emails
  Console -->|Better Auth + proxy| API
```

### API

Go / chi. Auth is `Authorization: Bearer ra_…` plus a `User-Agent`. Rate-limit headers on authenticated responses.

Unauthenticated: `/health`, `/docs`, `/openapi.yaml`, `/console/token`, `/console/provision`, `/inbound/ses`, `/billing/webhook`, `/oauth/token`.

### Worker

One process, three jobs:

1. **Asynq server** — `email:send` (critical), `broadcast:send`, `webhook:deliver`, `domain:verify`, `automation:step`, `ip:warmup-tick` (low). Default concurrency **20**.
2. **Tracking HTTP** — `/t/o/`, `/t/c/`, `/unsubscribe/`.
3. **SQS consumer** — SES events when `SQS_EVENTS_QUEUE_URL` is set.

A daily Asynq scheduler ticks dedicated-IP warmup at midnight UTC.

### SMTP

AUTH PLAIN; API key is the password. Parses a simple MIME body and calls the same `email.Service.Send` as `POST /emails`.

### Console

Next.js. Landing is currently `/` in this app. Better Auth uses the same Postgres as the API (`002_better_auth.sql`).

## Current AWS deploy

What `deploy/terraform` and `deploy/helm/raisin` create if you apply defaults. Images come from GHCR, not ECR.

Defaults: EKS **1.31**, **2× t3.medium**, RDS **db.t4g.medium** (50 GB gp3, single-AZ), ElastiCache **cache.t4g.small**, **1 NAT**, Helm replicas **2 / 2 / 1 / 2** (api / worker / smtp / console).

```mermaid
flowchart TB
  Net((Internet))

  subgraph vpc [VPC 10.20.0.0/16]
    subgraph public [Public subnets]
      ALB[ALB / ingress — api, app, t]
      NLB[NLB — smtp 25/587]
      NAT[NAT Gateway]
    end

    subgraph private [Private subnets]
      subgraph eks [EKS — 2x t3.medium]
        APIp[api x2]
        Wp[worker x2]
        SMTPp[smtp x1]
        CONp[console x2]
      end
      RDS[(RDS Postgres 16)]
      EC[(ElastiCache Redis 7)]
    end
  end

  SES[SES + configuration set]
  SNS[SNS events + inbound]
  SQS[SQS ses-events]
  S3A[(S3 attachments)]
  S3I[(S3 inbound)]
  GHCR[GHCR]

  Net --> ALB
  Net --> NLB
  ALB --> APIp
  ALB --> Wp
  ALB --> CONp
  NLB --> SMTPp
  APIp --> RDS
  APIp --> EC
  Wp --> RDS
  Wp --> EC
  SMTPp --> RDS
  SMTPp --> EC
  CONp --> RDS
  APIp --> S3A
  Wp --> SES
  SES --> SNS
  SNS --> SQS
  SQS --> Wp
  SES --> S3I
  SNS -->|HTTPS /inbound/ses| APIp
  eks --> NAT
  NAT --> GHCR
  NAT --> SES
```

Terraform provisions VPC, EKS, RDS, Redis, S3, SES/SNS/SQS, and the IRSA role. It does **not** provision the ALB, nginx ingress, or SMTP NLB — those appear when you install ingress + the Helm `LoadBalancer` service. Pin a Kubernetes version still in EKS standard support (1.31 is extended as of 2026 and bills 6×).

IAM on the app role: S3 attachments/inbound, SQS receive, SES send + identities + dedicated IP APIs.

## Send path

API and SMTP only enqueue. SES send happens on the worker.

```mermaid
sequenceDiagram
  participant C as Client or SMTP
  participant API as raisin-api
  participant PG as Postgres
  participant Q as Redis Asynq
  participant W as raisin-worker
  participant S3 as S3
  participant SES as Amazon SES
  participant Inbox as Recipient

  C->>API: POST /emails or SMTP DATA
  API->>PG: insert email, check quota
  API->>S3: store attachments
  API->>Q: enqueue email:send
  API-->>C: 200 queued

  W->>Q: dequeue email:send
  W->>PG: load email, suppressions, warmup cap
  W->>SES: SendEmail / SendRawEmail
  SES-->>Inbox: deliver

  SES->>Q: SNS to SQS to worker
  W->>PG: delivery / bounce / complaint
  W->>Q: enqueue webhook:deliver
  W->>C: POST customer webhook
```

Idempotency: `Idempotency-Key` on send. Webhooks: `Raisin-Signature: t=<unix>,v1=<hex>` over `${t}.${rawBody}`.

## Tracking

Same worker process as Asynq. `TRACKING_BASE_URL` (prod: `https://t.raisin.run`) is rewritten into outbound HTML.

```mermaid
sequenceDiagram
  participant Inbox as Recipient
  participant T as t.raisin.run
  participant PG as Postgres
  participant Q as Redis Asynq

  Inbox->>T: GET /t/o or /t/c
  T->>PG: record open or click
  T->>Q: webhook:deliver if configured
  Inbox->>T: GET /unsubscribe
  T->>PG: suppression and topic
```

## Inbound and billing

```mermaid
flowchart LR
  Mail[Inbound mail]
  SES[SES receipt rule]
  S3[(S3 inbound)]
  SNS[SNS inbound]
  API[POST /inbound/ses]
  PG[(Postgres)]
  Stripe[Stripe]
  Hook[POST /billing/webhook]

  Mail --> SES
  SES --> S3
  SES --> SNS
  SNS --> API
  API --> S3
  API --> PG
  Stripe --> Hook
  Hook --> PG
```

Receipt rule stores raw MIME on S3, then SNS hits the API (no API key). Stripe updates plan/quota on the same team row the send path checks.

## Local vs prod

| Concern | Local | Prod |
|---------|-------|------|
| Send | Mailpit (`SENDER_DRIVER=mailpit`) | SES v2 |
| AWS APIs | LocalStack (S3, SQS, SNS, SES) | Real AWS via IRSA |
| Attachments | `STORAGE_DRIVER=local` | S3 |
| Domain verify | Stub — marks verified | Polls SES DKIM / identity |
| Events | Worker send-path + `/test/events/` | SNS → SQS → worker |

Compose applies `001_init.sql`, `002_better_auth.sql`, `003_platform_extras.sql` on first boot.
