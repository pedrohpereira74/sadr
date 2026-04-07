package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestBulkFetchSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/rest/api/3/issue/bulkfetch" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Error("missing auth header")
		}

		resp := map[string]any{
			"issues": []map[string]any{
				{
					"key": "PROJ-101",
					"fields": map[string]any{
						"summary":  "Implement auth",
						"status":   map[string]string{"name": "Done"},
						"assignee": map[string]string{"displayName": "John"},
					},
				},
				{
					"key": "PROJ-102",
					"fields": map[string]any{
						"summary": "Fix bug",
						"status":  map[string]string{"name": "In Progress"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:  server.URL,
		Email:    "test@example.com",
		APIToken: "token",
		HTTP:     server.Client(),
	}

	issues, err := client.BulkFetch(context.Background(), []string{"PROJ-101", "PROJ-102"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues["PROJ-101"].Summary != "Implement auth" {
		t.Errorf("unexpected summary: %s", issues["PROJ-101"].Summary)
	}
	if issues["PROJ-101"].Assignee != "John" {
		t.Errorf("unexpected assignee: %s", issues["PROJ-101"].Assignee)
	}
	if issues["PROJ-102"].Status != "In Progress" {
		t.Errorf("unexpected status: %s", issues["PROJ-102"].Status)
	}
}

func TestBulkFetchPartial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"issues": []map[string]any{
				{
					"key": "PROJ-101",
					"fields": map[string]any{
						"summary": "Found one",
						"status":  map[string]string{"name": "Done"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:  server.URL,
		Email:    "test@example.com",
		APIToken: "token",
		HTTP:     server.Client(),
	}

	issues, err := client.BulkFetch(context.Background(), []string{"PROJ-101", "PROJ-999"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 issue (partial), got %d", len(issues))
	}
}

func TestBulkFetchAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer server.Close()

	client := &Client{
		BaseURL:  server.URL,
		Email:    "test@example.com",
		APIToken: "bad-token",
		HTTP:     server.Client(),
	}

	_, err := client.BulkFetch(context.Background(), []string{"PROJ-101"})
	if err == nil {
		t.Fatal("expected error on 401")
	}
}

func TestFetchAllBatches(t *testing.T) {
	var mu sync.Mutex
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()
		var req bulkRequest
		json.NewDecoder(r.Body).Decode(&req)

		var issues []map[string]any
		for _, key := range req.IssueIdsOrKeys {
			issues = append(issues, map[string]any{
				"key": key,
				"fields": map[string]any{
					"summary": "Issue " + key,
					"status":  map[string]string{"name": "Open"},
				},
			})
		}
		resp := map[string]any{"issues": issues}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:  server.URL,
		Email:    "test@example.com",
		APIToken: "token",
		HTTP:     server.Client(),
	}

	var keys []string
	for i := range 120 {
		keys = append(keys, fmt.Sprintf("PROJ-%d", i))
	}

	issues, err := client.FetchAll(context.Background(), keys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 120 {
		t.Errorf("expected 120 issues, got %d", len(issues))
	}
	if requestCount != 3 {
		t.Errorf("expected 3 requests, got %d", requestCount)
	}
}

func TestFetchAllEmpty(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
	}))
	defer server.Close()

	client := &Client{
		BaseURL:  server.URL,
		Email:    "test@example.com",
		APIToken: "token",
		HTTP:     server.Client(),
	}

	issues, err := client.FetchAll(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
	if requestCount != 0 {
		t.Errorf("expected 0 requests, got %d", requestCount)
	}
}

func TestFetchAllDeduplicates(t *testing.T) {
	var receivedKeys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req bulkRequest
		json.NewDecoder(r.Body).Decode(&req)
		receivedKeys = req.IssueIdsOrKeys

		resp := map[string]any{
			"issues": []map[string]any{
				{
					"key": "PROJ-1",
					"fields": map[string]any{
						"summary": "Test",
						"status":  map[string]string{"name": "Open"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:  server.URL,
		Email:    "test@example.com",
		APIToken: "token",
		HTTP:     server.Client(),
	}

	_, err := client.FetchAll(context.Background(), []string{"PROJ-1", "PROJ-1", "PROJ-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(receivedKeys) != 1 {
		t.Errorf("expected 1 deduplicated key, got %d", len(receivedKeys))
	}
}
