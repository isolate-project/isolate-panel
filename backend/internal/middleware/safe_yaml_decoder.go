package middleware

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

const (
	defaultMaxYAMLDepth     = 100
	defaultMaxYAMLSize      = 10 << 20
	defaultMaxYAMLStringLen = 1 << 20
	defaultMaxYAMLArrayLen  = 100_000
	defaultMaxYAMLKeyCount  = 10_000
)

// SafeYAMLDecoder enforces resource limits during YAML parsing.
// It decodes into a yaml.Node AST first, validates limits by walking the
// tree, then decodes the validated node into the target value.
type SafeYAMLDecoder struct {
	MaxDepth     int
	MaxSize      int64
	MaxStringLen int
	MaxArrayLen  int
	MaxKeyCount  int
}

func NewSafeYAMLDecoder() *SafeYAMLDecoder {
	return &SafeYAMLDecoder{
		MaxDepth:     defaultMaxYAMLDepth,
		MaxSize:      defaultMaxYAMLSize,
		MaxStringLen: defaultMaxYAMLStringLen,
		MaxArrayLen:  defaultMaxYAMLArrayLen,
		MaxKeyCount:  defaultMaxYAMLKeyCount,
	}
}

func (d *SafeYAMLDecoder) Decode(reader io.Reader, v interface{}) error {
	data, err := io.ReadAll(io.LimitReader(reader, d.MaxSize+1))
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if int64(len(data)) > d.MaxSize {
		return fmt.Errorf("request body exceeds max size %d bytes", d.MaxSize)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	if err := d.validateNode(&root, 0); err != nil {
		return err
	}

	if err := root.Decode(v); err != nil {
		return fmt.Errorf("YAML decode: %w", err)
	}
	return nil
}

func (d *SafeYAMLDecoder) validateNode(node *yaml.Node, depth int) error {
	if depth > d.MaxDepth {
		return fmt.Errorf("YAML depth %d exceeds max %d", depth, d.MaxDepth)
	}

	switch node.Kind {
	case yaml.ScalarNode:
		if node.Tag == "!!str" && len(node.Value) > d.MaxStringLen {
			return fmt.Errorf("YAML string length %d exceeds max %d", len(node.Value), d.MaxStringLen)
		}
	case yaml.MappingNode:
		if len(node.Content)/2 > d.MaxKeyCount {
			return fmt.Errorf("YAML key count %d exceeds max %d", len(node.Content)/2, d.MaxKeyCount)
		}
		for i := 0; i < len(node.Content); i += 2 {
			if err := d.validateNode(node.Content[i], depth+1); err != nil {
				return err
			}
			if err := d.validateNode(node.Content[i+1], depth+1); err != nil {
				return err
			}
		}
	case yaml.SequenceNode:
		if len(node.Content) > d.MaxArrayLen {
			return fmt.Errorf("YAML array length %d exceeds max %d", len(node.Content), d.MaxArrayLen)
		}
		for _, child := range node.Content {
			if err := d.validateNode(child, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}
