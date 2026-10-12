# Goals API

REST API for Goals, a couple finance and wedding planning application, built with Go, Echo v4, GORM, and MySQL.

Demo accounts: **Fajar Nugroho** and **Raisa Amelia** (password: `demo123`). Demo credentials are for local development only.

## Features

- Echo v4 HTTP server and GORM with MySQL.
- AES-256-GCM request and response encryption via `middleware/encryption.go`.
- JWT authentication with `Authorization: Bearer <token>`.
- JSON backup export/import and a 28-day backup reminder.
- Wallets, budgets, transactions, cash flows, wedding planning, plans, and goals.

## Local Setup

1. Configure environment variables:

   ```bash
   cp .env.example .env
   ```

2. Start MySQL using Docker Compose, or use a local MySQL instance:

   ```bash
   docker compose up -d
   ```

3. Install dependencies and start the API:

   ```bash
   go mod tidy
   go run main.go
   ```

The server runs at `http://localhost:8080`. On startup, it migrates the database and seeds demo users if the database is empty.

## Encryption Protocol

Requests with a body and all JSON responses use this envelope:

```json
{
  "data": "<base64(nonce(12B) || ciphertext || tag)>"
}
```

- Algorithm: AES-256-GCM.
- Key: 32 bytes, configured as 64 hexadecimal characters in `APP_ENC_KEY`.
- Clients must use the same key (see `src/lib/crypto.ts`).
- Invalid or tampered encrypted payloads are rejected.

**Security:** Always use HTTPS in production. A static key embedded in a distributed client can be extracted; use a safer key-management design before relying on this encryption layer for production security.

## API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| POST | `/auth/register` | Register an account |
| POST | `/auth/login` | Authenticate and receive a JWT |
| GET | `/auth/me` | Get the current user |
| POST | `/auth/link-partner` | Link a partner using a code |
| PATCH | `/auth/profile` | Update the user profile |
| GET, POST, PATCH, DELETE | `/wallets`, `/budgets`, `/wedding`, `/plans`, `/goals` | Manage resources |
| GET, POST, DELETE | `/transactions`, `/cashflows` | Manage financial activity |
| GET | `/backup/export` | Export user and partner data as JSON |
| POST | `/backup/import` | Import a JSON backup |
| GET | `/backup/status` | Check backup date and 28-day reminder |

## Dummy Mode

The frontend can run without the API using its built-in `localStorage` dummy mode (`src/lib/api.ts`). Leave `VITE_API_URL` empty in the frontend `.env` to enable it.
