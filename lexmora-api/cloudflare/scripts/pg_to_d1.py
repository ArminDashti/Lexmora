#!/usr/bin/env python3
"""Convert Postgres COPY/CSV-ish export or JSON rows into D1 INSERT statements."""
from __future__ import annotations

import csv
import json
import sys
from pathlib import Path


def esc(value: object) -> str:
    if value is None:
        return "NULL"
    s = str(value)
    return "'" + s.replace("'", "''") + "'"


def emit_users(rows: list[dict]) -> list[str]:
    out = ["DELETE FROM users;"]
    for r in rows:
        out.append(
            "INSERT INTO users (id, username, password_hash, created_at) VALUES ("
            f"{esc(r['id'])}, {esc(r['username'])}, {esc(r['password_hash'])}, {esc(r.get('created_at'))});"
        )
    return out


def emit_settings(rows: list[dict]) -> list[str]:
    out = []
    for r in rows:
        out.append(
            "UPDATE app_settings SET "
            f"openrouter_api_key = {esc(r.get('openrouter_api_key', ''))}, "
            f"gemini_api_key = {esc(r.get('gemini_api_key', ''))}, "
            f"opencode_go_api_key = {esc(r.get('opencode_go_api_key', ''))}, "
            f"api_provider = {esc(r.get('api_provider', 'openrouter'))}, "
            f"model_name = {esc(r.get('model_name', 'anthropic/claude-3.5-sonnet'))}, "
            f"updated_at = {esc(r.get('updated_at'))} WHERE id = 1;"
        )
    return out


def emit_instructions(rows: list[dict]) -> list[str]:
    out = ["DELETE FROM instructions;"]
    for r in rows:
        out.append(
            "INSERT INTO instructions (key, content, updated_at) VALUES ("
            f"{esc(r['key'])}, {esc(r['content'])}, {esc(r.get('updated_at'))});"
        )
    return out


def emit_history(rows: list[dict]) -> list[str]:
    out = ["DELETE FROM history;"]
    for r in rows:
        meta = r.get("metadata")
        if meta is not None and not isinstance(meta, str):
            meta = json.dumps(meta, ensure_ascii=False)
        out.append(
            "INSERT INTO history (id, type, input_text, result_text, model, instruction_key, metadata, created_at, quiz_shown_count) VALUES ("
            f"{esc(r['id'])}, {esc(r['type'])}, {esc(r['input_text'])}, {esc(r['result_text'])}, "
            f"{esc(r['model'])}, {esc(r['instruction_key'])}, {esc(meta)}, {esc(r.get('created_at'))}, "
            f"{int(r.get('quiz_shown_count') or 0)});"
        )
    return out


def main() -> int:
    if len(sys.argv) != 3:
        print("usage: pg_to_d1.py <export.json> <out.sql>", file=sys.stderr)
        return 2
    data = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
    # D1 import rejects SQL BEGIN/COMMIT; statements run sequentially.
    lines: list[str] = []
    lines += emit_users(data.get("users", []))
    lines += emit_settings(data.get("app_settings", []))
    lines += emit_instructions(data.get("instructions", []))
    lines += emit_history(data.get("history", []))
    Path(sys.argv[2]).write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(f"wrote {sys.argv[2]}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
