package modelsdev

// Catalog is the models.dev api.json root: provider id → provider record.
type Catalog map[string]Provider

// Provider is one models.dev provider entry.
type Provider struct {
	ID     string           `json:"id,omitempty"`
	Name   string           `json:"name,omitempty"`
	API    string           `json:"api,omitempty"` // upstream base URL when present
	NPM    string           `json:"npm,omitempty"`
	Doc    string           `json:"doc,omitempty"`
	Env    []string         `json:"env,omitempty"`
	Models map[string]Model `json:"models,omitempty"`
}

// Model is one models.dev model entry (fields Magic cares about).
type Model struct {
	ID               string            `json:"id,omitempty"`
	Name             string            `json:"name,omitempty"`
	ToolCall         *bool             `json:"tool_call,omitempty"`
	StructuredOutput *bool             `json:"structured_output,omitempty"`
	Reasoning        bool              `json:"reasoning,omitempty"`
	ReasoningOptions []ReasoningOption `json:"reasoning_options,omitempty"`
	Status           string            `json:"status,omitempty"`
	Attachment       bool              `json:"attachment,omitempty"`
	Limit            *Limit            `json:"limit,omitempty"`
	Cost             *Cost             `json:"cost,omitempty"`
	Modalities       *Modalities       `json:"modalities,omitempty"`
}

// ReasoningOption mirrors models.dev reasoning_options entries.
// Only type "effort" feeds thinkingLevelMap; toggle/budget_tokens are ignored for the map.
type ReasoningOption struct {
	Type   string   `json:"type"`
	Values []string `json:"values,omitempty"`
	Min    *int     `json:"min,omitempty"`
	Max    *int     `json:"max,omitempty"`
}

// Limit holds context/output token caps.
type Limit struct {
	Context int64 `json:"context,omitempty"`
	Output  int64 `json:"output,omitempty"`
}

// Cost is models.dev pricing in USD per million tokens.
type Cost struct {
	Input      float64 `json:"input,omitempty"`
	Output     float64 `json:"output,omitempty"`
	CacheRead  float64 `json:"cache_read,omitempty"`
	CacheWrite float64 `json:"cache_write,omitempty"`
}

// Modalities lists input/output modality strings.
type Modalities struct {
	Input  []string `json:"input,omitempty"`
	Output []string `json:"output,omitempty"`
}
