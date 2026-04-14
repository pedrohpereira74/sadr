package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func newTestClient(server *httptest.Server) *Client {
	return &Client{
		BaseURL:  server.URL,
		Username: "test@example.com",
		Password: "token",
		HTTP:     server.Client(),
	}
}

func searchHandler(issues []map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/rest/api/2/search" {
			http.Error(w, "unexpected path: "+r.URL.Path, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"issues": issues})
	}
}

func TestFetchAllSuccess(t *testing.T) {
	server := httptest.NewServer(searchHandler([]map[string]any{
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
	}))
	defer server.Close()

	issues, err := newTestClient(server).FetchAll(context.Background(), []string{"PROJ-101", "PROJ-102"})
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

func TestFetchAllPartial(t *testing.T) {
	server := httptest.NewServer(searchHandler([]map[string]any{
		{
			"key": "PROJ-101",
			"fields": map[string]any{
				"summary": "Found one",
				"status":  map[string]string{"name": "Done"},
			},
		},
	}))
	defer server.Close()

	issues, err := newTestClient(server).FetchAll(context.Background(), []string{"PROJ-101", "PROJ-999"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 issue (partial match), got %d", len(issues))
	}
}

func TestFetchAllAuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer server.Close()

	_, err := newTestClient(server).FetchAll(context.Background(), []string{"PROJ-101"})
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

		var req searchRequest
		json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"issues": []map[string]any{}})
	}))
	defer server.Close()

	var keys []string
	for i := range 120 {
		keys = append(keys, fmt.Sprintf("PROJ-%d", i))
	}

	_, err := newTestClient(server).FetchAll(context.Background(), keys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if requestCount != 3 {
		t.Errorf("expected 3 batch requests, got %d", requestCount)
	}
}

func TestFetchAllEmpty(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
	}))
	defer server.Close()

	issues, err := newTestClient(server).FetchAll(context.Background(), nil)
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
	var receivedBody searchRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"issues": []map[string]any{
				{
					"key":    "PROJ-1",
					"fields": map[string]any{"summary": "Test", "status": map[string]string{"name": "Open"}},
				},
			},
		})
	}))
	defer server.Close()

	_, err := newTestClient(server).FetchAll(context.Background(), []string{"PROJ-1", "PROJ-1", "PROJ-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Count(receivedBody.JQL, "PROJ-1") != 1 {
		t.Errorf("expected deduplicated JQL, got: %s", receivedBody.JQL)
	}
}

func TestApplyAuthBasic(t *testing.T) {
	authHeader := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"issues": []map[string]any{}})
	}))
	defer server.Close()

	newTestClient(server).FetchAll(context.Background(), []string{"PROJ-1"})
	if len(authHeader) == 0 {
		t.Error("expected Authorization header for Basic Auth")
	}
	if authHeader[:5] != "Basic" {
		t.Errorf("expected Basic auth, got: %s", authHeader[:5])
	}
}

func TestApplyAuthBearer(t *testing.T) {
	authHeader := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"issues": []map[string]any{}})
	}))
	defer server.Close()

	c := &Client{
		BaseURL: server.URL,
		Token:   "my-pat-token",
		HTTP:    server.Client(),
	}
	c.FetchAll(context.Background(), []string{"PROJ-1"})
	if authHeader != "Bearer my-pat-token" {
		t.Errorf("expected Bearer token, got: %s", authHeader)
	}
}

