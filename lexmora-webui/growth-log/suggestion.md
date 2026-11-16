# Suggestions

Duplicate deploy roots (root scripts and `.deploy/docker/`) were removed; `.armin/docker-scripts/` is the only entry point.

- Align `docker-compose.yml` env names with skill defaults (`IMAGE_TAG` / `PUBLISH_PORT`) or keep documenting the `WEB_*` mapping.
- Add history pagination in the UI when the list grows.
- Deploy `lexmora-api` (SQLite on `lexmora-api-vol`) on T3 `t3-net` so `https://lexmora.xaigrok.ir` login/`/api/*` work; HAProxy already routes `lexmora-api.xaigrok.ir`.
- Make `WEB_PUBLISH_PORT` truly optional in `docker-compose.yml` so empty server YAML does not bind host `:8082`.
- Strip inline `#` comments in the flat YAML reader used by deploy scripts.
