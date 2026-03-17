package service

import (
	"strings"
	"testing"
)

func TestSanitizeOutputRemovesControlCharacters(t *testing.T) {
	output := "hello\x00\r\nworld\x07"

	sanitized := sanitizeOutput(output)

	if sanitized != "hello\nworld" {
		t.Fatalf("unexpected sanitized output %q", sanitized)
	}
}

func TestSanitizeOutputTruncatesLargeOutput(t *testing.T) {
	output := strings.Repeat("a", maxOutputLength+10)

	sanitized := sanitizeOutput(output)

	if !strings.Contains(sanitized, "...[truncated]") {
		t.Fatalf("expected output to be truncated, got %q", sanitized)
	}
}
