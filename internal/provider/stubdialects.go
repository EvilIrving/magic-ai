package provider

// Stub SwitchWriters: harnesses whose adapter exists (skills/MCP projection)
// but whose model-provider dialect Magic cannot write. Each entry records what
// was checked and what is still missing — the reason it is a stub, not a TODO to
// "just write the file". A writer is added here only once the missing fact is
// first-hand verified.
//
// Source: upstream | @cline/core 0.2.0 dist/index.js (ProviderSettingsManager:
//   `<data dir>/settings/providers.json` = {version: 1, lastUsedProvider?,
//   modes, repairs?, providers: Record<id, {settings, updatedAt, tokenSource}>}
//   with updatedAt a required RFC3339 string; and the models.json registry
//   envelope {version: 1, providers: Record<id, {provider: {name, baseUrl,
//   defaultModelId?, protocol?, client?, capabilities?, modelsSourceUrl?},
//   models?, discoveredModelIds?}>}) | checked 2026-11-15 | verified
//   The credential (apiKey) and the selected provider live in providers.json,
//   not in models.json, so a models.json-only projection would register a
//   keyless provider; and providers.json's `updatedAt` belongs to the harness —
//   stamping it inside Files would need a clock, which makes Files impure and
//   plan's `after` (computed here) differ from what apply writes.
//   → cline: missing fact is a purity-preserving `updatedAt`. Needs a decision
//   (a Magic-owned timestamp field, or a ctx-supplied clock), not a guess.
//
// Source: vendor-doc | zcode.z.ai/en/docs/configuration | checked 2026-11-15 | verified
//   Model providers are added through Settings → Model Settings (custom
//   providers, Anthropic / OpenAI compatible); the only published statement about
//   the file itself is that in ~/.zcode/v2/config.json "a provider's options only
//   recognizes connection settings such as apiKey, baseURL, apiKeyRequired, and
//   headers". The provider entry itself (kind / models / which model is active)
//   and `~/.zcode/v2/provider_config.json` are not documented anywhere public.
//   → zcode: writing either file would be a guess about an Electron app's private
//   state; the documented subset (`options.*` alone) cannot select a model.

// stubReasons is the one-line answer `magic adapters` and `magic doctor` print
// for a harness whose SwitchWriter is still this stub: the short form of the
// missing fact, not a TODO. The long-form evidence stays in the Source: blocks
// above — a reason is added here only once that evidence exists.
//
// cline / zcode are the two stubs the Source: blocks above describe. gemini /
// copilot / trae are apply-only adapters.
//
// Source: internal | this file's Source: blocks (cline / zcode); switch.go's
// stub registration and docs/apply-design.md §Gemini/Copilot/Trae (apply-only,
// no third-party provider dialect) | checked 2026-11-14 | verified
var stubReasons = map[string]string{
	"cline":   "缺一个不改 Files 纯度的 updatedAt：凭据与选中 provider 在 harness 自己的 providers.json 里",
	"zcode":   "provider 条目（kind / models / 选中模型）没有公开格式，只能写 options.*，写不出选中模型",
	"gemini":  "只有 adapter 投影（MCP / hooks），没有稳定的第三方 provider 方言",
	"copilot": "没有公开的第三方 provider 方言；config.json 带 auth / runtime 状态，Magic 不镜像",
	"trae":    "没有公开的第三方 provider 方言，provider 只能在 IDE 设置里配",
}

// StubReason returns the short reason harness's provider write is not
// implemented. ok is false for a harness that is not a stub.
func StubReason(harness string) (string, bool) {
	r, ok := stubReasons[NormalizeHarness(harness)]
	return r, ok
}
