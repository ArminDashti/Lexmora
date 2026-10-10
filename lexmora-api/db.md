# Database Schema

PostgreSQL database for the Lexmora app. Migrations live in `migrations/`.

## Overview

| Item | Value |
|------|-------|
| Engine | PostgreSQL 16 |
| Database | `lexmora` |
| Migrations | `001_init` … `007_gemini_provider` |

## Tables

### app_settings

Singleton row for LLM provider configuration.

| Column | Type | Constraints |
|--------|------|-------------|
| id | INT | PRIMARY KEY, DEFAULT 1, CHECK (id = 1) |
| openrouter_api_key | TEXT | NOT NULL, DEFAULT '' |
| gemini_api_key | TEXT | NOT NULL, DEFAULT '' |
| opencode_go_api_key | TEXT | NOT NULL, DEFAULT '' |
| api_provider | TEXT | NOT NULL, DEFAULT `openrouter`, CHECK (`openrouter`\|`gemini`\|`opencode_go`) |
| model_name | TEXT | NOT NULL, DEFAULT `anthropic/claude-3.5-sonnet` |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |

### users

Application login accounts.

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PRIMARY KEY, DEFAULT gen_random_uuid() |
| username | TEXT | NOT NULL, UNIQUE |
| password_hash | TEXT | NOT NULL |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |

### instructions

Editable prompt text per operation mode.

| Column | Type | Constraints |
|--------|------|-------------|
| key | TEXT | PRIMARY KEY |
| content | TEXT | NOT NULL, DEFAULT '' |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |

### history

Stored transform results.

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PRIMARY KEY, DEFAULT gen_random_uuid() |
| type | history_type | NOT NULL |
| input_text | TEXT | NOT NULL |
| result_text | TEXT | NOT NULL |
| model | TEXT | NOT NULL |
| instruction_key | TEXT | NOT NULL |
| metadata | JSONB | nullable |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |

## Enum Types

### history_type

| Value | Description |
|-------|-------------|
| simplify | Simplify English text |
| en_fa | English to Persian |
| fa_en | Persian to English |
| term_en | English term lookup |
| term_fa | Persian term lookup |
| refine | Refine tone/style |
| symptoms | Symptoms operation |
| compare_en | Compare two words (English explanation) |
| compare_fa | Compare two words (Persian explanation) |

## Indexes

| Index | Table | Columns |
|-------|-------|---------|
| idx_history_created_at | history | created_at DESC |
| idx_history_type | history | type |

## Relationships

| From | To | Notes |
|------|----|-------|
| history.instruction_key | instructions.key | Logical link (no FK) |
