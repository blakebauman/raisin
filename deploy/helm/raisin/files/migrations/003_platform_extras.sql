-- Raisin phase-extras: automations, dedicated IPs, OAuth, BIMI/claims, inbound harden, multi-region helpers

-- Domain claiming + BIMI + receiving
ALTER TABLE domains
    ADD COLUMN IF NOT EXISTS claimed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS claim_token TEXT,
    ADD COLUMN IF NOT EXISTS receiving_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS bimi_selector TEXT NOT NULL DEFAULT 'default',
    ADD COLUMN IF NOT EXISTS bimi_svg_url TEXT,
    ADD COLUMN IF NOT EXISTS bimi_vmc_url TEXT,
    ADD COLUMN IF NOT EXISTS ip_pool_id UUID;

CREATE UNIQUE INDEX IF NOT EXISTS domains_claimed_name_uidx
    ON domains (lower(name))
    WHERE claimed_at IS NOT NULL AND status = 'verified';

ALTER TABLE templates
    ADD COLUMN IF NOT EXISTS react_source TEXT,
    ADD COLUMN IF NOT EXISTS editor_json JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE received_emails
    ADD COLUMN IF NOT EXISTS provider_message_id TEXT,
    ADD COLUMN IF NOT EXISTS domain_id UUID REFERENCES domains(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX IF NOT EXISTS received_emails_provider_msg_uidx
    ON received_emails (team_id, provider_message_id)
    WHERE provider_message_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS received_attachments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    received_email_id UUID NOT NULL REFERENCES received_emails(id) ON DELETE CASCADE,
    filename        TEXT NOT NULL,
    content_type    TEXT NOT NULL DEFAULT 'application/octet-stream',
    size_bytes      BIGINT NOT NULL DEFAULT 0,
    storage_key     TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS received_attachments_email_idx ON received_attachments(received_email_id);

-- Dedicated IPs + warmup
CREATE TABLE IF NOT EXISTS ip_pools (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    region          TEXT NOT NULL DEFAULT 'us-east-1',
    status          TEXT NOT NULL DEFAULT 'provisioning', -- provisioning|warming|active|paused|failed
    ses_config_set  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (team_id, name)
);

CREATE TABLE IF NOT EXISTS dedicated_ips (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pool_id         UUID NOT NULL REFERENCES ip_pools(id) ON DELETE CASCADE,
    address         TEXT NOT NULL,
    provider_ref    TEXT,
    status          TEXT NOT NULL DEFAULT 'assigned',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (pool_id, address)
);

CREATE TABLE IF NOT EXISTS warmup_schedules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pool_id         UUID NOT NULL UNIQUE REFERENCES ip_pools(id) ON DELETE CASCADE,
    day_index       INTEGER NOT NULL DEFAULT 0,
    daily_cap       INTEGER NOT NULL DEFAULT 50,
    sent_today      INTEGER NOT NULL DEFAULT 0,
    last_reset_at   TIMESTAMPTZ NOT NULL DEFAULT date_trunc('day', now()),
    plan            JSONB NOT NULL DEFAULT '[50,100,200,400,800,1600,3200,6400,10000,20000]'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

DO $$ BEGIN
    ALTER TABLE domains
        ADD CONSTRAINT domains_ip_pool_fk FOREIGN KEY (ip_pool_id) REFERENCES ip_pools(id) ON DELETE SET NULL;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Automations
CREATE TABLE IF NOT EXISTS automations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    description     TEXT,
    trigger_type    TEXT NOT NULL, -- email.delivered|email.opened|email.clicked|contact.created|email.received
    trigger_filter  JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled         BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS automations_team_trigger_idx ON automations(team_id, trigger_type) WHERE enabled;

CREATE TABLE IF NOT EXISTS automation_steps (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    automation_id   UUID NOT NULL REFERENCES automations(id) ON DELETE CASCADE,
    position        INTEGER NOT NULL,
    step_type       TEXT NOT NULL, -- wait|send_email|add_to_segment|condition
    config          JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (automation_id, position)
);

CREATE TABLE IF NOT EXISTS automation_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    automation_id   UUID NOT NULL REFERENCES automations(id) ON DELETE CASCADE,
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    contact_id      UUID REFERENCES contacts(id) ON DELETE SET NULL,
    email_id        UUID REFERENCES emails(id) ON DELETE SET NULL,
    received_email_id UUID REFERENCES received_emails(id) ON DELETE SET NULL,
    status          TEXT NOT NULL DEFAULT 'running', -- running|completed|canceled|failed
    context         JSONB NOT NULL DEFAULT '{}'::jsonb,
    current_step    INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS automation_runs_auto_idx ON automation_runs(automation_id, status);

CREATE TABLE IF NOT EXISTS automation_run_steps (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id          UUID NOT NULL REFERENCES automation_runs(id) ON DELETE CASCADE,
    step_id         UUID NOT NULL REFERENCES automation_steps(id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'pending', -- pending|running|completed|skipped|failed
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    error           TEXT,
    UNIQUE (run_id, step_id)
);

-- OAuth platform (third-party apps)
CREATE TABLE IF NOT EXISTS oauth_apps (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    client_id       TEXT NOT NULL UNIQUE,
    client_secret_hash TEXT NOT NULL,
    redirect_uris   TEXT[] NOT NULL DEFAULT '{}',
    scopes          TEXT[] NOT NULL DEFAULT ARRAY['emails:send','emails:read','domains:read'],
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS oauth_apps_team_idx ON oauth_apps(team_id);

CREATE TABLE IF NOT EXISTS oauth_authorization_codes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id          UUID NOT NULL REFERENCES oauth_apps(id) ON DELETE CASCADE,
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    code_hash       TEXT NOT NULL UNIQUE,
    redirect_uri    TEXT NOT NULL,
    scopes          TEXT[] NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    used_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS oauth_access_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id          UUID NOT NULL REFERENCES oauth_apps(id) ON DELETE CASCADE,
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    token_hash      TEXT NOT NULL UNIQUE,
    scopes          TEXT[] NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS oauth_refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id          UUID NOT NULL REFERENCES oauth_apps(id) ON DELETE CASCADE,
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    token_hash      TEXT NOT NULL UNIQUE,
    scopes          TEXT[] NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Allowed SES regions for multi-region
CREATE TABLE IF NOT EXISTS team_regions (
    team_id         UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    region          TEXT NOT NULL,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY (team_id, region)
);
