package provider

import "testing"

func TestParseModelsResponseOpenAI(t *testing.T) {
	body := []byte(`{"object":"list","data":[{"id":"gpt-5.6-sol","object":"model"},{"id":"gpt-5.6-luna","object":"model"},{"id":"gpt-5.6-sol"}]}`)
	models, err := ParseModelsResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 {
		t.Fatalf("want 2 unique models, got %+v", models)
	}
	if models[0].ID != "gpt-5.6-luna" || models[1].ID != "gpt-5.6-sol" {
		t.Fatalf("not sorted/deduped: %+v", models)
	}
}

func TestParseModelsResponseAnthropicDisplayName(t *testing.T) {
	body := []byte(`{"data":[{"id":"claude-sonnet-4-5","display_name":"Claude Sonnet 4.5","type":"model"}]}`)
	models, err := ParseModelsResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].ID != "claude-sonnet-4-5" || models[0].Name != "Claude Sonnet 4.5" {
		t.Fatalf("%+v", models)
	}
}

func TestParseModelsResponseBareArrayAndStrings(t *testing.T) {
	body := []byte(`["a","b",{"model":"c"}]`)
	models, err := ParseModelsResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 3 {
		t.Fatalf("%+v", models)
	}
}

func TestParseModelsResponseInvalid(t *testing.T) {
	if _, err := ParseModelsResponse([]byte("not json")); err == nil {
		t.Fatal("expected parse error")
	}
	if models, err := ParseModelsResponse([]byte(`{"object":"list"}`)); err != nil || len(models) != 0 {
		t.Fatalf("empty list should be ok: %+v %v", models, err)
	}
}
