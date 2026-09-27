# คำสั่งที่ใช้ทั้งหมด

- ทุกคำสั่งใช้ใน **PowerShell** ได้ (ไม่มี `&&`) และต้องรันใน **โฟลเดอร์โปรเจกต์** (โฟลเดอร์ที่มี `docker-compose.yml`)
- ต้องเปิด **Docker Desktop** ไว้ก่อนเสมอ

## ▶️ เปิด / ปิดระบบ

| ทำอะไร | คำสั่ง |
|---|---|
| **สร้างไฟล์ `.env` (ครั้งแรกครั้งเดียว)** | `copy .env.example .env` |
| เปิดระบบครั้งแรก / หลังแก้โค้ด (build ใหม่) | `docker compose up -d --build` |
| เปิดระบบ (ไม่ build ใหม่) | `docker compose up -d` |
| build ใหม่เฉพาะ service เดียว | `docker compose up -d --build booking-service` |
| หยุดระบบ (ข้อมูลยังอยู่) | `docker compose stop` |
| ปิดและลบ container (ข้อมูลยังอยู่) | `docker compose down` |
| **ล้างทุกอย่างรวม DB** (ข้อมูลหาย) | `docker compose down -v` |

ชื่อ service: `auth-service` (8081) · `service-service` (8082) · `booking-service` (8083) · `notification-service` (8084) · `postgres` (5433 บนเครื่อง)

## 🔍 เช็กสถานะ / ดู log

| ทำอะไร | คำสั่ง |
|---|---|
| ทุก service ขึ้นหรือยัง (ต้องเป็น `Up`) | `docker compose ps` |
| service ตอบสนองไหม (ได้ `{"status":"ok"}`) | `curl.exe http://localhost:8081/health` (เปลี่ยน port 8082–8084 ได้) |
| ดู log ล่าสุด | `docker compose logs booking-service --tail 50` |
| ดู log ต่อเนื่อง (Ctrl+C ออก) | `docker compose logs -f booking-service` |
| ให้ภาษาไทยใน PowerShell ไม่เป็น `????` | `chcp 65001` |

## 🎬 คำสั่งตอน Demo (เรียงตามลำดับ)

**เตรียม**

```powershell
docker compose down -v
docker compose up -d --build
docker compose ps
chcp 65001
```

**0. ภาพรวม (คนที่ 1)**

```powershell
docker compose ps
docker compose exec postgres psql -U postgres -c "\l"
docker compose exec postgres sh -c 'PGPASSWORD=$AUTH_DB_PASSWORD psql -h localhost -U auth_user -d booking_db -l'
```

คำสั่งสุดท้ายต้องขึ้น error สีแดง `permission denied` = ถูกต้อง (แสดงว่า DB แยกกันจริง)

**1. หลังส่วน Auth (คนที่ 1)**

```powershell
docker compose exec postgres psql -U auth_user -d auth_db -c "select id, name, email, role, status, left(password_hash, 20) as password_hash from users"
```

**2. หลังส่วน Service (คนที่ 2)**

```powershell
docker compose exec postgres psql -U service_user -d service_db -c "select * from services" -c "select * from availability"
```

**3. หลังชุด [คิววันนี้] (คนที่ 3)**

```powershell
docker compose exec postgres psql -U booking_user -d booking_db -c "select id, service_id, date, time, status from bookings order by id" -c "select * from queue_status"
```

**4. หลังส่วน Notification & Report (คนที่ 4)**

```powershell
docker compose exec postgres psql -U notification_user -d notification_db -c "select * from notifications" -c "select * from report_logs"
docker compose exec notification-service ls -l /data/reports
```

**หลังจบ**

```powershell
docker compose stop
```

## 🗄️ เปิดดูฐานข้อมูลเอง

| Service | user / database | password (อยู่ในไฟล์ `.env`) |
|---|---|---|
| auth-service | `auth_user` / `auth_db` | `AUTH_DB_PASSWORD` |
| service-service | `service_user` / `service_db` | `SERVICE_DB_PASSWORD` |
| booking-service | `booking_user` / `booking_db` | `BOOKING_DB_PASSWORD` |
| notification-service | `notification_user` / `notification_db` | `NOTIFICATION_DB_PASSWORD` |

| ทำอะไร | คำสั่ง (ตัวอย่าง booking) |
|---|---|
| ดูรายชื่อตาราง | `docker compose exec postgres psql -U booking_user -d booking_db -c "\dt"` |
| ดูโครงสร้างตาราง | `docker compose exec postgres psql -U booking_user -d booking_db -c "\d bookings"` |
| ดูข้อมูล | `docker compose exec postgres psql -U booking_user -d booking_db -c "select * from bookings"` |

ต่อด้วยโปรแกรม (DBeaver / pgAdmin): host `localhost` port `5433` + user ตามตาราง และ password จากไฟล์ `.env`

## 🧪 รันเทสต์อัตโนมัติ (ต้องมี Git Bash)

```powershell
& "C:\Program Files\Git\bin\bash.exe" -c './scripts/test-auth.sh && ./scripts/test-service.sh && ./scripts/test-booking.sh && ./scripts/test-notification.sh'
```

ต้องขึ้น PASS ทั้งหมด ไม่มี FAIL (ถ้าใช้ Git Bash อยู่แล้ว รันส่วนในเครื่องหมาย `'...'` ได้ตรงๆ)

## 🌐 Radmin VPN — ต่อเข้าเครื่องคนที่ 1

ฝั่งเพื่อน (หลัง join network `queue-booking-team-4821` และคนที่ 1 เปิด firewall ให้แล้ว):

```powershell
curl.exe http://26.151.109.184:8081/health
```

ได้ `{"status":"ok"}` = ต่อได้ → ใน Postman แท็บ **Variables** ของ collection เปลี่ยน `localhost` เป็น `26.151.109.184` ทั้ง 4 ตัว

## 🔥 Firewall (เฉพาะเครื่องคนที่ 1 — PowerShell แบบ **Run as administrator**)

| ทำอะไร | คำสั่ง |
|---|---|
| เปิด (ใส่ IP Radmin ของเพื่อน คั่นด้วย comma ไม่มีช่องว่าง) | `New-NetFirewallRule -DisplayName "Queue Booking API (Radmin)" -Direction Inbound -Protocol TCP -LocalPort 8081-8084 -Action Allow -Profile Any -InterfaceAlias "Radmin VPN" -RemoteAddress 26.x.x.x,26.y.y.y,26.z.z.z` |
| เช็กว่าเปิดอยู่ | `Get-NetFirewallRule -DisplayName "Queue Booking API (Radmin)" \| Select-Object DisplayName, Enabled, Action` |
| แก้รายการ IP (ใส่ของทุกคนใหม่ทั้งหมด) | `Set-NetFirewallRule -DisplayName "Queue Booking API (Radmin)" -RemoteAddress 26.x.x.x,26.y.y.y` |
| **ปิด (ลบ rule) — ทำทุกครั้งหลังใช้เสร็จ** | `Remove-NetFirewallRule -DisplayName "Queue Booking API (Radmin)"` |
| ดู IP ของเครื่อง | `ipconfig` |

## 📦 Git / สร้าง zip ใหม่ (เฉพาะคนที่ 1)

| ทำอะไร | คำสั่ง |
|---|---|
| ดูว่าไฟล์ไหนถูกแก้ | `git status` |
| commit แล้วสร้าง zip ใหม่ | `git add -A; git commit -m "แก้อะไรไป"; if ($?) { git archive --format=zip --prefix=queue-booking/ -o ../queue-booking.zip HEAD }` |
| ดูประวัติ | `git log --oneline` |

## 🔧 แก้ปัญหาที่เจอบ่อย

| อาการ | วิธีแก้ |
|---|---|
| `required variable ... is missing a value: copy .env.example to .env` | ยังไม่มีไฟล์ `.env` — รัน `copy .env.example .env` แล้ว up ใหม่ |
| service ขึ้น `Restarting` และ log บอก `password authentication failed` | รหัสผ่าน DB ใน `.env` ถูกแก้หลังสร้าง volume แล้ว — `docker compose down -v` แล้ว `docker compose up -d` |
| `port is already allocated` | แก้เลข **ฝั่งซ้าย** ของ `ports:` ใน `docker-compose.yml` แล้ว `docker compose up -d` และแก้ URL ใน Postman Variables ให้ตรง |
| service ขึ้น `Restarting` | `docker compose logs <ชื่อ service> --tail 30` ดูว่า error อะไร |
| ข้อมูลรก / เทสต์เพี้ยน | `docker compose down -v` แล้ว `docker compose up -d` |
| Docker ไม่ตอบสนอง | เปิด Docker Desktop รอจนขึ้น Running |
| `The token '&&' is not a valid statement separator` | PowerShell 5.1 ไม่รองรับ `&&` — ใช้คำสั่งในไฟล์นี้ (ใช้ `;` แทน) |
| Postman ได้ 401 | token หมดอายุ / เปลี่ยนเครื่องปลายทาง → Run collection ใหม่ตั้งแต่ Register customer |
