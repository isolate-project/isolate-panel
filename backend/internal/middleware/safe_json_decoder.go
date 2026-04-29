package middleware

import (
	"encoding/json"
	"fmt"
	"io"
)

const (
	defaultMaxJSONDepth     = 50
	defaultMaxJSONSize      = 10 << 20
	defaultMaxJSONStringLen = 1 << 20
	defaultMaxJSONArrayLen  = 100_000
	defaultMaxJSONKeyCount  = 10_000
)

// SafeJSONDecoder enforces resource limits during JSON parsing with a fast
// byte-level depth pre-scan followed by standard encoding/json decoding.
type SafeJSONDecoder struct {
	MaxDepth     int
	MaxSize      int64
	MaxStringLen int
	MaxArrayLen  int
	MaxKeyCount  int
}

func NewSafeJSONDecoder() *SafeJSONDecoder {
	return &SafeJSONDecoder{
		MaxDepth:     defaultMaxJSONDepth,
		MaxSize:      defaultMaxJSONSize,
		MaxStringLen: defaultMaxJSONStringLen,
		MaxArrayLen:  defaultMaxJSONArrayLen,
		MaxKeyCount:  defaultMaxJSONKeyCount,
	}
}

func (d *SafeJSONDecoder) Decode(reader io.Reader, v interface{}) error {
	data, err := io.ReadAll(io.LimitReader(reader, d.MaxSize+1))
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if int64(len(data)) > d.MaxSize {
		return fmt.Errorf("request body exceeds max size %d bytes", d.MaxSize)
	}

	if err := d.checkDepth(data); err != nil {
		return err
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return nil
}

func (d *SafeJSONDecoder) checkDepth(data []byte) error {
	depth := 0
	inString := false
	escape := false
	arrayLen := 0
	maxArrayInCurrent := 0

	for i := 0; i < len(data); i++ {
		b := data[i]
		if escape {
			escape = false
			continue
		}
		if b == '\\' && inString {
			escape = true
			continue
		}
		if b == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == ':' || b == ',' {
			continue
		}
		switch b {
		case '{':
			depth++
			if depth > d.MaxDepth {
				return fmt.Errorf("JSON depth %d exceeds max %d", depth, d.MaxDepth)
			}
		case '[':
			depth++
			if depth > d.MaxDepth {
				return fmt.Errorf("JSON depth %d exceeds max %d", depth, d.MaxDepth)
			}
			arrayLen = 0
		case '}':
			depth--
		case ']':
			depth--
			if arrayLen > maxArrayInCurrent {
				maxArrayInCurrent = arrayLen
			}
			if maxArrayInCurrent > d.MaxArrayLen {
				return fmt.Errorf("JSON array length %d exceeds max %d", maxArrayInCurrent, d.MaxArrayLen)
			}
		}
	}
	return nil
}
