#!/bin/sh
# Runs once, when the postgres container initializes an EMPTY data directory
# (mounted at /docker-entrypoint-initdb.d). Creates the two roles Netopology uses:
#
#   netopology_owner  owns the schema, runs migrations (`netopology migrate`). BYPASSRLS so that
#                     SECURITY DEFINER maintenance functions (e.g. prune_workspace_events) work.
#                     The api/worker NEVER connect as this role.
#   netopology_app    what api and worker connect as. NOT superuser, NOBYPASSRLS: row-level
#                     security applies to it (docs/design/data-model.md §6).
#
# Passwords come from the container environment (see compose files / .env.example).
set -eu

: "${NETOPOLOGY_DB_OWNER_PASSWORD:?set NETOPOLOGY_DB_OWNER_PASSWORD}"
: "${NETOPOLOGY_DB_APP_PASSWORD:?set NETOPOLOGY_DB_APP_PASSWORD}"
DB="${NETOPOLOGY_DB_NAME:-netopology}"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres \
  -v owner_pw="$NETOPOLOGY_DB_OWNER_PASSWORD" -v app_pw="$NETOPOLOGY_DB_APP_PASSWORD" -v db="$DB" <<'SQL'
CREATE ROLE netopology_owner LOGIN NOSUPERUSER NOCREATEROLE BYPASSRLS PASSWORD :'owner_pw';
CREATE ROLE netopology_app   LOGIN NOSUPERUSER NOCREATEROLE NOBYPASSRLS PASSWORD :'app_pw';
SELECT format('CREATE DATABASE %I OWNER netopology_owner', :'db') \gexec
SQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$DB" <<'SQL'
REVOKE ALL ON SCHEMA public FROM PUBLIC;
ALTER SCHEMA public OWNER TO netopology_owner;
GRANT USAGE ON SCHEMA public TO netopology_app;
-- everything the owner creates later (migrations, River tables) is usable by the app role
ALTER DEFAULT PRIVILEGES FOR ROLE netopology_owner IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO netopology_app;
ALTER DEFAULT PRIVILEGES FOR ROLE netopology_owner IN SCHEMA public
  GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO netopology_app;
ALTER DEFAULT PRIVILEGES FOR ROLE netopology_owner IN SCHEMA public
  GRANT EXECUTE ON FUNCTIONS TO netopology_app;
SQL
