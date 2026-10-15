# Lexmora Web UI

Vue 3 SPA for the Lexmora app. Dark-themed UI for login, transforms (including compare), history, editable AI instructions, usage stats, and OpenRouter settings. Includes installable PWA support with a service worker that precaches the UI shell and static assets.

The Go API (SQLite database) lives in the separate [lexmora-api](https://github.com/ArminDashti/lexmora-api) repository.

## Tech stack

- Vue 3 + TypeScript + Tailwind CSS
- Vite 6 (dev `/api` proxy)
- `vite-plugin-pwa` for manifest and service worker
- nginx in Docker for production (proxies `/api` to the API container)

## Run

### Development

1. Start the API from `lexmora-api` on port 8080
2. `npm install && npm run dev` → http://localhost:5173

### Docker (preferred)

Primary deploy assets live under `.armin/docker-scripts/` (YAML-only; no CLI flags).

```powershell
# Edit YAML if needed, then:
.\.armin\docker-scripts\run-on-docker-local.ps1

# Remote (fill ssh + volume_dir in run-on-docker-server.yaml first):
.\.armin\docker-scripts\run-on-docker-server.ps1
```

- Local URL: http://localhost:8082 (`publish_port` → `WEB_PUBLISH_PORT`)
- Scripts set `WEB_IMAGE_TAG`, `WEB_PUBLISH_PORT`, `DOCKER_NETWORK`, `API_HOST`, and `API_PORT` for `docker-compose.yml`
- Local YAML sets `delete_image: "yes"` so each deploy rebuilds from a clean image
- API stack must be on network `t3-net` first (same `docker_network` as the web UI)
- Local/server scripts use `--force-recreate` so network and API host changes always apply

Default login: `armin` / `Translator@2024` (validated by the API)

## PWA behavior

- Installable in supported browsers (best on HTTPS in production)
- Offline behavior is app-shell only (cached UI and static files)
- API requests (`/api/*`) are network-only and are not cached
