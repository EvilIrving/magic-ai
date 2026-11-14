package provider

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAML helpers for the dialect writers whose config is YAML (Hermes config.yaml,
// Oh My Pi models.yml). They edit a yaml.Node tree instead of a decoded map, so
// the user's comments, key order and unrelated keys survive the round trip.

// parseYAMLMap parses text into a document whose root is a mapping. An empty
// document yields an empty mapping.
func parseYAMLMap(text string) (*yaml.Node, error) {
	doc := &yaml.Node{}
	if strings.TrimSpace(text) != "" {
		if err := yaml.Unmarshal([]byte(text), doc); err != nil {
			return nil, err
		}
	}
	root := yamlDocumentRoot(doc)
	if root == nil || root.Kind == 0 {
		root = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		doc = &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}
	}
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("YAML document root is not a mapping")
	}
	return doc, nil
}

// encodeYAML renders a document with two-space indents.
func encodeYAML(doc *yaml.Node) ([]byte, error) {
	var b strings.Builder
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}

func yamlDocumentRoot(doc *yaml.Node) *yaml.Node {
	if doc == nil {
		return nil
	}
	if doc.Kind == yaml.DocumentNode {
		if len(doc.Content) == 0 {
			return nil
		}
		return doc.Content[0]
	}
	return doc
}

// yamlMappingChild returns the value node for key in a mapping, or nil.
func yamlMappingChild(m *yaml.Node, key string) *yaml.Node {
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

// yamlChildMap returns the mapping under key, creating it when absent. A key
// holding something other than a mapping is replaced: the dialect requires a
// mapping there (Hermes's `model:` ships as `""` until the first setup).
func yamlChildMap(m *yaml.Node, key string) *yaml.Node {
	if child := yamlMappingChild(m, key); child != nil && child.Kind == yaml.MappingNode {
		return child
	}
	fresh := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	yamlSetChild(m, key, fresh)
	return fresh
}

// yamlScalarString reads a scalar child as a string.
func yamlScalarString(m *yaml.Node, key string) (string, bool) {
	child := yamlMappingChild(m, key)
	if child == nil || child.Kind != yaml.ScalarNode {
		return "", false
	}
	return child.Value, true
}

// yamlSetString writes a string scalar child, keeping the key's position.
func yamlSetString(m *yaml.Node, key, value string) {
	yamlSetChild(m, key, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value})
}

// yamlSetValue writes any JSON-shaped value as a child node.
func yamlSetValue(m *yaml.Node, key string, value any) error {
	raw, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return err
	}
	node := yamlDocumentRoot(&doc)
	if node == nil {
		node = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
	}
	yamlSetChild(m, key, node)
	return nil
}

// yamlSetChild replaces key's value node in place, or appends the pair.
func yamlSetChild(m *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			value.HeadComment = m.Content[i+1].HeadComment
			value.LineComment = m.Content[i+1].LineComment
			value.FootComment = m.Content[i+1].FootComment
			m.Content[i+1] = value
			return
		}
	}
	m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, value)
}

// yamlDeleteKey removes key from a mapping; the comments attached to it go too.
func yamlDeleteKey(m *yaml.Node, key string) {
	if m == nil || m.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value != key {
			continue
		}
		m.Content = append(m.Content[:i], m.Content[i+2:]...)
		return
	}
}

// yamlMapKeys lists a mapping's keys in document order.
func yamlMapKeys(m *yaml.Node) []string {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	keys := make([]string, 0, len(m.Content)/2)
	for i := 0; i+1 < len(m.Content); i += 2 {
		keys = append(keys, m.Content[i].Value)
	}
	return keys
}
