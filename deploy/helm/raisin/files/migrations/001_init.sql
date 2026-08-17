-- Raisin initial schema
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Identity
CREATE TABLE teams (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    test_mode       BOOLEAN NOT NULL DEFAULT FALSE,
    monthly_quota   INTEGER NOT NULL DEFAULT 3000,
    stripe_customer_id TEXT,
    stripe_subscription_id TEXT,
    billing_status  TEXT NOT NULL DEFAULT 'free', -- free|active|past_due|canceled|paused
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT NOT NULL UNIQUE,
    name            TEXT,
    image           TEXT,
    email_verified  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE members (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role            TEXT NOT NULL DEFAULT 'member', -- owner|admin|member
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (team_id, user_id)
);

CREATE TABLE api_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    prefix          TEXT NOT NULL, -- ra_xxxx shown
    key_hash        TEXT NOT NULL UNIQUE,
    permission      TEXT NOT NULL DEFAULT 'full_access', -- full_access|sending_access
    last_used_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX api_keys_team_idx ON api_keys(team_id);

-- Domains
CREATE TABLE domains (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    region          TEXT NOT NULL DEFAULT 'us-east-1',
    status          TEXT NOT NULL DEFAULT 'not_started', -- not_started|pending|verified|failed
    open_tracking   BOOLEAN NOT NULL DEFAULT TRUE,
    click_tracking  BOOLEAN NOT NULL DEFAULT TRUE,
    tracking_subdomain TEXT NOT NULL DEFAULT 'links',
    ses_identity_arn TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (team_id, name)
);

CREATE TABLE domain_records (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id       UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    record_type     TEXT NOT NULL, -- SPF|DKIM|Tracking|DMARC
    name            TEXT NOT NULL,
    type            TEXT NOT NULL, -- TXT|CNAME|MX
    value           TEXT NOT NULL,
    priority        INTEGER,
    status          TEXT NOT NULL DEFAULT 'not_started',
    ttl             TEXT NOT NULL DEFAULT 'Auto'
);
CREATE INDEX domain_records_domain_idx ON domain_records(domain_id);

-- Emails
CREATE TABLE emails (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    domain_id       UUID REFERENCES domains(id),
    from_addr       TEXT NOT NULL,
    to_addrs        TEXT[] NOT NULL,
    cc_addrs        TEXT[] NOT NULL DEFAULT '{}',
    bcc_addrs       TEXT[] NOT NULL DEFAULT '{}',
    reply_to        TEXT[] NOT NULL DEFAULT '{}',
    subject         TEXT NOT NULL DEFAULT '',
    html            TEXT,
    text            TEXT,
    headers         JSONB NOT NULL DEFAULT '{}',
    tags            JSONB NOT NULL DEFAULT '{}',
    status          TEXT NOT NULL DEFAULT 'queued', -- queued|scheduled|sent|delivered|bounced|complained|failed|canceled|suppressed
    provider_message_id TEXT,
    template_id     UUID,
    broadcast_id    UUID,
    scheduled_at    TIMESTAMPTZ,
    sent_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX emails_team_created_idx ON emails(team_id, created_at DESC);
CREATE INDEX emails_status_idx ON emails(status) WHERE status IN ('queued', 'scheduled');
CREATE INDEX emails_provider_idx ON emails(provider_message_id) WHERE provider_message_id IS NOT NULL;

CREATE TABLE email_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    email_id        UUID NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    type            TEXT NOT NULL, -- email.sent|email.delivered|email.bounced|...
    data            JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX email_events_email_idx ON email_events(email_id, created_at);

CREATE TABLE attachments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    email_id        UUID NOT NULL REFERENCES emails(id) ON DELETE CASCADE,
    filename        TEXT NOT NULL,
    content_type    TEXT NOT NULL DEFAULT 'application/octet-stream',
    size_bytes      BIGINT NOT NULL DEFAULT 0,
    s3_key          TEXT NOT NULL,
    content_id      TEXT, -- for inline
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE idempotency_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    key             TEXT NOT NULL,
    email_id        UUID REFERENCES emails(id) ON DELETE SET NULL,
    response_body   JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '24 hours'),
    UNIQUE (team_id, key)
);

-- Suppressions
CREATE TABLE suppressions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    email           TEXT NOT NULL,
    reason          TEXT NOT NULL, -- bounce|complaint|manual
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (team_id, email)
);

-- Audience
CREATE TABLE contacts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    email           TEXT NOT NULL,
    first_name      TEXT,
    last_name       TEXT,
    unsubscribed    BOOLEAN NOT NULL DEFAULT FALSE,
    properties      JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (team_id, email)
);

CREATE TABLE contact_properties (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    key             TEXT NOT NULL,
    type            TEXT NOT NULL DEFAULT 'string', -- string|number
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (team_id, key)
);

CREATE TABLE segments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE segment_members (
    segment_id      UUID NOT NULL REFERENCES segments(id) ON DELETE CASCADE,
    contact_id      UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (segment_id, contact_id)
);

CREATE TABLE topics (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    description     TEXT,
    default_subscription TEXT NOT NULL DEFAULT 'opt_in', -- opt_in|opt_out
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE topic_subscriptions (
    topic_id        UUID NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    contact_id      UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    subscribed      BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (topic_id, contact_id)
);

-- Templates & broadcasts
CREATE TABLE templates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    subject         TEXT,
    html            TEXT,
    text            TEXT,
    variables       JSONB NOT NULL DEFAULT '[]',
    status          TEXT NOT NULL DEFAULT 'draft', -- draft|published
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE broadcasts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    segment_id      UUID REFERENCES segments(id),
    name            TEXT,
    from_addr       TEXT NOT NULL,
    subject         TEXT NOT NULL,
    html            TEXT,
    text            TEXT,
    template_id     UUID REFERENCES templates(id),
    status          TEXT NOT NULL DEFAULT 'draft', -- draft|queued|sending|sent|canceled
    scheduled_at    TIMESTAMPTZ,
    sent_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Webhooks
CREATE TABLE webhooks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    endpoint        TEXT NOT NULL,
    events          TEXT[] NOT NULL,
    secret          TEXT NOT NULL,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE webhook_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    webhook_id      UUID NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    type            TEXT NOT NULL,
    payload         JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE webhook_attempts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_event_id UUID NOT NULL REFERENCES webhook_events(id) ON DELETE CASCADE,
    status_code     INTEGER,
    success         BOOLEAN NOT NULL DEFAULT FALSE,
    error           TEXT,
    attempted_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- API logs & usage
CREATE TABLE api_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    method          TEXT NOT NULL,
    path            TEXT NOT NULL,
    status_code     INTEGER NOT NULL,
    request_id      TEXT,
    duration_ms     INTEGER,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX api_logs_team_created_idx ON api_logs(team_id, created_at DESC);

CREATE TABLE usage_periods (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    period_start    DATE NOT NULL,
    period_end      DATE NOT NULL,
    emails_sent     INTEGER NOT NULL DEFAULT 0,
    UNIQUE (team_id, period_start)
);

-- Inbound emails (phase 4)
CREATE TABLE received_emails (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    domain_id       UUID REFERENCES domains(id),
    from_addr       TEXT NOT NULL,
    to_addrs        TEXT[] NOT NULL,
    subject         TEXT,
    html            TEXT,
    text            TEXT,
    s3_key          TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed a demo team for local development
INSERT INTO teams (id, name, slug, test_mode, monthly_quota)
VALUES ('00000000-0000-0000-0000-000000000001', 'Demo Team', 'demo', TRUE, 10000);

INSERT INTO users (id, email, name, email_verified)
VALUES ('00000000-0000-0000-0000-000000000002', 'demo@raisin.run', 'Demo User', TRUE);

INSERT INTO members (team_id, user_id, role)
VALUES ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000002', 'owner');

-- Demo API key: ra_demo_00000000000000000000000000000000
-- SHA256 of that string (hex)
INSERT INTO api_keys (team_id, name, prefix, key_hash, permission)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Demo Key',
    'ra_demo',
    encode(digest('ra_demo_00000000000000000000000000000000', 'sha256'), 'hex'),
    'full_access'
);
