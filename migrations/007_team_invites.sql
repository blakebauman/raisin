-- Team invites for multi-user console access
CREATE TABLE IF NOT EXISTS team_invites (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id      UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    email        TEXT NOT NULL,
    role         TEXT NOT NULL DEFAULT 'member', -- admin|member
    token_hash   TEXT NOT NULL UNIQUE,
    invited_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    expires_at   TIMESTAMPTZ NOT NULL,
    accepted_at  TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT team_invites_role_check CHECK (role IN ('admin', 'member'))
);

CREATE INDEX IF NOT EXISTS team_invites_team_idx ON team_invites(team_id);
CREATE INDEX IF NOT EXISTS team_invites_email_idx ON team_invites(email);

-- One open invite per email per team
CREATE UNIQUE INDEX IF NOT EXISTS team_invites_pending_email_uidx
    ON team_invites (team_id, email)
    WHERE accepted_at IS NULL AND revoked_at IS NULL;
