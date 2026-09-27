# Queue & Appointment Booking — Go Microservices

| Service | Port | Database | Paths |
|---|---|---|---|
| auth-service | 8081 | auth_db (`users`) | `/api/auth/*`, `/api/users/*` |
| service-service | 8082 | service_db (`services`, `availability`) | `/api/services/*` |
| booking-service | 8083 | booking_db (`bookings`, `queue_status`) | `/api/bookings/*`, `/api/queue/*` |
| notification-service | 8084 | notification_db (`notifications`, `report_logs`) | `/api/notifications/*`, `/api/reports/*` |

แต่ละ service เป็น Go module แยกกัน: `cmd/<name>/main.go`, `internal/{handler,service,repository,model,middleware,client}`, Gin + GORM/PostgreSQL, มี Dockerfile ของตัวเอง

## รัน

ครั้งแรกให้สร้างไฟล์ `.env` จากตัวอย่างก่อน (รหัสผ่าน DB, `JWT_SECRET`, key ภายใน และบัญชี admin อยู่ในไฟล์นี้ ซึ่งไม่ถูก commit):

```bash
copy .env.example .env
```

(bash / macOS / Linux ใช้ `cp .env.example .env`) ค่าในตัวอย่างใช้รันบนเครื่องได้เลย ถ้าใช้งานจริงให้เปลี่ยน โดยเฉพาะ `JWT_SECRET`

```bash
docker compose up -d --build
```

ทดสอบ (สคริปต์เป็น bash + curl):

Git Bash / Linux / macOS:

```bash
./scripts/test-auth.sh && ./scripts/test-service.sh && ./scripts/test-booking.sh && ./scripts/test-notification.sh
```

Windows PowerShell (5.1 ไม่รองรับ `&&` และ `bash` ใน PATH มักเป็นของ WSL จึงเรียก Git Bash ด้วย path เต็ม):

```powershell
& "C:\Program Files\Git\bin\bash.exe" -c './scripts/test-auth.sh && ./scripts/test-service.sh && ./scripts/test-booking.sh && ./scripts/test-notification.sh'
```

Postman: import `postman/queue-booking.postman_collection.json` แล้ว Run collection ได้ทั้งชุด (token/id ถูกเก็บลง collection variables อัตโนมัติ)

Admin เริ่มต้นถูกสร้างตอน start จาก `ADMIN_EMAIL` / `ADMIN_PASSWORD` ใน `.env`
(ถ้าแก้ค่าใน `.env` ต้องแก้ตัวแปร `admin_email`, `admin_password`, `internal_key` ใน Postman collection ให้ตรงกันด้วย)

## Database: 1 container, 4 databases

ใช้ PostgreSQL **container เดียว** แยก **4 database** (`docker/postgres/init.sh`) แต่ละ DB มี user เป็นเจ้าของเอง
(`auth_user`, `service_user`, ...) รหัสผ่านมาจาก `.env` และ revoke สิทธิ์ CONNECT จาก PUBLIC — service หนึ่งจึงเข้า DB ของ service อื่นไม่ได้
Postgres เปิดที่ host port **5433** · ตารางสร้างด้วย GORM AutoMigrate ตอน service start
(init.sh รันเฉพาะตอนสร้าง volume ครั้งแรก — ถ้าแก้ init.sh หรือรหัสผ่าน DB ใน `.env` ต้อง `docker compose down -v`)

## การตัดสินใจที่ spec ไม่ได้ระบุ

- **การตรวจ token ข้าม service:** service อื่นเรียก `GET /api/users/me` ของ auth-service ทุก request ที่ต้อง login
  → logout (revoke) และการ suspend บัญชีมีผลทันทีทุก service. Denylist ของ token ที่ logout เก็บใน memory ของ auth-service
  (หายเมื่อ restart; token มีอายุ 1 ชม.)
- **Internal calls:** `POST /api/notifications/send` และการที่ notification-service อ่าน `GET /api/bookings/{id}`
  ใช้ header `X-Internal-Key` (env `INTERNAL_API_KEY`)
- **Error body:** `{"error": "..."}` ทุก service; 401 เมื่อไม่มี/token ไม่ถูกต้อง, 503 ถ้า auth-service ติดต่อไม่ได้
- **`GET /api/services?provider_id=`** — query เพิ่มจาก spec (response shape เดิม) ให้ booking-service หา service ของ provider
- **`GET /api/services/{id}/availability`** คืน slot ตามตารางเวลา (แบ่งช่วงตาม `duration_min`) ไม่ได้หัก slot ที่ถูกจองแล้ว;
  booking-service เป็นตัวกันจองซ้ำ (409 + partial unique index `service_id,date,time WHERE status <> 'cancelled'`)
- **สิทธิ์:** แก้/ลบ service และเพิ่ม availability = เจ้าของหรือ admin · ดู/ยกเลิก booking = ลูกค้าเจ้าของ, provider เจ้าของ service, admin ·
  เลื่อนนัด = ลูกค้าเจ้าของหรือ admin · เปลี่ยนสถานะงาน = provider เจ้าของ service หรือ admin
- **สถานะ booking:** ยกเลิก/เลื่อนได้เฉพาะ `pending`; เปลี่ยนสถานะที่ไม่ถูกลำดับ → 409
- **Queue:** `waiting` = booking `pending` ของวันนี้ (Asia/Bangkok) เรียงตามเวลา; `current_serving` = booking ล่าสุดที่ถูกตั้ง `in_progress` วันนี้ (ไม่มี = `null`)
  เป็น REST ให้ client poll (ไม่มี WebSocket)
- **Notification email:** ยังไม่มี mail provider — บันทึกลง DB และเขียน log แทนการส่งจริง; ไม่มี scheduler ในตัว (ให้ cron/ระบบภายนอกเรียก `/send`)
- **Reports:** ดึงข้อมูลจาก booking-service ด้วย token ของ admin ที่เรียก แล้วคำนวณ; ทุกครั้งบันทึก `report_logs`.
  `cancellation-rate` ไม่ใส่ `service_id` → `{"services":[{"service_id","cancellation_rate"}...]}`.
  `export` บันทึกไฟล์ที่ `/data/reports` (volume `reports`) และส่งกลับเป็น `report.csv`
- ทุก service เก็บ `TIMESTAMP` เป็น UTC และตอบเวลาเป็น RFC3339 (`...Z`)
