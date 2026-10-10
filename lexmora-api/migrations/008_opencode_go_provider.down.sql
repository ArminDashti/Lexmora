ALTER TABLE app_settings
  DROP CONSTRAINT IF EXISTS app_settings_api_provider_check;

ALTER TABLE app_settings
  ADD CONSTRAINT app_settings_api_provider_check
  CHECK (api_provider IN ('openrouter', 'gemini'));

ALTER TABLE app_settings
  DROP COLUMN IF EXISTS opencode_go_api_key;
