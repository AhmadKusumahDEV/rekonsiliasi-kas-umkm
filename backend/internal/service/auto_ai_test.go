package service

import (
	"strings"
	"testing"
)

func TestAIMinScoreBounds(t *testing.T) {
	cases := []struct {
		value string
		want  float64
	}{
		{"", 0.90},
		{"abc", 0.90},
		{"NaN", 0.90},
		{"0.59", 0.90},
		{"1.01", 0.90},
		{"0.85", 0.85},
		{"0.95", 0.95},
	}
	for _, tc := range cases {
		t.Setenv("AI_MIN_SCORE", tc.value)
		if got := aiMinScore(); got != tc.want {
			t.Fatalf("aiMinScore(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

func TestSanitizeAIReason(t *testing.T) {
	got := sanitizeAIReason("  nominal\x00sama\n referensi  ")
	if got != "nominal sama referensi" {
		t.Fatalf("reason = %q", got)
	}
	long := strings.Repeat("á", 200)
	if runes := []rune(sanitizeAIReason(long)); len(runes) != 120 {
		t.Fatalf("rune length = %d, want 120", len(runes))
	}
}
