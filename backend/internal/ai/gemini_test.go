package ai

import (
	"context"
	"strings"
	"testing"
)

func TestNewFromEnvDisabledWithoutKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	if client := NewFromEnv(); client != nil {
		t.Fatalf("client = %#v, want nil", client)
	}
}

func TestNewFromEnvRespectsExplicitDisable(t *testing.T) {
	t.Setenv("AI_ENABLED", "false")
	t.Setenv("GEMINI_API_KEY", "test-key")
	if client := NewFromEnv(); client != nil {
		t.Fatalf("client = %#v, want nil when AI_ENABLED=false", client)
	}
}

func TestReviewRejectsMissingKey(t *testing.T) {
	client := &Client{}
	if _, err := client.Review(context.Background(), []Candidate{{SaleID: "s1", ReceiptID: "r1"}}); err == nil {
		t.Fatal("want error for missing api key")
	}
}

func TestFilterVerdictsDropsUnknownIDs(t *testing.T) {
	candidates := []Candidate{{SaleID: "s1", ReceiptID: "r1"}}
	verdicts := []Verdict{
		{SaleID: "s1", ReceiptID: "r1", Match: true, Reason: " nominal sama "},
		{SaleID: "evil", ReceiptID: "r9", Match: true, Reason: "karangan"},
	}
	out := filterVerdicts(candidates, verdicts)
	if len(out) != 1 {
		t.Fatalf("verdicts = %d, want 1", len(out))
	}
	if out[0].Reason != "nominal sama" {
		t.Fatalf("reason = %q", out[0].Reason)
	}
}

func TestFilterVerdictsTrimsLongReason(t *testing.T) {
	candidates := []Candidate{{SaleID: "s1", ReceiptID: "r1"}}
	long := strings.Repeat("x", 300)
	out := filterVerdicts(candidates, []Verdict{{SaleID: "s1", ReceiptID: "r1", Match: true, Reason: long}})
	if len(out) != 1 || len(out[0].Reason) != 200 {
		t.Fatalf("reason length = %d, want 200", len(out[0].Reason))
	}
}
