# Goals API (Go + Echo + GORM + MySQL)

Backend REST API untuk aplikasi Goals (Couple Finance & Wedding Planner).
Dummy data awal: **Fajar Nugroho** & **Raisa Amelia** (password: `demo123`).

## Fitur utama
- Echo v4 sebagai HTTP framework, GORM sebagai ORM ke MySQL.
- Seluruh request & response body **dienkripsi AES-256-GCM** secara transparan
  lewat middleware (`middleware/encryption.go`), jadi komunikasi client ↔ API aman.
- JWT auth sederhana (`Authorization: Bearer <token>`).
- Endpoint backup penuh (`/backup/export`, `/backup/import`) + status pengingat
  backup tiap 28 hari (`/backup/status`).
- Semua modul frontend punya endpoint: wallets, budgets, transactions,
  cashflows, wedding planner, plans ("Rencana Kita"), goals.

## Menjalankan

1. Copy env:
   ```bash
   cp .env.example .env
   ```
2. Jalankan MySQL (via docker-compose atau instance lokal):
   ```bash
   docker compose up -d
   ```
3. Install dependencies & jalankan:
   ```bash
   go mod tidy
   go run main.go
   ```
   Server berjalan di `http://localhost:8080`, otomatis migrate tabel dan
   seed data dummy Fajar & Raisa jika database masih kosong.

## Protokol enkripsi (penting untuk frontend)

Setiap body request (kecuali GET tanpa body) dan setiap body response harus
berbentuk:
```json
{ "data": "<base64(nonce(12B) || ciphertext || tag)>" }
```
menggunakan AES-256-GCM dengan key 32-byte yang sama persis dengan
`APP_ENC_KEY` (hex 64 karakter) di `.env`. Key yang sama **wajib** dipasang di
frontend (`src/lib/crypto.ts`). Karena GCM authenticated, payload yang
dirusak di tengah jalan otomatis ditolak.

> Catatan produksi: demo ini pakai *shared static key* untuk kesederhanaan.
> Untuk production sebaiknya key di-rotate per sesi (mis. via handshake
> Diffie-Hellman/ECDH setelah login) dan selalu jalankan di atas HTTPS.

## Endpoint ringkas

| Method | Path                     | Keterangan                         |
|--------|--------------------------|-------------------------------------|
| POST   | /auth/register           | Daftar akun baru                    |
| POST   | /auth/login              | Login, dapat JWT token              |
| GET    | /auth/me                 | Profil user login                   |
| POST   | /auth/link-partner       | Hubungkan pasangan via kode         |
| PATCH  | /auth/profile            | Update profil                       |
| GET/POST/PATCH/DELETE | /wallets, /budgets, /wedding, /plans, /goals | CRUD standar |
| GET/POST/DELETE | /transactions, /cashflows | Catat transaksi/cashflow |
| GET    | /backup/export           | Export seluruh data user (+pasangan) sebagai JSON |
| POST   | /backup/import           | Import JSON backup                  |
| GET    | /backup/status           | Info kapan backup terakhir & apakah sudah lewat 28 hari |

## Dummy mode

Jika suatu saat MySQL belum siap / API belum dijalankan, frontend tetap bisa
berjalan pakai mode dummy (localStorage) bawaan `src/lib/api.ts` — cukup
biarkan `VITE_API_URL` kosong di frontend `.env`.
