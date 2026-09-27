# Shared helpers for the smoke tests (source this file).
# Admin credentials and the internal key come from the project's .env.
ENV_FILE="$(dirname "${BASH_SOURCE[0]}")/../.env"
if [ ! -f "$ENV_FILE" ]; then
  echo "missing .env - copy .env.example to .env first" >&2
  exit 1
fi
set -a; . <(tr -d '\r' < "$ENV_FILE"); set +a
AUTH=${AUTH:-http://localhost:8081}
FAIL=0
# req METHOD URL EXPECTED_CODE [TOKEN] [JSON_BODY]  -> prints body, sets BODY
req() {
  local m=$1 url=$2 want=$3 tok=${4:-} data=${5:-}
  local args=(-s -o /tmp/body -w '%{http_code}' -X "$m" "$url")
  [ -n "$tok" ] && args+=(-H "Authorization: Bearer $tok")
  [ -n "$data" ] && args+=(-H 'Content-Type: application/json' --data-binary @-)
  local code; code=$(printf '%s' "$data" | curl "${args[@]}")
  BODY=$(cat /tmp/body)
  if [ "$code" = "$want" ]; then echo "PASS $m $url -> $code $BODY"; else echo "FAIL $m $url -> got $code want $want $BODY"; FAIL=1; fi
}
jsonval() { sed -n "s/.*\"$1\":\"\{0,1\}\([^\",}]*\).*/\1/p" <<<"$BODY"; }

SVC=${SVC:-http://localhost:8082}
BOOK=${BOOK:-http://localhost:8083}
NOTI=${NOTI:-http://localhost:8084}
INTERNAL_KEY=$INTERNAL_API_KEY

# new_user ROLE -> registers + logs in a fresh user; sets TOKEN and USER_ID
new_user() {
  local email="$1$RANDOM$RANDOM@example.com"
  req POST $AUTH/api/auth/register 201 "" "{\"name\":\"test $1\",\"email\":\"$email\",\"password\":\"secret123\",\"role\":\"$1\"}"
  USER_ID=$(jsonval id)
  req POST $AUTH/api/auth/login 200 "" "{\"email\":\"$email\",\"password\":\"secret123\"}"
  TOKEN=$(jsonval token)
}
admin_login() {
  req POST $AUTH/api/auth/login 200 "" "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}"
  ADMIN=$(jsonval token)
}
