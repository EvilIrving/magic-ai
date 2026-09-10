package provider

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"magic/internal/catalog"
)

// TestGateway GETs url with API key. authMode: "bearer" | "x-api-key".
// Harness-agnostic — used by magic provider test against the selected provider endpoint.
func TestGateway(client *http.Client, url, apiKey, authMode string) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "magic-provider-test/1.0")
	switch strings.ToLower(authMode) {
	case "x-api-key", "anthropic":
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	default:
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gateway test failed: could not reach %s: %w", url, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	switch resp.StatusCode {
	case 200:
		return fmt.Sprintf("Gateway reachable and accepted the API key (HTTP %d): %s", resp.StatusCode, url), nil
	case 401, 403:
		return "", fmt.Errorf("gateway rejected the API key (HTTP %d): %s", resp.StatusCode, url)
	default:
		return "", fmt.Errorf("gateway test failed (HTTP %d): %s", resp.StatusCode, url)
	}
}

// TestProvider resolves target against library and GETs {base}/models (or anthropic-style).
func TestProvider(ctx Context, rawTarget string) (string, error) {
	f, err := ctx.File()
	if err != nil {
		return "", err
	}
	rt, err := ResolveUseTarget(f, rawTarget)
	if err != nil {
		return "", err
	}
	if rt.Mode == ModeOfficial {
		return "", fmt.Errorf("official mode has no third-party gateway to test — pass a provider name or company (active)")
	}
	ep, err := PreferredEndpoint(rt.Provider, "") // any endpoint
	if err != nil {
		ep, err = PreferredEndpoint(rt.Provider, HarnessCodex)
		if err != nil {
			return "", err
		}
	}
	keyEnv := strings.TrimSpace(rt.Provider.KeyEnv)
	if keyEnv == "" {
		keyEnv = rt.Name
	}
	key, err := ResolveSecret(ctx.SecretsDir(), keyEnv)
	if err != nil {
		return "", err
	}
	url, auth := modelsURL(ep)
	return TestGateway(ctx.HTTP, url, key, auth)
}

func modelsURL(ep catalog.Endpoint) (url, auth string) {
	base := strings.TrimRight(strings.TrimSpace(ep.BaseURL), "/")
	proto := catalog.NormalizeProtocol(ep.Protocol)
	auth = "bearer"
	if proto == catalog.ProtocolAnthropicMessages {
		auth = "x-api-key"
		// anthropic base may already end with /anthropic
		if strings.HasSuffix(base, "/anthropic") {
			return base + "/v1/models", auth
		}
		return base + "/v1/models", auth
	}
	return base + "/models", auth
}

// CheapResponsesPing POSTs a minimal /responses body when key present (optional Pi probe).
func CheapResponsesPing(client *http.Client, baseURL, apiKey, model string) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	if model == "" {
		model = "gpt-5.6-sol"
	}
	url := strings.TrimRight(baseURL, "/") + "/responses"
	body := fmt.Sprintf(`{"model":%q,"input":"ping","max_output_tokens":16}`, model)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("responses ping failed: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return fmt.Sprintf("responses ping ok (HTTP %d)", resp.StatusCode), nil
	}
	return "", fmt.Errorf("responses ping failed (HTTP %d)", resp.StatusCode)
}
