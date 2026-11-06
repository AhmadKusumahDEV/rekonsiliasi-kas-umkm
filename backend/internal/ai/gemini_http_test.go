package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestReviewReadsAnswerAfterReasoningPart(t *testing.T) {
	var seenBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenBody, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[
 			{"text":"Saya menganalisis kandidat satu per satu."},
 			{"text":"{\"verdicts\":[{\"sale_id\":\"s1\",\"receipt_id\":\"r1\",\"match\":true,\"reason\":\"referensi sama\"}]}"}
 		]}}]}`))
	}))
	defer server.Close()

	client := &Client{APIKey: "k", Model: "gemini-2.5-flash", BaseURL: server.URL, HTTP: server.Client()}
	candidates := []Candidate{{SaleID: "s1", ReceiptID: "r1", Score: 0.93}}
	verdicts, err := client.Review(context.Background(), candidates)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if len(verdicts) != 1 || verdicts[0].SaleID != "s1" || !verdicts[0].Match {
		t.Fatalf("verdicts = %+v", verdicts)
	}

	var sent map[string]any
	if err := json.Unmarshal(seenBody, &sent); err != nil {
		t.Fatalf("request body bukan JSON: %v", err)
	}
	config, ok := sent["generationConfig"].(map[string]any)
	if !ok {
		t.Fatalf("generationConfig hilang")
	}
	thinking, ok := config["thinkingConfig"].(map[string]any)
	if !ok || thinking["thinkingBudget"] != float64(0) {
		t.Fatalf("thinkingConfig = %v, ingin budget 0 untuk model flash", config["thinkingConfig"])
	}
}

func TestReviewRejectsPlainTextAnswer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"menurut saya cocok"}]}}]}`))
	}))
	defer server.Close()

	client := &Client{APIKey: "k", Model: "gemini-2.5-flash", BaseURL: server.URL, HTTP: server.Client()}
	if _, err := client.Review(context.Background(), []Candidate{{SaleID: "s1", ReceiptID: "r1"}}); err == nil {
		t.Fatal("jawaban non-JSON seharusnya ditolak")
	}
}

func TestReviewSkipsThinkingConfigForOtherModels(t *testing.T) {
	var seenBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenBody, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"verdicts\":[]}"}]}}]}`))
	}))
	defer server.Close()

	client := &Client{APIKey: "k", Model: "gemini-2.0-flash", BaseURL: server.URL, HTTP: server.Client()}
	if _, err := client.Review(context.Background(), []Candidate{{SaleID: "s1", ReceiptID: "r1"}}); err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if strings.Contains(string(seenBody), "thinkingConfig") {
		t.Fatal("model non-2.5-flash tidak boleh menerima thinkingConfig")
	}
}

func TestReviewDefaultTimeoutAllowsSlowModel(t *testing.T) {
	if reviewTimeout < 20*time.Second {
		t.Fatalf("reviewTimeout = %v, terlalu pendek untuk gemini-2.5-flash", reviewTimeout)
	}
}
