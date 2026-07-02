package resource

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// document pairs a parsed yaml.Node with its kind and name so callers can
// decode the full spec into the correct typed struct.
type document struct {
	kind string
	name string
	node *yaml.Node
}

// parseFile reads all YAML documents from path and returns them as documents.
// Multi-document files (separated by ---) are fully supported.
func parseFile(path string) ([]document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q: %w", path, err)
	}
	return parseBytes(data)
}

// parseBytes parses one or more YAML documents from raw bytes.
func parseBytes(data []byte) ([]document, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var docs []document

	for {
		var node yaml.Node
		if err := dec.Decode(&node); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("YAML parse error: %w", err)
		}

		// Skip empty documents (--- with nothing after).
		if node.Kind == 0 {
			continue
		}

		// Decode just the envelope to get kind and name.
		var env envelope
		if err := node.Decode(&env); err != nil {
			return nil, fmt.Errorf("failed to decode resource envelope: %w", err)
		}
		if env.Kind == "" && env.APIVersion == "" && env.Metadata.Name == "" {
			continue // skip truly empty / comment-only documents
		}
		if env.Kind == "" {
			return nil, fmt.Errorf("resource is missing required field 'kind'")
		}
		if env.Metadata.Name == "" {
			return nil, fmt.Errorf("resource of kind %q is missing metadata.name", env.Kind)
		}

		docs = append(docs, document{
			kind: env.Kind,
			name: env.Metadata.Name,
			node: &node,
		})
	}

	return docs, nil
}
