package mcp

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// upsertYAMLSection merges entries under topKey (a top-level mapping key) into
// a YAML document, preserving every other key and its comments via yaml.Node
// round-tripping. Missing file → a fresh document with just the managed key.
func upsertYAMLSection(text, topKey string, entries map[string]map[string]any) ([]byte, error) {
	var doc yaml.Node
	if strings.TrimSpace(text) != "" {
		if err := yaml.Unmarshal([]byte(text), &doc); err != nil {
			return nil, err
		}
	}
	root := documentRoot(&doc)
	if root == nil || root.Kind == 0 {
		root = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}
	}
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("YAML document root is not a mapping")
	}
	section := mappingValue(root, topKey)
	if section == nil {
		section = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: topKey}, section)
	}
	if section.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("YAML key %q is not a mapping", topKey)
	}
	for _, name := range sortedKeys(entries) {
		node, err := toYAMLNode(entries[name])
		if err != nil {
			return nil, err
		}
		replaced := false
		for i := 0; i+1 < len(section.Content); i += 2 {
			if section.Content[i].Value == name {
				section.Content[i+1] = node
				replaced = true
				break
			}
		}
		if !replaced {
			section.Content = append(section.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name}, node)
		}
	}
	var b strings.Builder
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}

// documentRoot returns the document's top node, or nil for an empty document.
func documentRoot(doc *yaml.Node) *yaml.Node {
	if doc.Kind == yaml.DocumentNode {
		if len(doc.Content) == 0 {
			return nil
		}
		return doc.Content[0]
	}
	return doc
}

// mappingValue returns the value node for key in a mapping, or nil.
func mappingValue(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// toYAMLNode converts a JSON-shaped value into a fresh YAML node.
func toYAMLNode(v any) (*yaml.Node, error) {
	raw, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	root := documentRoot(&doc)
	if root == nil {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}, nil
	}
	return root, nil
}
