#!/usr/bin/env bash
# Smoke test for auth-service. Usage: AUTH=http://localhost:8081 ./scripts/test-auth.sh
set -u
source "$(dirname "$0")/lib.sh"
EMAIL="u$RANDOM$RANDOM@example.com"
req POST $AUTH/api/auth/register 201 "" "{\"name\":\"สมชาย ใจดี\",\"email\":\"$EMAIL\",\"password\":\"secret123\",\"role\":\"customer\"}"
UID_=$(jsonval id)
req POST $AUTH/api/auth/register 409 "" "{\"name\":\"x\",\"email\":\"$EMAIL\",\"password\":\"secret123\",\"role\":\"customer\"}"
req POST $AUTH/api/auth/register 400 "" "{\"name\":\"x\",\"email\":\"bad\",\"password\":\"secret123\",\"role\":\"admin\"}"
req POST $AUTH/api/auth/login 401 "" "{\"email\":\"$EMAIL\",\"password\":\"wrong-pass\"}"
req POST $AUTH/api/auth/login 200 "" "{\"email\":\"$EMAIL\",\"password\":\"secret123\"}"
TOK=$(jsonval token)
req GET $AUTH/api/users/me 401
req GET $AUTH/api/users/me 200 "$TOK"
req PUT $AUTH/api/users/me 200 "$TOK" '{"name":"สมชาย ใจดีมาก","phone":"0812345678"}'
req PUT $AUTH/api/users/me 400 "$TOK" '{}'
req GET "$AUTH/api/users" 403 "$TOK"
admin_login
ADM=$ADMIN
req GET "$AUTH/api/users?role=customer&page=1&limit=5" 200 "$ADM"
req PUT $AUTH/api/users/$UID_/status 200 "$ADM" '{"status":"suspended"}'
req GET $AUTH/api/users/me 401 "$TOK"
req POST $AUTH/api/auth/login 401 "" "{\"email\":\"$EMAIL\",\"password\":\"secret123\"}"
req PUT $AUTH/api/users/$UID_/status 200 "$ADM" '{"status":"active"}'
req PUT $AUTH/api/users/999999/status 404 "$ADM" '{"status":"active"}'
req PUT $AUTH/api/users/$UID_/status 403 "$TOK" '{"status":"active"}'
req POST $AUTH/api/auth/logout 200 "$TOK"
req GET $AUTH/api/users/me 401 "$TOK"
exit $FAIL
