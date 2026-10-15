CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now'))
);

CREATE TABLE app_settings (
    id                  INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    openrouter_api_key  TEXT NOT NULL DEFAULT '',
    gemini_api_key      TEXT NOT NULL DEFAULT '',
    opencode_go_api_key TEXT NOT NULL DEFAULT '',
    api_provider        TEXT NOT NULL DEFAULT 'openrouter' CHECK (api_provider IN ('openrouter', 'gemini', 'opencode_go')),
    model_name          TEXT NOT NULL DEFAULT 'anthropic/claude-3.5-sonnet',
    updated_at          DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now'))
);

INSERT INTO app_settings (id) VALUES (1);

CREATE TABLE instructions (
    key        TEXT PRIMARY KEY,
    content    TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now'))
);

CREATE TABLE history (
    id              TEXT PRIMARY KEY,
    type            TEXT NOT NULL CHECK (type IN (
                        'simplify', 'en_fa', 'fa_en', 'term_en', 'term_fa', 'refine', 'symptoms',
                        'compare_en', 'compare_fa', 'grammar_en', 'grammar_fa', 'cursor', 'frontend')),
    input_text      TEXT NOT NULL,
    result_text     TEXT NOT NULL,
    model           TEXT NOT NULL,
    instruction_key TEXT NOT NULL,
    metadata        TEXT,
    quiz_shown_count INTEGER NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%f', 'now'))
);

CREATE INDEX idx_history_created_at ON history(created_at DESC);
CREATE INDEX idx_history_type ON history(type);
