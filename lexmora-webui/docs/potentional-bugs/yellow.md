# Minor risks / code smells

- **[Docker]** Local deploy requires YAML `publish_port` (currently `8082`). Server YAML leaves it empty, but `docker-compose.yml` still defaults `WEB_PUBLISH_PORT` to `8082`, so the host port is published on T3 anyway.
- **[Docker]** Flat YAML reader does not strip inline `#` comments — keep values on their own line (no trailing comments).
- **[Prod]** Lexmora on T3 uses SQLite (`lexmora-api-vol` → `/data/lexmora.db`); `lexmora-pgsql` removed. Login/`/api/*` need `lexmora-api` Up on `t3-net` behind HAProxy.
- **[nginx]** Fixed: variable `proxy_pass` must not include a URI path (`/api/`); that replaced the full request URI and broke `/api/v1/*` proxying.
- **[Auth]** Client only checks token presence in localStorage; expired JWTs still pass the router guard until the next API call returns 401.
- **[Stats]** If an older API without `compare` in `StatsBucket` is used, the Compare column shows `undefined`.
