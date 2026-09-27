#!/usr/bin/env bash
# Smoke test for service-service (needs auth-service running).
set -u
source "$(dirname "$0")/lib.sh"
new_user provider; PROV=$TOKEN; PROV_ID=$USER_ID
new_user provider; OTHER=$TOKEN
new_user customer; CUST=$TOKEN

req POST $SVC/api/services 401 "" '{"name":"x","duration_min":30,"price":1,"category":"c"}'
req POST $SVC/api/services 403 "$CUST" '{"name":"x","duration_min":30,"price":1,"category":"c"}'
req POST $SVC/api/services 400 "$PROV" '{"name":"x","duration_min":0,"category":"c"}'
req POST $SVC/api/services 201 "$PROV" '{"name":"นวดแผนไทย","duration_min":60,"price":300,"category":"massage"}'
SID=$(jsonval id)
req GET "$SVC/api/services?category=massage&keyword=%E0%B8%99%E0%B8%A7%E0%B8%94" 200
req GET $SVC/api/services/$SID 200
req GET $SVC/api/services/999999 404
req PUT $SVC/api/services/$SID 403 "$OTHER" '{"price":320}'
req PUT $SVC/api/services/$SID 200 "$PROV" '{"name":"นวดแผนไทย (อัปเดต)","price":320}'
req PUT $SVC/api/services/999999 404 "$PROV" '{"price":320}'

DATE=$(date -d tomorrow +%F); DOW=$(date -d tomorrow +%w)
req POST $SVC/api/services/$SID/availability 400 "$PROV" "{\"day_of_week\":$DOW,\"start_time\":\"17:00\",\"end_time\":\"09:00\"}"
req POST $SVC/api/services/$SID/availability 403 "$OTHER" "{\"day_of_week\":$DOW,\"start_time\":\"09:00\",\"end_time\":\"12:00\"}"
req POST $SVC/api/services/$SID/availability 201 "$PROV" "{\"day_of_week\":$DOW,\"start_time\":\"09:00\",\"end_time\":\"12:00\"}"
req GET "$SVC/api/services/$SID/availability?date=$DATE" 200
req GET "$SVC/api/services/999999/availability?date=$DATE" 404

req DELETE $SVC/api/services/$SID 403 "$OTHER"
req DELETE $SVC/api/services/$SID 204 "$PROV"
req DELETE $SVC/api/services/$SID 404 "$PROV"
exit $FAIL
