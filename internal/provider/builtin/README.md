# Built-in provider template

`providers.yaml` in this directory is the **product's default catalog**. It is
embedded into the Magic binary and never written to disk as a user file. It
exists so a new machine can reach a working provider set without hand-writing
`~/.magic-ai/providers.yaml` from vendor docs.

## What it is / is not

| | |
|---|---|
| is | default `name` / `key_env` / `endpoints[]` / suggested `pick` per provider |
| is | merge base for `magic provider add` and `magic apply` |
| is not | secrets (no key values, no `MAGIC_PASSPHRASE`) |
| is not | the user's live config (`~/.magic-ai/providers.yaml`) |
| is not | model metadata (that is `$MAGIC_HOME/registry/models.json`) |

## Merge semantics (planned)

The onboarding flow that consumes this template — `magic login <provider>` (fetch the
provider's live `/models`, let the user pick) — is specified in
[`docs/provider-onboarding.md`](../../../docs/provider-onboarding.md). Not implemented yet.

Merge happens in memory; nothing is written unless the user runs a command.

```
effective = merge(template, user_providers.yaml)
```

- Provider only in template → available as a default for `magic provider add <id>`.
- Provider in both → user wins:
  - scalar fields (`key_env`, `name`) — user value replaces template value
  - `endpoints[]` — if the user defines any, they replace the template list **whole**
    (no per-endpoint merging; avoids half-merged protocol sets)
  - `pick` — same rule: user list replaces template list whole
  - `models` — merged per role key, user wins per role
- Template never injects itself into `providers.yaml` silently.

## Schema

Mirrors `internal/catalog.Provider` plus two additive, tooling-only fields:

| field | required | meaning |
|---|---|---|
| `name` | no | display name; not projected to harnesses. Needs a `Name` field on `catalog.Provider` (small additive change). |
| `key_env` | yes | logical credential index (`<VENDOR>_API_KEY`) |
| `endpoints[].id` | yes | stable selector for `magic copy --endpoint` / `magic provider use` |
| `endpoints[].base_url` | yes | protocol-specific base URL |
| `endpoints[].protocol` | yes | `openai-chat-completions` \| `openai-responses` \| `anthropic-messages` |
| `pick` | no | suggested default model ids |
| `models` | no | Claude Code role → model id (`haiku`/`sonnet`/`opus`/`default`/`reasoning`/`subagent`) |
| `note` | no | human note; ignored by tooling |

`version` at the root is informational.

## Coverage

25 official providers, two groups. Relays and gateways are **not** shipped
here — they belong in the user's `providers.yaml`.

1. **国内官方** — `zai`, `zai-coding-cn`, `deepseek`,
   `kimi`/`kimi-cn`/`kimi-coding`, `minimax`/`minimax-cn`,
   `qwen-token-plan`/`-cn`, `xiaomi` + token plans.
2. **海外官方** — `anthropic`, `openai`, `xai`, `cerebras`,
   `openrouter`, `fireworks`, `nvidia`, `huggingface`, `baseten`,
   `ant-ling`, `vercel-ai-gateway`.

Deliberately absent until Magic grows the matching dialect or per-account
resolution: `google`/`google-vertex` (Generative AI), `mistral`
(Conversations), `amazon-bedrock` (Converse), `github-copilot` and
`cloudflare-ai-gateway` (per-account base URL), `opencode` Zen (router-chosen).

## Sources

- official entries: vendor docs, cross-checked against Pi's shipped catalog
  (`@earendil-works/pi-ai` `dist/providers/*.js`) and models.dev
- Z.AI / 智谱 coding endpoints: vendor docs (open.bigmodel.cn / api.z.ai)

## Secrets

`key_env` is an index into the secret store. See
[`docs/secrets-and-sync.md`](../../../docs/secrets-and-sync.md) (D2).

### Local (always plaintext — never `.enc`)

```
$MAGIC_HOME/secrets/providers.json       {"<KEY_ENV>": "<value>"}   0600
```

- Written by `magic secrets set <key_env> --value …`. No passphrase needed.
- Read directly by `magic copy` / `magic apply` / `magic provider`.
- Local is not an encrypted state. No `providers.json.enc`, no `MAGIC_SECRETS_MODE`.

### Cloud (`.enc` is the remote state)

`magic sync push` encrypts secret-bearing files and uploads them as `<rel>.enc`.
Everything else stays plaintext on the cloud. `magic sync pull` decrypts `.enc`
and writes local plaintext (no `.enc` on disk). The passphrase is needed only
for push/pull and never enters the library.

### Rules

- Values are never printed by `list`/`status`; only key names.
- `copy` resolves `key_env` (fallback: provider name); never reads process env.
- A provider whose `key_env` has no stored value still projects placeholders
  instead of failing apply.
