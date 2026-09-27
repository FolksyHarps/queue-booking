#!/usr/bin/env bash
# Smoke test for notification-service (needs the whole stack running).
set -u
source "$(dirname "$0")/lib.sh"
# internal-key requests
ireq() {
  local m=$1 url=$2 want=$3 key=$4 data=$5
  local code; code=$(printf '%s' "$data" | curl -s -o /tmp/body -w '%{http_code}' -X "$m" "$url" -H "X-Internal-Key: $key" -H 'Content-Type: application/json' --data-binary @-)
  BODY=$(cat /tmp/body)
  if [ "$code" = "$want" ]; then echo "PASS $m $url -> $code $BODY"; else echo "FAIL $m $url -> got $code want $want $BODY"; FAIL=1; fi
}
new_user provider; PROV=$TOKEN
new_user customer; CUST=$TOKEN
new_user customer; CUST2=$TOKEN
admin_login
req POST $SVC/api/services 201 "$PROV" '{"name":"นวดแผนไทย","duration_min":60,"price":300,"category":"massage"}'
SID=$(jsonval id)
TOMORROW=$(date -u -d '+31 hour' +%F); DOW=$(date -u -d '+31 hour' +%w)
req POST $SVC/api/services/$SID/availability 201 "$PROV" "{\"day_of_week\":$DOW,\"start_time\":\"09:00\",\"end_time\":\"17:00\"}"
req POST $BOOK/api/bookings 201 "$CUST" "{\"service_id\":$SID,\"date\":\"$TOMORROW\",\"time\":\"09:00\"}"; B1=$(jsonval id)
req POST $BOOK/api/bookings 201 "$CUST" "{\"service_id\":$SID,\"date\":\"$TOMORROW\",\"time\":\"10:00\"}"; B2=$(jsonval id)
req POST $BOOK/api/bookings 201 "$CUST2" "{\"service_id\":$SID,\"date\":\"$TOMORROW\",\"time\":\"11:00\"}"; B3=$(jsonval id)
req PUT $BOOK/api/bookings/$B2/cancel 200 "$CUST"
req PUT $BOOK/api/bookings/$B3/status 200 "$PROV" '{"status":"completed"}'

ireq POST $NOTI/api/notifications/send 401 wrong-key "{\"booking_id\":$B1,\"channel\":\"email\"}"
ireq POST $NOTI/api/notifications/send 400 "$INTERNAL_KEY" "{\"booking_id\":$B1,\"channel\":\"sms\"}"
ireq POST $NOTI/api/notifications/send 400 "$INTERNAL_KEY" '{"booking_id":999999,"channel":"email"}'
ireq POST $NOTI/api/notifications/send 400 "$INTERNAL_KEY" "{\"booking_id\":$B2,\"channel\":\"email\"}"
ireq POST $NOTI/api/notifications/send 201 "$INTERNAL_KEY" "{\"booking_id\":$B1,\"channel\":\"email\"}"
ireq POST $NOTI/api/notifications/send 201 "$INTERNAL_KEY" "{\"booking_id\":$B1,\"channel\":\"in-app\"}"
NID=$(jsonval id)

req GET $NOTI/api/notifications 401
req GET $NOTI/api/notifications 200 "$CUST"
req GET "$NOTI/api/notifications?read=maybe" 400 "$CUST"
req PUT $NOTI/api/notifications/$NID/read 404 "$CUST2"
req PUT $NOTI/api/notifications/$NID/read 200 "$CUST"
req PUT $NOTI/api/notifications/999999/read 404 "$CUST"
req GET "$NOTI/api/notifications?read=false" 200 "$CUST"

req GET "$NOTI/api/reports/summary" 403 "$CUST"
req GET "$NOTI/api/reports/summary?start_date=$TOMORROW&end_date=$TOMORROW" 200 "$ADMIN"
req GET "$NOTI/api/reports/summary?start_date=bad" 400 "$ADMIN"
req GET "$NOTI/api/reports/cancellation-rate?service_id=$SID" 200 "$ADMIN"
req GET "$NOTI/api/reports/cancellation-rate" 200 "$ADMIN"
req GET "$NOTI/api/reports/cancellation-rate" 403 "$PROV"
req GET "$NOTI/api/reports/export?type=bookings&start_date=$TOMORROW&end_date=$TOMORROW" 200 "$ADMIN"
req GET "$NOTI/api/reports/export?type=cancellations" 200 "$ADMIN"
req GET "$NOTI/api/reports/export?type=other" 400 "$ADMIN"
req GET "$NOTI/api/reports/export?type=bookings" 403 "$CUST"
curl -s -D - -o /dev/null -H "Authorization: Bearer $ADMIN" "$NOTI/api/reports/export?type=bookings" | grep -i -E '^content-(type|disposition)'
exit $FAIL
