package modelsdev

import (
	"testing"
)

func TestEffortThinkingLevelMap_Basic(t *testing.T) {
	m := EffortThinkingLevelMap([]ReasoningOption{
		{Type: "effort", Values: []string{"minimal", "low", "medium", "high"}},
	})
	if m == nil {
		t.Fatal("expected map")
	}
	if m["off"] != nil {
		t.Fatalf("off want nil, got %v", m["off"])
	}
	for _, level := range []string{"minimal", "low", "medium", "high"} {
		if m[level] == nil || *m[level] != level {
			t.Fatalf("%s=%v", level, m[level])
		}
	}
	for _, level := range []string{"xhigh", "max"} {
		if m[level] != nil {
			t.Fatalf("%s want nil", level)
		}
	}
}

func TestEffortThinkingLevelMap_NoneMapsToOff(t *testing.T) {
	m := EffortThinkingLevelMap([]ReasoningOption{
		{Type: "effort", Values: []string{"none", "low", "high", "max"}},
	})
	if m == nil || m["off"] == nil || *m["off"] != "none" {
		t.Fatalf("off=%v", m["off"])
	}
	if m["minimal"] != nil {
		t.Fatal("minimal should be null")
	}
	if m["low"] == nil || *m["low"] != "low" {
		t.Fatal("low")
	}
}

func TestEffortThinkingLevelMap_IgnoresToggleAndDefault(t *testing.T) {
	if EffortThinkingLevelMap([]ReasoningOption{{Type: "toggle"}}) != nil {
		t.Fatal("toggle alone must not produce map")
	}
	min := 1024
	if EffortThinkingLevelMap([]ReasoningOption{{Type: "budget_tokens", Min: &min}}) != nil {
		t.Fatal("budget alone must not produce map")
	}
	one := 1
	m := EffortThinkingLevelMap([]ReasoningOption{
		{Type: "toggle"},
		{Type: "effort", Values: []string{"default", "low", "high"}},
		{Type: "budget_tokens", Min: &one},
	})
	if m == nil {
		t.Fatal("expected map from effort")
	}
	if m["low"] == nil || *m["low"] != "low" {
		t.Fatal("low")
	}
	if _, ok := m["default"]; ok {
		t.Fatal("default must not be a map key")
	}
}

func TestEffortThinkingLevelMap_EmptyEffort(t *testing.T) {
	if EffortThinkingLevelMap(nil) != nil {
		t.Fatal("nil options")
	}
	if EffortThinkingLevelMap([]ReasoningOption{{Type: "effort", Values: []string{"default", ""}}}) != nil {
		t.Fatal("only default/empty")
	}
}

func TestOpenRouterThinkingLevelMap_Mandatory(t *testing.T) {
	m := OpenRouterThinkingLevelMap([]string{"low", "high"}, true)
	if m == nil || m["off"] != nil {
		t.Fatalf("mandatory off must be null: %v", m["off"])
	}
	if m["low"] == nil || *m["low"] != "low" {
		t.Fatal("low")
	}
	m2 := OpenRouterThinkingLevelMap(nil, true)
	if m2 == nil || m2["off"] != nil {
		t.Fatal("empty+mandatory → {off:null}")
	}
	if OpenRouterThinkingLevelMap(nil, false) != nil {
		t.Fatal("empty+not mandatory → nil")
	}
}
