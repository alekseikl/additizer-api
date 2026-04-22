# additizer-api

A small REST service written in Go that exposes user registration and login
endpoints backed by PostgreSQL (via GORM) and uses JWTs for authentication.

The HTTP layer is built on the standard library `net/http` package using the
Go 1.22+ method-aware `ServeMux`.

## Project layout

```
cmd/server                 # main entrypoint
internal/config            # env-based configuration loader
internal/database          # gorm connection + auto-migration
internal/models            # GORM models (User)
internal/auth              # password hashing + JWT issuer/parser
internal/middleware        # JWT auth middleware
internal/handlers          # HTTP handlers (auth, health)
internal/httpx             # small JSON response helpers
```

## Requirements

- Go 1.22+
- A running PostgreSQL instance

## Configuration

Copy `.env.example` to `.env` and adjust values:

```bash
cp .env.example .env
```

| Variable               | Description                                         | Default   |
|------------------------|-----------------------------------------------------|-----------|
| `HTTP_ADDR`            | Address the HTTP server listens on                  | `:8080`   |
| `DATABASE_URL`         | Full GORM Postgres DSN                              | *(required)* |
| `DB_HOST`/`DB_PORT`/…  | Used to build a DSN when `DATABASE_URL` is empty    | —         |
| `JWT_SECRET`           | HMAC secret used to sign JWTs                       | *(required)* |
| `JWT_EXPIRATION_HOURS` | Token lifetime in hours                             | `24`      |
| `BCRYPT_COST`          | bcrypt cost factor                                  | `12`      |

## Run

```bash
go mod tidy
go run ./cmd/server
```

On startup the service connects to Postgres and runs `AutoMigrate` for the
`users` table.

## Endpoints

### `POST /api/v1/auth/register`

```json
{
  "email": "jane@example.com",
  "username": "jane",
  "password": "supersecret"
}
```

Returns `201 Created` with the created user and a JWT:

```json
{
  "token": "eyJhbGci…",
  "expires_at": "2026-04-22T10:00:00Z",
  "user": {
    "id": "…",
    "email": "jane@example.com",
    "username": "jane",
    "created_at": "…"
  }
}
```

### `POST /api/v1/auth/login`

Accepts either an email or a username as `identifier`:

```json
{
  "identifier": "jane@example.com",
  "password": "supersecret"
}
```

Returns `200 OK` with the same shape as `register`.

### `GET /api/v1/me` (protected)

Requires an `Authorization: Bearer <token>` header. Returns the authenticated
user.

### `GET /healthz`

Liveness probe.

## cURL examples

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"jane@example.com","username":"jane","password":"supersecret"}'

curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"identifier":"jane","password":"supersecret"}'

curl -s http://localhost:8080/api/v1/me \
  -H "Authorization: Bearer $TOKEN"
```
