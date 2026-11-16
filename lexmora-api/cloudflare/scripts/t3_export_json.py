#!/usr/bin/env python3
"""Export Lexmora SQLite tables from Irancell-T3 to local JSON (for D1 / offline use).

Lexmora on T3 is SQLite-only (`DATABASE_PATH=/data/lexmora.db` on volume
`lexmora-api-vol`). Postgres (`lexmora-pgsql`) was removed 2026-11-16.
"""
from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

KEY = Path.home() / ".ssh" / "id_ed25519_cloud-admin_2-144-27-74"
HOST = "cloud-admin@2.144.27.124"
PORT = "443"
OUT = Path(__file__).resolve().parent / "t3_export.json"

REMOTE_PY = r"""
import json, sqlite3
c = sqlite3.connect("/tmp/lexmora_export.db")
c.row_factory = sqlite3.Row

def rows(table):
    return [dict(r) for r in c.execute(f"SELECT * FROM {table}")]

print(json.dumps({
    "users": rows("users"),
    "app_settings": rows("app_settings"),
    "instructions": rows("instructions"),
    "history": rows("history"),
}, ensure_ascii=False, default=str))
""".strip()


def run(cmd: list[str], check: bool = True) -> subprocess.CompletedProcess[str]:
    proc = subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
    )
    if check and proc.returncode != 0:
        raise SystemExit(
            f"command failed ({proc.returncode}): {' '.join(cmd)}\n"
            f"stdout: {proc.stdout}\nstderr: {proc.stderr}"
        )
    return proc


def ssh(remote_cmd: str) -> str:
    return run(
        [
            "ssh",
            "-p",
            PORT,
            "-i",
            str(KEY),
            "-o",
            "IdentitiesOnly=yes",
            "-o",
            "BatchMode=yes",
            HOST,
            remote_cmd,
        ]
    ).stdout


def main() -> None:
    # Checkpoint WAL into a copyable main DB, then dump tables as JSON on the host.
    remote = (
        "docker exec lexmora-api sh -c "
        "'cp /data/lexmora.db /tmp/lexmora_export.db "
        "&& cp /data/lexmora.db-wal /tmp/lexmora_export.db-wal 2>/dev/null || true "
        "&& cp /data/lexmora.db-shm /tmp/lexmora_export.db-shm 2>/dev/null || true' && "
        "docker cp lexmora-api:/tmp/lexmora_export.db /tmp/lexmora_export.db && "
        "docker cp lexmora-api:/tmp/lexmora_export.db-wal /tmp/lexmora_export.db-wal 2>/dev/null || true && "
        "docker cp lexmora-api:/tmp/lexmora_export.db-shm /tmp/lexmora_export.db-shm 2>/dev/null || true && "
        "python3 - <<'PY'\n"
        f"{REMOTE_PY}\n"
        "PY"
    )
    raw = ssh(remote).strip()
    # ssh may prepend non-JSON noise; take last JSON object line.
    line = raw.splitlines()[-1]
    data = json.loads(line)
    OUT.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    for table, rows in data.items():
        print(f"{table}: {len(rows)} rows", file=sys.stderr)
    print(f"wrote {OUT}")


if __name__ == "__main__":
    main()
