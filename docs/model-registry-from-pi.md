# Magic model registry — design from Pi’s catalog pipeline

Status: **normalize + apply enrichment implemented** (2026-09-09). `magic registry sync --source models.dev|openrouter|all` writes Pi-shaped `$MAGIC_HOME/registry/models.json` (thinkingLevelMap, costs, api/baseUrl). `registry search|show` / `catalog pick` prefer it. Apply / `provider use` enrich Pi (full model fields), Grok (context/reasoning), Claude (`CLAUDE_CODE_*`), Codex (`models.json` + effort). OpenRouter = slim-cache + optional enrichment. `aliases.json` / patches / meta etags still **FUTURE**.

## Why copy Pi

Pi’s `@earendil-works/pi-ai` package builds a rich model catalog with the fields Magic needs for relay picks, cost UX, and thinking/effort controls:

| Field | Role |
|-------|------|
| `id` | Provider-local model id (OpenRouter keeps `org/model`) |
| `api` | Wire dialect (`openai-completions`, `openai-responses`, `anthropic-messages`, …) |
| `provider` | Catalog namespace (`openai`, `openrouter`, `anthropic`, …) |
| `baseUrl` | Default upstream base URL for that catalog entry |
| `reasoning` | Boolean: model advertises reasoning |
| `input[]` | Modalities, typically `text` and optionally `image` |
| `cost{input,output,cacheRead,cacheWrite}` | USD per **million** tokens (tiers optional later) |
| `contextWindow` / `maxTokens` | Context and max output |
| `thinkingLevelMap` | Maps Magic/Pi levels (`off|minimal|low|medium|high|xhigh|max`) → provider effort string, or `null` = unsupported |

Magic today only caches a slim OpenRouter list (`internal/openrouter`). That is useful for search/pick, but too thin for thinking maps, stable costs, or multi-provider identity. Pi shows how to treat **models.dev as the primary metadata source** and OpenRouter as a **supplement / live overlay**.

## How Pi generates the catalog (study summary)

Primary sources (build-time script `packages/ai/scripts/generate-models.ts`):

1. **models.dev** — `GET https://models.dev/api.json`  
   Shape: `{ [providerId]: { models: { [modelId]: ModelsDevModel } } }`.  
   Relevant model fields: `id`, `name`, `tool_call`, `reasoning`, `reasoning_options[]`, `limit.{context,output}`, `cost.{input,output,cache_read,cache_write,tiers?}`, `modalities.input[]`, `structured_output`, `status`.
2. **OpenRouter** — `GET https://openrouter.ai/api/v1/models`  
   Pi keeps only models with `supported_parameters` including `"tools"`. Pricing is per-token strings → ×1e6 → `$/MTok`. Reasoning → `thinkingLevelMap` via OpenRouter metadata.
3. **Other live lists** (Pi-only; Magic need not copy initially): Vercel AI Gateway, NVIDIA NIM id probe.

Pipeline:

```
fetch models.dev + OpenRouter (+ …)
  → per-provider hard-coded api/baseUrl mapping + filter tool_call
  → record reasoning_options keyed by provider:id
  → hand patches (cost/context/maxTokens/compat quirks)
  → applyModelsDevReasoningOptionMetadata  // effort → thinkingLevelMap
  → applyThinkingLevelMetadata             // Pi-only overrides
  → apply*CompatMetadata                   // openai/anthropic quirks
  → dedupe by (provider, id); models.dev wins over OpenRouter
  → emit provider JSON shards + models.generated.ts
```

### thinkingLevelMap mapping (copy this)

From `models-dev-reasoning-options.ts`:

- Input: `reasoning_options` entries with `type: "effort"` and `values: ("none"|"minimal"|"low"|"medium"|"high"|"xhigh"|"max"|"default"|null)[]`.
- Output map keys: `off`, `minimal`, `low`, `medium`, `high`, `xhigh`, `max`.
- Rule: for each Pi level, if present in `values` → map to itself; else `null`.  
  `off` → `"none"` if `"none"` is supported, else `null`.
- Ignore `default` and JSON `null` values (no Pi equivalent).
- Toggle / `budget_tokens` options alone do **not** produce a map; Pi only materializes effort maps when the model’s API can send effort directly (`supportsDirectReasoningEffort`).

From `openrouter-reasoning-options.ts`:

- Uses the same effort converter on `reasoning.supported_efforts`.
- If `mandatory === true`, force `off: null` (cannot disable).
- Empty `supported_efforts` + mandatory → `{ off: null }`; otherwise undefined.

### Build-time vs runtime refresh (do not conflate)

| | Build-time (`generate-models.ts`) | Runtime (`models-store.json`) |
|--|-----------------------------------|-------------------------------|
| Who | Package maintainers / CI | Coding-agent process |
| Source | models.dev + OpenRouter (+ patches) | Per-provider **pi.dev** overlay `GET /api/models/providers/:id` |
| Output | Checked-in `src/providers/data/*.json` + generated TS | `~/.pi/agent/models-store.json` keyed by provider |
| Cadence | Manual / release | Soft refresh every **4h** (`REMOTE_CATALOG_REFRESH_INTERVAL_MS`), ETag/`If-None-Match`, `Last-Modified` gate vs local build timestamp |
| Merge | models.dev priority at generation | Overlay replaces/extends **same id** only if remote `lastModified` > local generated-at |

Magic should mirror the **build-time normalize** story as a user-invoked sync CLI, not Pi’s pi.dev CDN overlay. Optional later: ETag caching when re-fetching models.dev (the CDN already returns `etag`).

## Magic target architecture

### Data sources and refresh cadence

`magic registry sync` becomes multi-source:

| `--source` | Behavior |
|------------|----------|
| `models.dev` (new default long-term; stub exists) | Download raw `https://models.dev/api.json` → `$MAGIC_HOME/registry/models.dev.json` |
| `openrouter` (current default for back-compat) | Existing slim cache → `openrouter-models.json` |
| `all` | Both raw fetches; then normalize → `models.json` (OR enrich) |

Recommended product cadence:

- **On demand**: `magic registry sync` (and `sync --source all` before catalog pick UX).
- **Suggested TTL**: treat raw files older than **24h** as stale in `registry search/show` warnings (do not auto-network without user intent).
- **Optional cron/docs**: weekly `sync --source all` for operators.
- OpenRouter remains **optional** enrichment (live tool-capable ids, OR-specific pricing, OR reasoning metadata), never the only truth.

### On-disk layout under `$MAGIC_HOME/registry/`

```
$MAGIC_HOME/registry/
  models.dev.json          # raw models.dev api.json (authoritative upstream dump)
  openrouter-models.json   # existing slim OpenRouter cache (keep)
  models.json              # normalized Magic catalog ({version,source,synced_at,models:[]})
  aliases.json             # FUTURE: relay short-id → canonical registry id
  patches.json             # FUTURE: Magic-owned overrides (cost/context/thinking)
  meta.json                # FUTURE: synced_at, etags, source urls, schema_version
```

**Implemented:** raw dumps + normalized `models.json`. Keep `openrouter-models.json` as fallback for search/show/pick when normalize cache missing. `aliases.json` / `patches.json` / `meta.json` remain FUTURE files (do not invent readers).

### Normalized Magic model schema (`models.json`)

Each entry (JSON object):

```json
{
  "id": "claude-opus-4-6",
  "name": "Claude Opus 4.6",
  "api": "anthropic-messages",
  "provider": "anthropic",
  "baseUrl": "https://api.anthropic.com",
  "reasoning": true,
  "input": ["text", "image"],
  "cost": {
    "input": 5,
    "output": 25,
    "cacheRead": 0.5,
    "cacheWrite": 6.25
  },
  "contextWindow": 1000000,
  "maxTokens": 128000,
  "thinkingLevelMap": {
    "off": null,
    "minimal": null,
    "low": "low",
    "medium": "medium",
    "high": "high",
    "xhigh": null,
    "max": "max"
  },
  "source": "models.dev",
  "modelsDevProvider": "anthropic"
}
```

Notes:

- Costs are **$/MTok**, same as Pi (`cache_read` → `cacheRead`).
- `source` is Magic-only provenance (`models.dev` | `openrouter` | `patch`).
- Magic **include-all** models (unlike Pi’s tool_call filter); record `toolCall` when present.
- Default fallbacks: `contextWindow`/`maxTokens` → `4096` when limits missing.

### Mapping rules: models.dev → Magic

1. **Identity**: `provider` = models.dev top-level key (or remapped via Magic provider table); `id` = models.dev model key.
2. **api + baseUrl**: maintain a small **ProviderDefaults** table in Go (not 3k lines of Pi quirks). Start with the protocols Magic already understands:
   - `openai` → `openai-responses` / `https://api.openai.com/v1`
   - `anthropic` → `anthropic-messages` / `https://api.anthropic.com`
   - `google` → skip or mark unsupported until Magic has a Gemini dialect
   - `xai`, `groq`, `deepseek`, `moonshotai`, `openrouter`, … → `openai-completions` (or responses where known) + known base URL
   - Unknown providers: still emit metadata with `api: ""` / empty `baseUrl` so search works; mark `unsupported: true`.
3. **reasoning**: `m.reasoning == true`.
4. **input**: `["text"]`, plus `"image"` if `modalities.input` contains `image`.
5. **cost**: `getModelsDevCost` — copy base rates; optional `tiers` later.
6. **contextWindow / maxTokens**: `limit.context` / `limit.output`.
7. **thinkingLevelMap**: port `getEffortThinkingLevelMap` literally; apply only when Magic will send effort for that `api` (initially: openai-responses, openai-completions with reasoning, anthropic adaptive — can start permissive and refine).
8. **OpenRouter merge** (optional second pass): convert OR models with tools; set `provider: "openrouter"`, full `id`, dual baseUrl (Anthropic messages path for `anthropic/*`); fill map via `getOpenRouterThinkingLevelMap`. Do **not** overwrite an existing `(provider,id)` from models.dev.

### Alias layer for relay short ids

Relay providers (e.g. seeded `shu`) use **short picks** (`gpt-5.6-sol`, `deepseek-v4-pro`) that are not OpenRouter ids (`openai/…`) and may not match models.dev keys 1:1.

`$MAGIC_HOME/registry/aliases.json`:

```json
{
  "schema_version": 1,
  "aliases": {
    "gpt-5.6-sol": { "provider": "openai", "id": "gpt-5.6-sol" },
    "glm-4.5": { "provider": "openrouter", "id": "z-ai/glm-4.5" },
    "kimi-k2.5": { "provider": "moonshotai", "id": "kimi-k2.5" }
  }
}
```

Resolution order for `registry show` / `catalog pick` / future apply:

1. Exact `(provider, id)` in `models.json` if caller passes provider.
2. Exact `id` unique match in `models.json`.
3. `aliases.json` short → canonical.
4. OpenRouter slim cache (legacy).
5. Miss → clear error suggesting `magic registry sync --source all`.

Seeded `providers.yaml` picks stay short; aliases keep the library ergonomic without baking OpenRouter ids into every relay profile.

### What NOT to copy

Leave in a Magic `patches.json` (or skip entirely) rather than porting Pi’s generator bulk:

- Pi `compat` blobs (`OpenAICompletionsCompat`, Anthropic mid-convo effort, grammar tools, tool-search, explicit prompt cache).
- Provider-specific thinking overrides (`applyThinkingLevelMetadata` walls of Claude/GPT/Gemini/Grok/Together/Fireworks/OpenCode exceptions).
- Hand-added missing models (gpt-6-astra, deepseek-v4-*, ant-ling, openai-codex clones, Azure clones).
- Copilot / Bedrock / Vertex / Cloudflare Workers AI / AI Gateway / NVIDIA NIM live probes.
- Build-time TypeScript codegen / model-data manifests.
- Runtime **pi.dev** remote catalog overlay and `models-store.json` semantics.

Magic’s job is a **portable metadata registry** for catalog/copy/apply UX — not a full Pi streaming compat matrix.

## Evolving `internal/openrouter`

**Now (2026-09-09):**

- `internal/openrouter` = OpenRouter **source adapter** only (slim cache).
- `internal/modelsdev` + `internal/modelfile` = primary normalize path → `models.json`.
- CLI: `registry sync|search|show`; `catalog pick` / apply enrichment prefer `models.json`, fall back to OpenRouter slim.
- Do not delete `openrouter-models.json` (fallback + `--source all` enrichment).
- Still FUTURE: `aliases.json` resolve, `patches.json`, `meta.json` etags.

## Implementation map (normalize done; aliases still next)

Suggested files:

| Path | Purpose |
|------|---------|
| `internal/modelsdev/fetch.go` | Download/write raw `models.dev.json` (**stub done**) |
| `internal/modelsdev/types.go` | Unmarshal structs for models.dev providers/models |
| `internal/modelsdev/reasoning.go` | Port `getEffortThinkingLevelMap` (+ OR helper) |
| `internal/modelsdev/defaults.go` | Provider → api/baseUrl table (Magic-sized) |
| `internal/modelfile/model.go` | Normalized `Model` + `Cost` + `ThinkingLevelMap` types |
| `internal/modelfile/normalize.go` | models.dev (+ optional OR) → `[]Model` |
| `internal/modelfile/aliases.go` | Load/resolve `aliases.json` |
| `internal/modelfile/patches.go` | Apply `patches.json` overrides |
| `cmd/magic/registry.go` | `--source models.dev\|openrouter\|all`; later `normalize` substep |
| `testdata/registry/` | Tiny fixtures for normalize unit tests |

Suggested CLI progression:

1. ~~`sync --source models.dev` raw download~~
2. ~~`sync --source all` raw both~~
3. ~~Internal `Normalize(raw) → models.json` behind `sync`~~
4. ~~Point `search`/`show`/`catalog pick` at normalized~~ (aliases still FUTURE)
5. Document TTL / `meta.json` etags (still FUTURE)

Out of scope for the next pass: Pi compat metadata, auto background refresh, shipping a giant checked-in catalog inside the Magic binary.

## Implemented (normalize pass)

- `internal/modelsdev` — raw Sync + types + `EffortThinkingLevelMap` + `KnownProviders` defaults table.
- `internal/modelfile` — normalized `Model` / `Catalog`, `Normalize`, Search/FindFuzzy, models.json I/O.
- CLI: `magic registry sync --source models.dev|openrouter|all` → after fetch, write `models.json`.
  - `models.dev` / `all`: always normalize (all enriches from OpenRouter slim cache; models.dev wins on same provider/id).
  - `openrouter` (default): existing slim sync; re-normalize from existing `models.dev.json` if present.
- `registry search|show` and `catalog pick` prefer `models.json`, fall back to OpenRouter cache.
- Include-all models (unlike Pi’s tool_call filter); `toolCall` field recorded when present.
- **Apply / provider-use enrichment** (same pass): each pick → `modelfile.ResolvePickMeta` / `FindFuzzy` on `models.json`:
  - **Pi**: full model row fields (`contextWindow` / `maxTokens` / `thinkingLevelMap` / `cost` / `reasoning` / `input`); `apiKey` stays `$KEY_ENV`.
  - **Grok**: `context_window` / `max_completion_tokens` / `name` / `reasoning_efforts` from meta.
  - **Claude**: role envs + `CLAUDE_CODE_EFFORT_LEVEL` / `CLAUDE_CODE_AUTO_COMPACT_WINDOW` from meta.
  - **Codex**: native `models.json` fields + reasoning effort; third-party only sets `model_catalog_json`.
  - Missing meta → safe defaults; never fail apply.
- Still FUTURE: `aliases.json`, `patches.json`, `meta.json` etags, Pi compat quirks.

### ProviderDefaults map (documented)

| provider | api | baseUrl |
|----------|-----|---------|
| openai | openai-responses | https://api.openai.com/v1 |
| anthropic | anthropic-messages | https://api.anthropic.com |
| deepseek | openai-completions | https://api.deepseek.com (or models.dev `provider.api`) |
| xai | openai-responses | https://api.x.ai/v1 |
| groq | openai-completions | https://api.groq.com/openai/v1 |
| openrouter | openai-completions | https://openrouter.ai/api/v1 |
| moonshotai / moonshotai-cn | openai-completions | moonshot.ai / moonshot.cn `/v1` |
| cerebras, together, fireworks-ai, huggingface | openai-completions | known URLs |
| minimax / minimax-cn / kimi-for-coding | anthropic-messages | known URLs |
| google / google-vertex | *(empty — unsupported in Magic)* | Gemini URL when known |
| unknown | npm heuristic or empty + `unsupported: true` | models.dev `provider.api` if any |
