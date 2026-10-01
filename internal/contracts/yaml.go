package contracts

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var metadataInt = regexp.MustCompile(`^(0|[1-9][0-9]{0,8})$`)

// ParseYAML decodes the restricted, JSON-compatible YAML subset used by
// contracts: mappings with string keys, sequences, strings, booleans, null,
// and small non-negative metadata integers. Aliases, anchors, merge keys,
// custom tags, duplicate keys, floats, and multiple documents are rejected.
func ParseYAML(raw []byte) (any, error) {
	dec := yaml.NewDecoder(strings.NewReader(string(raw)))
	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		return nil, err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 {
		return nil, errors.New("expected exactly one YAML document")
	}
	v, err := convert(doc.Content[0])
	if err != nil {
		return nil, err
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, errors.New("multiple YAML documents are not allowed")
		}
		return nil, err
	}
	return v, nil
}

func convert(n *yaml.Node) (any, error) {
	if n.Kind == yaml.AliasNode || n.Anchor != "" {
		return nil, fmt.Errorf("line %d: anchors and aliases are not allowed", n.Line)
	}
	switch n.Kind {
	case yaml.MappingNode:
		if n.Tag != "!!map" {
			return nil, fmt.Errorf("line %d: tag %s is not allowed", n.Line, n.Tag)
		}
		m := make(map[string]any, len(n.Content)/2)
		for i := 0; i+1 < len(n.Content); i += 2 {
			k := n.Content[i]
			if k.Kind != yaml.ScalarNode || k.Tag != "!!str" {
				return nil, fmt.Errorf("line %d: mapping keys must be strings", k.Line)
			}
			if k.Value == "<<" && k.Style == 0 {
				return nil, fmt.Errorf("line %d: merge keys are not allowed", k.Line)
			}
			if _, dup := m[k.Value]; dup {
				return nil, fmt.Errorf("line %d: duplicate key %q", k.Line, k.Value)
			}
			v, err := convert(n.Content[i+1])
			if err != nil {
				return nil, err
			}
			m[k.Value] = v
		}
		return m, nil
	case yaml.SequenceNode:
		if n.Tag != "!!seq" {
			return nil, fmt.Errorf("line %d: tag %s is not allowed", n.Line, n.Tag)
		}
		s := make([]any, 0, len(n.Content))
		for _, c := range n.Content {
			v, err := convert(c)
			if err != nil {
				return nil, err
			}
			s = append(s, v)
		}
		return s, nil
	case yaml.ScalarNode:
		switch n.Tag {
		case "!!str":
			return n.Value, nil
		case "!!bool":
			switch n.Value {
			case "true":
				return true, nil
			case "false":
				return false, nil
			}
			return nil, fmt.Errorf("line %d: write Booleans as true or false", n.Line)
		case "!!null":
			if n.Value != "null" && n.Value != "~" && n.Value != "" {
				return nil, fmt.Errorf("line %d: write null as null", n.Line)
			}
			return nil, nil
		case "!!int":
			if !metadataInt.MatchString(n.Value) {
				return nil, fmt.Errorf("line %d: integer %q must be a small non-negative decimal; quote test values as strings", n.Line, n.Value)
			}
			v, _ := strconv.ParseInt(n.Value, 10, 64)
			return v, nil
		case "!!float":
			return nil, fmt.Errorf("line %d: floating-point scalars are not allowed; quote the value", n.Line)
		}
		return nil, fmt.Errorf("line %d: tag %s is not allowed", n.Line, n.Tag)
	}
	return nil, fmt.Errorf("line %d: unsupported YAML node", n.Line)
}
