# Docker deployment

Web UI nginx container from this repo. The API (SQLite) runs separately from [lexmora-api](https://github.com/ArminDashti/lexmora-api).

## Preferred files (`.armin/docker-scripts/`)

| File | Purpose |
|------|---------|
| `run-on-docker-local.ps1` | Local Docker deploy (reads sibling YAML only) |
| `run-on-docker-local.yaml` | Local stack/image/network/API settings |
| `run-on-docker-server.ps1` | Remote deploy over SSH (YAML-only) |
| `run-on-docker-server.yaml` | Remote settings (`ssh t3`, `volume_dir` under `/home/cloud-admin/docker/`) |

## Other Docker files

| File | Purpose |
|------|---------|
| `Dockerfile` | Vue build + nginx image |
| `docker-compose.yml` | `lexmora-webui` on external `t3-net` |
| `nginx.conf.template` | Proxies `/api/` to the API container (Docker DNS resolver) |
| `.docker/stack.manifest.json` | Image tags and ports |

## Service

| Service | Container | Host port | Notes |
|---------|-----------|-----------|-------|
| `lexmora-webui` | `lexmora-webui` | `8082` (`publish_port`) | Serves static UI; proxies `/api/*` → `${API_HOST}:${API_PORT}` |

## Local run

```powershell
.\.armin\docker-scripts\run-on-docker-local.ps1
```

API stack must be on `t3-net` first. Scripts create the network if missing and run `compose up -d --force-recreate` so a previous network name (e.g. `lexmora-net`) cannot stick.

| YAML / env | Default | Description |
|------------|---------|-------------|
| `api_host` → `API_HOST` | `lexmora-api` | API hostname on Docker network |
| `api_port` → `API_PORT` | `8080` | API port |
| `image_tag` → `WEB_IMAGE_TAG` | `lexmora-webui:latest` | Image used by compose |
| `publish_port` → `WEB_PUBLISH_PORT` | `8082` | Host port for the UI (required locally; leave empty on server — compose still defaults to `8082` today) |
| `delete_image` | `yes` | Remove image before rebuild on each deploy |
| `docker_network` → `DOCKER_NETWORK` | `t3-net` | Shared network with the API stack |

## Server (Irancell-T3)

- Host: `t3` / `t3-new` (`2.144.27.74`), user `cloud-admin`
- Deploy: `.\.armin\docker-scripts\run-on-docker-server.ps1` (`build_image_on: local`)
- Public URL: `https://lexmora.xaigrok.ir` via HAProxy on `t3-net` → `lexmora-webui:80`
- HAProxy already routes SNI/Host `lexmora.xaigrok.ir` (cert PEM present); no cfg change needed for web UI
- Companion API must also run on `t3-net` as `lexmora-api:8080` for `/api/*` and `lexmora-api.xaigrok.ir`

## nginx proxy

`nginx.conf.template` uses Docker embedded DNS (`127.0.0.11`) and a variable `proxy_pass` so the API hostname is resolved at request time. That keeps nginx from exiting at startup if the API is briefly unreachable; the web UI and API containers must still share `docker_network`.

With a variable upstream, `proxy_pass` must not include a URI path (use `http://$api_upstream` only). A path like `/api/` would replace the full request URI and break proxied routes (e.g. `/api/v1/auth/login` → `/api/`).
