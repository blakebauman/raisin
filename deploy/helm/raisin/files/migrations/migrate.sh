#!/usr/bin/env bash
# Apply Raisin SQL migrations once each, tracked in schema_migrations.
# Usage: DATABASE_URL=... ./scripts/migrate.sh [migrations_dir]
set -euo pipefail

DIR="${1:-$(cd "$(dirname "$0")/.." && pwd)/migrations}"
: "${DATABASE_URL:?DATABASE_URL is required}"

psql_q() {
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 "$@"
}

psql_q -c '
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
'

# Existing DBs applied SQL before version tracking: stamp matching files, skip re-run.
count="$(psql_q -tAc 'SELECT count(*) FROM schema_migrations' | tr -d '[:space:]')"
if [[ "$count" == "0" ]]; then
  has_teams="$(psql_q -tAc "SELECT CASE WHEN to_regclass('public.teams') IS NULL THEN 0 ELSE 1 END" | tr -d '[:space:]')"
  if [[ "$has_teams" == "1" ]]; then
    echo "Bootstrapping schema_migrations for existing database…"
    stamp() {
      local v="$1"
      [[ -f "$DIR/$v" ]] || return 0
      psql_q -c "INSERT INTO schema_migrations(version) VALUES ('$v') ON CONFLICT DO NOTHING"
    }
    stamp "001_init.sql"
    has_ba="$(psql_q -tAc 'SELECT CASE WHEN to_regclass('\''public.user'\'') IS NULL THEN 0 ELSE 1 END' | tr -d '[:space:]')"
    [[ "$has_ba" == "1" ]] && stamp "002_better_auth.sql"
    has_auto="$(psql_q -tAc "SELECT CASE WHEN to_regclass('public.automations') IS NULL THEN 0 ELSE 1 END" | tr -d '[:space:]')"
    [[ "$has_auto" == "1" ]] && stamp "003_platform_extras.sql"
  fi
fi

shopt -s nullglob
files=("$DIR"/*.sql)
if [[ ${#files[@]} -eq 0 ]]; then
  echo "No migrations in $DIR"
  exit 1
fi

# Lexicographic order matches 001_, 002_, …
IFS=$'\n' sorted=($(printf '%s\n' "${files[@]}" | sort))
unset IFS

for f in "${sorted[@]}"; do
  ver="$(basename "$f")"
  applied="$(psql_q -tAc "SELECT 1 FROM schema_migrations WHERE version='$ver'" | tr -d '[:space:]')"
  if [[ "$applied" == "1" ]]; then
    echo "skip $ver"
    continue
  fi
  echo "==> $ver"
  psql_q -f "$f"
  psql_q -c "INSERT INTO schema_migrations(version) VALUES ('$ver')"
done

echo "Migrations complete."
