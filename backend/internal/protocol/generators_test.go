package protocol

import (
	"strings"
	"testing"
)

func TestGenerateUUIDv4_Format(t *testing.T) {
	uuid := GenerateUUIDv4()

	// UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	if len(uuid) != 36 {
		t.Fatalf("Expected UUID length 36, got %d", len(uuid))
	}

	// Check format with regex-like validation
	parts := strings.Split(uuid, "-")
	if len(parts) != 5 {
		t.Fatalf("Expected 5 parts, got %d", len(parts))
	}

	// Part lengths: 8-4-4-4-12
	expectedLengths := []int{8, 4, 4, 4, 12}
	for i, expected := range expectedLengths {
		if len(parts[i]) != expected {
			t.Errorf("Part %d: expected length %d, got %d", i, expected, len(parts[i]))
		}
	}

	// Version 4 check (3rd part should start with 4)
	if parts[2][0] != '4' {
		t.Errorf("Expected version 4 (3rd part starts with 4), got %c", parts[2][0])
	}

	// Variant check (4th part should start with 8, 9, a, or b)
	variant := parts[3][0]
	if variant < '8' || variant > 'b' {
		t.Errorf("Expected variant 8-b, got %c", variant)
	}
}

func TestGeneratePassword_Length(t *testing.T) {
	tests := []int{16, 32, 64, 128}

	for _, length := range tests {
		t.Run(string(rune(length)), func(t *testing.T) {
			password, err := GeneratePassword(length)
			if err != nil {
				t.Fatalf("GeneratePassword(%d) failed: %v", length, err)
			}
			if len(password) != length {
				t.Errorf("Expected password length %d, got %d", length, len(password))
			}
		})
	}
}

func TestGeneratePassword_Uniqueness(t *testing.T) {
	generated := make(map[string]bool)

	// Generate 100 passwords and ensure they're all unique
	for i := 0; i < 100; i++ {
		password, err := GeneratePassword(32)
		if err != nil {
			t.Fatalf("GeneratePassword(32) failed at iteration %d: %v", i, err)
		}
		if generated[password] {
			t.Errorf("Duplicate password generated at iteration %d", i)
		}
		generated[password] = true
	}
}

func TestGenerateBase64Token_ValidChars(t *testing.T) {
	token := GenerateBase64Token(16)

	validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_="
	for i, c := range token {
		if !containsRune(validChars, c) {
			t.Errorf("Invalid character at position %d: %c", i, c)
		}
	}
}

func TestGenerateRandomPath_Prefix(t *testing.T) {
	for i := 0; i < 10; i++ {
		path, err := GenerateRandomPath("")
		if err != nil {
			t.Fatalf("GenerateRandomPath failed: %v", err)
		}
		if len(path) == 0 {
			t.Error("Expected non-empty path")
		}
		if path[0] != '/' {
			t.Errorf("Expected path to start with /, got %s", path)
		}
	}
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
