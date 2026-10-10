ALTER TABLE app_settings
  ADD COLUMN IF NOT EXISTS opencode_go_api_key TEXT NOT NULL DEFAULT '';

ALTER TABLE app_settings
  DROP CONSTRAINT IF EXISTS app_settings_api_provider_check;

ALTER TABLE app_settings
  ADD CONSTRAINT app_settings_api_provider_check
  CHECK (api_provider IN ('openrouter', 'gemini', 'opencode_go'));
