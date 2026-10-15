# Lexmora API

Go REST API for the Lexmora app: AI-powered English ↔ Persian transforms via OpenRouter, with JWT auth, history, instructions, stats, and settings.

Companion frontend: [lexmora-webui](https://github.com/ArminDashti/lexmora-webui)

## Stack

- Go 1.22 (Gin)
- SQLite (pure Go driver `modernc.org/sqlite`, no CGO)
- OpenRouter (API key stored in DB via settings)

## Prerequisites

- Go 1.22+
- Docker (optional, for the containerized stack)

## Setup

1. Copy environment file:

```bash
cp .env.example .env
```

2. Edit `.env` — set `JWT_SECRET` to a long random string.

3. Start the API with Docker:

```powershell
.\run-on-docker-local.ps1
```

Settings live in `.armin/docker-scripts/run-on-docker-local.yaml` (no CLI flags). See [docs/modules/docker.md](docs/modules/docker.md).

Or run the API locally (the SQLite file is created automatically at `DATABASE_PATH`):

```powershell
go run ./cmd/server
```

API listens on [http://localhost:8080](http://localhost:8080).

### Development with the web UI

Run this API on port 8080, then start the web UI dev server from the `lexmora-webui` repo (`npm run dev` on port 5173). Set `CORS_ORIGINS` to include the Vite origin if needed.

## Default login

| Field    | Value            |
|----------|------------------|
| Username | `armin`          |
| Password | `IUpe8SqXxtJrEJxZ` |

Override via `DEFAULT_USERNAME` and `DEFAULT_PASSWORD` in `.env` (only used on first boot when no users exist).

## Configuration

| Variable | Description |
|----------|-------------|
| `PORT` | HTTP port (default `8080`) |
| `JWT_SECRET` | Secret for signing login tokens |
| `DATABASE_PATH` | SQLite database file path (default `lexmora.db`; Docker uses `/data/lexmora.db`) |
| `CORS_ORIGINS` | Comma-separated allowed browser origins |
| `STATIC_DIR` | Optional path to serve a built SPA (leave unset for API-only) |
| `DEFAULT_USERNAME` | Initial admin username |
| `DEFAULT_PASSWORD` | Initial admin password |

OpenRouter API key and model are configured through the Settings API (stored in the database).

## API overview

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/health` | No | Health check |
| POST | `/api/v1/auth/login` | No | Login |
| POST | `/api/v1/transform` | JWT | Run operation |
| GET | `/api/v1/transform/options` | JWT | Dynamic operation catalog |
| GET | `/api/v1/history` | JWT | List history (`type` multi comma-separated, `from`, `to`, sort, paging) |
| GET | `/api/v1/history/:id` | JWT | Get history item |
| DELETE | `/api/v1/history/:id` | JWT | Delete history item |
| POST | `/api/v1/quiz` | JWT | Build MCQ quiz from history |
| GET | `/api/v1/stats` | JWT | Usage statistics |
| GET | `/api/v1/instructions` | JWT | List instructions |
| POST | `/api/v1/instructions` | JWT | Create instruction from operation fields |
| GET | `/api/v1/instructions/:key` | JWT | Get instruction |
| PUT | `/api/v1/instructions/:key` | JWT | Update instruction |
| GET | `/api/v1/settings` | JWT | Get settings |
| PATCH | `/api/v1/settings` | JWT | Update settings |
| GET | `/api/v1/settings/models` | JWT | Search OpenRouter models |
| GET | `/api/v1/settings/credits` | JWT | OpenRouter credits / key usage |
| DELETE | `/api/v1/settings/data` | JWT | Clear all history |

See [docs/endpoints.md](docs/endpoints.md) for details.

## Project structure

```
cmd/server/           API entry point
internal/config/      Environment configuration
internal/db/          SQLite connection and migrations
internal/domain/      Entities and DTOs
internal/handler/     HTTP handlers
internal/middleware/  JWT auth middleware
internal/repository/  Data access
internal/service/     Business logic
migrations/           SQL migrations
instructions/         Reference prompt templates
```

## Docker

Build and run the API (SQLite lives in the `lexmora-api-vol` volume):

```bash
docker network create lexmora-net
docker compose up -d --build
```

The web UI container (separate repo) should join the same `lexmora-net` network and proxy `/api` to the `lexmora-api` service hostname.
