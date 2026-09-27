#!/bin/sh
# One PostgreSQL container, four isolated databases (database-per-service).
# Each service gets its own login role that owns only its own database.
# Passwords come from the container environment (set in .env); runs only when
# the data volume is first created.
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres \
  -v auth_pw="$AUTH_DB_PASSWORD" \
  -v service_pw="$SERVICE_DB_PASSWORD" \
  -v booking_pw="$BOOKING_DB_PASSWORD" \
  -v notification_pw="$NOTIFICATION_DB_PASSWORD" <<'EOSQL'
CREATE USER auth_user WITH PASSWORD :'auth_pw';
CREATE DATABASE auth_db OWNER auth_user;
REVOKE ALL ON DATABASE auth_db FROM PUBLIC;

CREATE USER service_user WITH PASSWORD :'service_pw';
CREATE DATABASE service_db OWNER service_user;
REVOKE ALL ON DATABASE service_db FROM PUBLIC;

CREATE USER booking_user WITH PASSWORD :'booking_pw';
CREATE DATABASE booking_db OWNER booking_user;
REVOKE ALL ON DATABASE booking_db FROM PUBLIC;

CREATE USER notification_user WITH PASSWORD :'notification_pw';
CREATE DATABASE notification_db OWNER notification_user;
REVOKE ALL ON DATABASE notification_db FROM PUBLIC;
EOSQL
