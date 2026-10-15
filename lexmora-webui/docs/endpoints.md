# Endpoints / Commands

## Docker deploy scripts

Entry point: `.armin/docker-scripts/` (YAML-only; no CLI flags).

| Command | Auth | Description |
|---------|------|-------------|
| `.\.armin\docker-scripts\run-on-docker-local.ps1` | No | Local deploy: build, force-recreate on `t3-net`, publish `publish_port` |
| `.\.armin\docker-scripts\run-on-docker-server.ps1` | SSH (`t3`) | Remote deploy to Irancell-T3; YAML has `ssh: "ssh t3"` + `volume_dir` |
| `.\create-image.ps1` | No | Build `lexmora-webui` image only |

# API Endpoints

Consumed from [lexmora-api](https://github.com/ArminDashti/lexmora-api). All authenticated routes require `Authorization: Bearer <jwt>`.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/health` | No | Health check (not called by UI) |
| POST | `/api/v1/auth/login` | No | Login with username/password, returns JWT |
| POST | `/api/v1/transform` | Yes | Run a transform operation |
| GET | `/api/v1/transform/options` | Yes | Dynamic Operation / Direction / Mode catalog from instruction keys |
| GET | `/api/v1/history` | Yes | List history (`sort_by`, `sort_order`, `type`, `from`, `to`, `limit`, `offset`) |
| GET | `/api/v1/history/:id` | Yes | Get single history record |
| DELETE | `/api/v1/history/:id` | Yes | Delete history record |
| GET | `/api/v1/stats` | Yes | Request counts by period and type |
| GET | `/api/v1/instructions` | Yes | List all instruction keys |
| POST | `/api/v1/instructions` | Yes | Create instruction from operation / direction / mode |
| GET | `/api/v1/instructions/:key` | Yes | Get instruction content |
| PUT | `/api/v1/instructions/:key` | Yes | Update instruction content |
| GET | `/api/v1/settings` | Yes | Get provider keys and model (`api_provider`: `openrouter`\|`gemini`\|`opencode_go`, keys, `model_name`) |
| PATCH | `/api/v1/settings` | Yes | Update provider, keys, and/or model |
| GET | `/api/v1/settings/models` | Yes | Search models for active provider (`q`; OpenRouter or OpenCode Go) |
| GET | `/api/v1/settings/credits` | Yes | Remaining OpenRouter credits / key usage |
| DELETE | `/api/v1/settings/data` | Yes | Delete all history rows |

## History filters

- `type` — exact history type code (`en_fa`, `simplify`, …)
- `from` / `to` — `YYYY-MM-DD` inclusive calendar days (local server TZ)

## Transform options — `GET /api/v1/transform/options`

Derived from instruction keys:

| Key pattern | UI |
|-------------|-----|
| `translate-english-persian-{mode}` | Translate · English → Persian · mode |
| `translate-persian-english-{mode}` | Translate · Persian → English · mode |
| `translate-…-scientific-{topic}` | Translate · Scientific · topic |
| `cursor-persian-english-{mode}` | Cursor · skill/agent |
| `frontend-{dir}-for-agent` | Frontend · for-agent |
| `refine-english-english-{style}` | Refine · style |
| `term-english-english-{style}` | Term · style |
| `compare-english-english` / `compare-persian-persian` | Compare · language |
| `grammar-english-english` / `grammar-persian-persian` | Grammar · language |
| `simplify-english-english` | Simplify |
| `symptoms-english-english` | Symptoms |

Create new modes/styles on the Instructions page (`POST /instructions` with `operation`, `direction`, `mode` / `style` / `language`, optional `topic`).

## Transform — `POST /api/v1/transform`

### Full request body shape

```json
{
  "operation": "translate|cursor|frontend|simplify|term|refine|symptoms|compare|grammar",
  "text": "...",
  "text1": "...",
  "text2": "...",
  "direction": "english-persian|persian-english|english-english",
  "mode": "<slug>",
  "topic": "<scientific topic>",
  "movie_name": "...",
  "language": "en|fa",
  "style": "<slug>"
}
```

Only include fields relevant to the selected operation. Modes/styles must exist as instruction keys.

### Operations

| Operation | Required fields | History type |
|-----------|-----------------|--------------|
| `translate` | `text`, `direction`, `mode` (+ `topic` if scientific; `movie_name` if movie) | `en_fa` / `fa_en` |
| `cursor` | `text`, `direction`, `mode` | `cursor` |
| `frontend` | `text`, `direction`, `mode` | `frontend` |
| `simplify` | `text` | `simplify` |
| `term` | `text`, `style` (`language` optional) | `term_en` / `term_fa` |
| `refine` | `text`, `style` | `refine` |
| `symptoms` | `text` | `symptoms` |
| `compare` | `text1`, `text2`, `language` | `compare_en` / `compare_fa` |
| `grammar` | `text`, `language` | `grammar_en` / `grammar_fa` |

### Compare

Compare two words or phrases. Do not send `text`.

```json
{
  "operation": "compare",
  "text1": "ask",
  "text2": "request",
  "language": "en"
}
```

- `language`: explanation language (`en` or `fa`)
- History `input_text` is stored as `"ask vs request"`
- Instruction keys: `compare-english-english`, `compare-persian-persian`

### Stats

`StatsBucket` includes: `simplify`, `en_fa`, `fa_en`, `term`, `refine`, `symptoms`, `compare`, `grammar`, `cursor`, `frontend`, `total`.
