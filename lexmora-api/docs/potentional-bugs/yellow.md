# Potential bugs (minor)

**[Migration 002]** — Fixed: `default_model_id` FK is dropped before `llm_models`. Use `--delete-volume=yes` if a prior failed migration left the DB dirty.

**[CORS]** — Local Docker defaults allow `localhost:5173` (Vite) and `localhost:8082` (web UI). Add other hostnames via `CORS_ORIGINS` / deploy YAML.

**[JWT]** — Tokens expire after 7 days with no refresh flow; user must re-login.

**[OpenRouter key]** — Stored in plaintext in SQLite (`app_settings`). Acceptable for single-user local use; encrypt at rest for shared deployments.
