#!/usr/bin/env bash
# Smoke test for booking-service (needs auth-service + service-service running).
set -u
source "$(dirname "$0")/lib.sh"
new_user provider; PROV=$TOKEN
new_user provider; OTHERPROV=$TOKEN
new_user customer; CUST=$TOKEN; CUST_ID=$USER_ID
new_user customer; CUST2=$TOKEN

req POST $SVC/api/services 201 "$PROV" '{"name":"ตัดผมชาย","duration_min":30,"price":150,"category":"hair"}'
SID=$(jsonval id)
# Bangkok (UTC+7) tomorrow
TOMORROW=$(date -u -d '+31 hour' +%F); TOMORROW_DOW=$(date -u -d '+31 hour' +%w)
req POST $SVC/api/services/$SID/availability 201 "$PROV" "{\"day_of_week\":$TOMORROW_DOW,\"start_time\":\"09:00\",\"end_time\":\"12:00\"}"

req POST $BOOK/api/bookings 401 "" "{\"service_id\":$SID,\"date\":\"$TOMORROW\",\"time\":\"10:00\"}"
req POST $BOOK/api/bookings 400 "$CUST" "{\"service_id\":$SID,\"date\":\"$TOMORROW\",\"time\":\"13:00\"}"
req POST $BOOK/api/bookings 400 "$CUST" "{\"service_id\":999999,\"date\":\"$TOMORROW\",\"time\":\"10:00\"}"
req POST $BOOK/api/bookings 400 "$CUST" "{\"service_id\":$SID,\"date\":\"2020-01-01\",\"time\":\"10:00\"}"
req POST $BOOK/api/bookings 201 "$CUST" "{\"service_id\":$SID,\"date\":\"$TOMORROW\",\"time\":\"10:00\"}"
B1=$(jsonval id)
req POST $BOOK/api/bookings 409 "$CUST2" "{\"service_id\":$SID,\"date\":\"$TOMORROW\",\"time\":\"10:00\"}"
req POST $BOOK/api/bookings 201 "$CUST2" "{\"service_id\":$SID,\"date\":\"$TOMORROW\",\"time\":\"10:30\"}"
B2=$(jsonval id)

req GET "$BOOK/api/bookings?status=pending&page=1&limit=10" 200 "$CUST"
req GET "$BOOK/api/bookings" 200 "$PROV"
req GET "$BOOK/api/bookings" 200 "$OTHERPROV"
req GET "$BOOK/api/bookings" 401
req GET $BOOK/api/bookings/$B1 200 "$CUST"
req GET $BOOK/api/bookings/$B1 200 "$PROV"
req GET $BOOK/api/bookings/$B1 403 "$CUST2"
req GET $BOOK/api/bookings/$B1 403 "$OTHERPROV"
req GET $BOOK/api/bookings/999999 404 "$CUST"

req PUT $BOOK/api/bookings/$B1/reschedule 409 "$CUST" "{\"new_date\":\"$TOMORROW\",\"new_time\":\"10:30\"}"
req PUT $BOOK/api/bookings/$B1/reschedule 403 "$CUST2" "{\"new_date\":\"$TOMORROW\",\"new_time\":\"11:00\"}"
req PUT $BOOK/api/bookings/999999/reschedule 404 "$CUST" "{\"new_date\":\"$TOMORROW\",\"new_time\":\"11:00\"}"
req PUT $BOOK/api/bookings/$B1/reschedule 200 "$CUST" "{\"new_date\":\"$TOMORROW\",\"new_time\":\"11:00\"}"

req PUT $BOOK/api/bookings/$B2/cancel 403 "$CUST"
req PUT $BOOK/api/bookings/999999/cancel 404 "$CUST2"
req PUT $BOOK/api/bookings/$B2/cancel 200 "$CUST2"
# cancelled slot can be booked again
req POST $BOOK/api/bookings 201 "$CUST" "{\"service_id\":$SID,\"date\":\"$TOMORROW\",\"time\":\"10:30\"}"

req PUT $BOOK/api/bookings/$B1/status 403 "$CUST" '{"status":"completed"}'
req PUT $BOOK/api/bookings/$B1/status 403 "$OTHERPROV" '{"status":"completed"}'
req PUT $BOOK/api/bookings/$B1/status 200 "$PROV" '{"status":"completed"}'

# today's realtime queue: a 1-minute service open all day always has a slot a
# few minutes ahead, except just before midnight, when we wait for the new day.
SECS_TO_MIDNIGHT=$(( 86400 - $(date -u -d '+7 hour' +%s) % 86400 ))
if [ "$SECS_TO_MIDNIGHT" -le 300 ]; then
  echo "waiting ${SECS_TO_MIDNIGHT}s for midnight (Asia/Bangkok) so today's slots are in the future..."
  sleep $((SECS_TO_MIDNIGHT + 5))
fi
TODAY=$(date -u -d '+7 hour' +%F); TODAY_DOW=$(date -u -d '+7 hour' +%w)
Q1=$(date -u -d '+7 hour +2 minutes' +%H:%M); Q2=$(date -u -d '+7 hour +3 minutes' +%H:%M)
req POST $SVC/api/services 201 "$PROV" '{"name":"คิวทดสอบ","duration_min":1,"price":0,"category":"test"}'
QSID=$(jsonval id)
req POST $SVC/api/services/$QSID/availability 201 "$PROV" "{\"day_of_week\":$TODAY_DOW,\"start_time\":\"00:00\",\"end_time\":\"23:59\"}"
req POST $BOOK/api/bookings 201 "$CUST" "{\"service_id\":$QSID,\"date\":\"$TODAY\",\"time\":\"$Q1\"}"
T1=$(jsonval id)
req POST $BOOK/api/bookings 201 "$CUST2" "{\"service_id\":$QSID,\"date\":\"$TODAY\",\"time\":\"$Q2\"}"
T2=$(jsonval id)
req GET $BOOK/api/queue/$QSID 200
[ "$BODY" = "{\"service_id\":$QSID,\"current_serving\":null,\"waiting\":[$T1,$T2]}" ] || { echo "FAIL queue before serving: $BODY"; FAIL=1; }
req PUT $BOOK/api/bookings/$T1/status 200 "$PROV" '{"status":"in_progress"}'
req GET $BOOK/api/queue/$QSID 200
[ "$BODY" = "{\"service_id\":$QSID,\"current_serving\":$T1,\"waiting\":[$T2]}" ] || { echo "FAIL queue while serving: $BODY"; FAIL=1; }
req GET $BOOK/api/queue/999999 404
exit $FAIL
