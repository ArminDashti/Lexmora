# Docker deployment

The API (with its SQLite database) runs from this repo. The web UI is deployed separately from [lexmora-webui](https://github.com/ArminDashti/lexmora-webui).

## Files

| Path | Purpose |
|------|---------|
| `.armin/docker-scripts/run-on-docker-local.ps1` | Local Docker daemon deploy (YAML-only) |
| `.armin/docker-scripts/run-on-docker-local.yaml` | Local settings |
| `.armin/docker-scripts/run-on-docker-server.ps1` | Remote SSH deploy (YAML-only) |
| `.armin/docker-scripts/run-on-docker-server.yaml` | Remote settings (`ssh`, `volume_dir`) |
| `.deploy/docker/Dockerfile` | Go API image (build context = repo root) |
| `docker-compose.yml` | `lexmora-api` on external `t3-net` (SQLite file in volume `lexmora-api-vol`) |
| `.deploy/docker/docker-compose.openrouter-proxy.yml` | Server overlay: join `openvpn-net`, set `OPENROUTER_HTTP_PROXY=http://mullvad-1:8778` |
| `.deploy/docker/docker-compose.publish.yml` | Legacy optional host-port overlay (compose now has inline `API_PUBLISH_PORT`) |
| `.deploy/docker/run-on-docker-*.ps1` / `.yaml` | Legacy deploy pair (prefer `.armin/docker-scripts/`) |
| `.deploy/docker/common.ps1` | Shared helpers for legacy scripts |
| `run-on-docker-local.ps1` | Repo-root stub → `.armin/docker-scripts/` |
| `run-on-docker-server.ps1` | Repo-root stub → `.armin/docker-scripts/` |
| `.docker/stack.manifest.json` | Stack metadata (names, network, remote path) |

## Services

| Service | Container | Host port | Notes |
|---------|-----------|-----------|-------|
| `lexmora-api` | `lexmora-api` | Host bind only via `docker-compose.publish.yml` when `publish_port` is set | HAProxy uses Docker DNS `lexmora-api:8080` on `t3-net` |

Compose override env vars from `.armin` scripts: `API_IMAGE_TAG`, `DOCKER_NETWORK`, `API_PUBLISH_PORT`.

## Local run

Edit `.armin/docker-scripts/run-on-docker-local.yaml` if needed, then:

```powershell
.\run-on-docker-local.ps1
```

Or:

```powershell
.\.armin\docker-scripts\run-on-docker-local.ps1
```

The script builds the image, ensures external `t3-net`, and runs `docker compose up -d`.

**Default login:** `armin` / `IUpe8SqXxtJrEJxZ` (compose defaults).

**CORS:** Local compose / `run-on-docker-local.yaml` defaults include Vite (`5173`), Docker web UI (`8082`), and `https://lexmora.xaigrok.ir`. Override `CORS_ORIGINS` for other hostnames.

## Remote run (Irancell-T3 / HAProxy)

Target host: `ssh t3 -p 80` (`cloud-admin@2.144.27.74`). Stack dir: `/cloud-admin/docker/lexmora`. Network: external `t3-net` (shared with HAProxy).

HAProxy already routes:

| Host | Backend |
|------|---------|
| `lexmora-api.xaigrok.ir` | `lexmora-api:8080` |
| `lexmora.xaigrok.ir` | `lexmora-webui:80` |

1. Confirm `.armin/docker-scripts/run-on-docker-server.yaml` (`ssh: "ssh t3 -p 80"`, `delete_image: "yes"`, empty `publish_port`).
2. Do not put inline `#` comments on the same line as YAML values (flat parser treats them as part of the value).
3. Run:

```powershell
.\run-on-docker-server.ps1
```

`build_image_on: local` (default): build here → `docker save` → SCP → remote teardown (`compose down` + optional `image rm`) → `docker load` → `compose up -d`.  
`build_image_on: server`: sync repo → teardown → remote `docker build` → `compose up -d`.

Image load/build runs **after** `delete_image` teardown so a fresh upload is not wiped.

Public DNS: `lexmora.xaigrok.ir` → `2.144.27.74`. Ensure `lexmora-api.xaigrok.ir` has the same A record (cert + HAProxy backend already exist).

## Full stack

1. Start this repo: `.\run-on-docker-local.ps1`
2. Start [lexmora-webui](https://github.com/ArminDashti/lexmora-webui) on the same `t3-net` network with `API_HOST=lexmora-api`.
