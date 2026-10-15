# Database Schema

SQLite database for the Lexmora app. Schema lives in `migrations/001_init.up.sql`.

## Overview

| Item | Value |
|------|-------|
| Engine | SQLite (via `modernc.org/sqlite`, pure Go) |
| File | `DATABASE_PATH` (default `lexmora.db`; Docker: `/data/lexmora.db`) |
| Migrations | `001_init` (golang-migrate, runs on API start) |
| Pragmas | `journal_mode=WAL`, `busy_timeout=5000`, `foreign_keys=1` |

## Conventions

- UUIDs are stored as `TEXT` and generated in Go.
- Timestamps are `DATETIME` text in UTC (`YYYY-MM-DD HH:MM:SS.SSS`), defaulted by `strftime`.
- JSON metadata is stored as `TEXT`.

## Tables

### app_settings

Singleton row for LLM provider configuration.

| Column | Type | Constraints |
|--------|------|-------------|
| id | INTEGER | PRIMARY KEY, DEFAULT 1, CHECK (id = 1) |
| openrouter_api_key | TEXT | NOT NULL, DEFAULT '' |
| gemini_api_key | TEXT | NOT NULL, DEFAULT '' |
| opencode_go_api_key | TEXT | NOT NULL, DEFAULT '' |
| api_provider | TEXT | NOT NULL, DEFAULT `openrouter`, CHECK (`openrouter`, `gemini`, `opencode_go`) |
| model_name | TEXT | NOT NULL, DEFAULT `anthropic/claude-3.5-sonnet` |
| updated_at | DATETIME | NOT NULL |

### users

Application login accounts.

| Column | Type | Constraints |
|--------|------|-------------|
| id | TEXT | PRIMARY KEY (UUID) |
| username | TEXT | NOT NULL, UNIQUE |
| password_hash | TEXT | NOT NULL |
| created_at | DATETIME | NOT NULL |

### instructions

Editable prompt text per operation mode.

| Column | Type | Constraints |
|--------|------|-------------|
| key | TEXT | PRIMARY KEY |
| content | TEXT | NOT NULL, DEFAULT '' |
| updated_at | DATETIME | NOT NULL |

### history

Stored transform results.

| Column | Type | Constraints |
|--------|------|-------------|
| id | TEXT | PRIMARY KEY (UUID) |
| type | TEXT | NOT NULL, CHECK (one of the values below) |
| input_text | TEXT | NOT NULL |
| result_text | TEXT | NOT NULL |
| model | TEXT | NOT NULL |
| instruction_key | TEXT | NOT NULL |
| metadata | TEXT | nullable (JSON) |
| quiz_shown_count | INTEGER | NOT NULL, DEFAULT 0 |
| created_at | DATETIME | NOT NULL |

## Allowed `history.type` values

`simplify`, `en_fa`, `fa_en`, `term_en`, `term_fa`, `refine`, `symptoms`, `compare_en`, `compare_fa`, `grammar_en`, `grammar_fa`, `cursor`, `frontend`

## Indexes

| Index | Table | Columns |
|-------|-------|---------|
| idx_history_created_at | history | created_at DESC |
| idx_history_type | history | type |

## Relationships

| From | To | Notes |
|------|----|-------|
| history.instruction_key | instructions.key | Logical link (no FK) |
