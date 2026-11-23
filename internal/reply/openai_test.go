package reply

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIGenerateParsesStructuredReply(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Errorf("missing authorization header")
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request["model"] != "test-model" {
			t.Errorf("unexpected model: %v", request["model"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"message\":\"今天已经很努力了。\",\"action\":\"倒一杯温水，慢慢喝完。\"}"}}]}`))
	}))
	defer server.Close()

	generator := NewOpenAI(server.URL, "test-secret", "test-model", server.Client())
	response, err := generator.Generate(context.Background(), 1, "今天很累")
	if err != nil {
		t.Fatal(err)
	}
	if response.Source != "ai" || response.Message != "今天已经很努力了。" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestOpenAIGenerateRejectsMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"not-json"}}]}`))
	}))
	defer server.Close()
	generator := NewOpenAI(server.URL, "secret", "model", server.Client())
	if _, err := generator.Generate(context.Background(), 1, "今天很累"); err == nil {
		t.Fatal("malformed model response should fail")
	}
}
