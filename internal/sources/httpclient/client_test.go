package httpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type testResponse struct {
	Message string `json:"message"`
	Value   int    `json:"value"`
}

func TestNew(t *testing.T) {
	cfg := Config{
		BaseURL:  "https://api.example.com",
		Token:    "test-token",
		AuthType: AuthBearer,
		Accept:   "application/vnd.api+json",
		Timeout:  10 * time.Second,
	}

	client := New(cfg)

	if client.baseURL != cfg.BaseURL {
		t.Errorf("baseURL = %q, want %q", client.baseURL, cfg.BaseURL)
	}
	if client.token != cfg.Token {
		t.Errorf("token = %q, want %q", client.token, cfg.Token)
	}
	if client.authType != cfg.AuthType {
		t.Errorf("authType = %v, want %v", client.authType, cfg.AuthType)
	}
	if client.accept != cfg.Accept {
		t.Errorf("accept = %q, want %q", client.accept, cfg.Accept)
	}
}

func TestNew_DefaultAccept(t *testing.T) {
	client := New(Config{})

	if client.accept != "application/json" {
		t.Errorf("accept = %q, want %q", client.accept, "application/json")
	}
}

func TestClient_Get_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want %q", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/test" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/test")
		}

		w.Header().Set("Content-Type", "application/json")
		resp := testResponse{Message: "hello", Value: 42}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encoding response: %v", err)
		}
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL})

	var result testResponse
	err := client.Get(context.Background(), server.URL+"/test", &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if result.Message != "hello" {
		t.Errorf("result.Message = %q, want %q", result.Message, "hello")
	}
	if result.Value != 42 {
		t.Errorf("result.Value = %d, want %d", result.Value, 42)
	}
}

func TestClient_Get_AuthBearer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", auth, "Bearer test-token")
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{}")); err != nil {
			t.Fatalf("writing response: %v", err)
		}
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:  server.URL,
		Token:    "test-token",
		AuthType: AuthBearer,
	})

	var result map[string]interface{}
	err := client.Get(context.Background(), server.URL+"/test", &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestClient_Get_AuthToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "token test-token" {
			t.Errorf("Authorization = %q, want %q", auth, "token test-token")
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{}")); err != nil {
			t.Fatalf("writing response: %v", err)
		}
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:  server.URL,
		Token:    "test-token",
		AuthType: AuthToken,
	})

	var result map[string]interface{}
	err := client.Get(context.Background(), server.URL+"/test", &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestClient_Get_AuthBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected Basic auth to be present")
		}
		if user != "test-token" {
			t.Errorf("username = %q, want %q", user, "test-token")
		}
		if pass != "" {
			t.Errorf("password = %q, want empty string", pass)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{}")); err != nil {
			t.Fatalf("writing response: %v", err)
		}
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:  server.URL,
		Token:    "test-token",
		AuthType: AuthBasic,
	})

	var result map[string]interface{}
	err := client.Get(context.Background(), server.URL+"/test", &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestClient_Get_AuthNone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "" {
			t.Errorf("Authorization = %q, want empty", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{}")); err != nil {
			t.Fatalf("writing response: %v", err)
		}
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:  server.URL,
		Token:    "test-token",
		AuthType: AuthNone,
	})

	var result map[string]interface{}
	err := client.Get(context.Background(), server.URL+"/test", &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestClient_Get_NoToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "" {
			t.Errorf("Authorization = %q, want empty", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{}")); err != nil {
			t.Fatalf("writing response: %v", err)
		}
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:  server.URL,
		AuthType: AuthBearer, // Even with AuthBearer, no header if no token
	})

	var result map[string]interface{}
	err := client.Get(context.Background(), server.URL+"/test", &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestClient_Get_AcceptHeader(t *testing.T) {
	customAccept := "application/vnd.github+json"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		if accept != customAccept {
			t.Errorf("Accept = %q, want %q", accept, customAccept)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{}")); err != nil {
			t.Fatalf("writing response: %v", err)
		}
	}))
	defer server.Close()

	client := New(Config{
		BaseURL: server.URL,
		Accept:  customAccept,
	})

	var result map[string]interface{}
	err := client.Get(context.Background(), server.URL+"/test", &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestClient_Get_ExtraHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiVersion := r.Header.Get("X-API-Version")
		if apiVersion != "2022-11-28" {
			t.Errorf("X-API-Version = %q, want %q", apiVersion, "2022-11-28")
		}
		customHeader := r.Header.Get("X-Custom-Header")
		if customHeader != "custom-value" {
			t.Errorf("X-Custom-Header = %q, want %q", customHeader, "custom-value")
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{}")); err != nil {
			t.Fatalf("writing response: %v", err)
		}
	}))
	defer server.Close()

	client := New(Config{
		BaseURL: server.URL,
		ExtraHeaders: map[string]string{
			"X-API-Version":   "2022-11-28",
			"X-Custom-Header": "custom-value",
		},
	})

	var result map[string]interface{}
	err := client.Get(context.Background(), server.URL+"/test", &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestClient_Get_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL})

	var result map[string]interface{}
	err := client.Get(context.Background(), server.URL+"/test", &result)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should contain status code: %v", err)
	}
	if !strings.Contains(err.Error(), "Not Found") {
		t.Errorf("error should contain body: %v", err)
	}
}

func TestClient_Get_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("not valid json")); err != nil {
			t.Fatalf("writing response: %v", err)
		}
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL})

	var result map[string]interface{}
	err := client.Get(context.Background(), server.URL+"/test", &result)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "decoding response") {
		t.Errorf("error should mention decoding: %v", err)
	}
}

func TestClient_Get_NetworkError(t *testing.T) {
	client := New(Config{BaseURL: "http://localhost:1"})

	var result map[string]interface{}
	err := client.Get(context.Background(), "http://localhost:1/test", &result)
	if err == nil {
		t.Fatal("expected error for network failure")
	}

	if !strings.Contains(err.Error(), "making request") {
		t.Errorf("error should mention making request: %v", err)
	}
}

func TestClient_Get_InvalidURL(t *testing.T) {
	client := New(Config{})

	var result map[string]interface{}
	err := client.Get(context.Background(), "://invalid-url", &result)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}

	if !strings.Contains(err.Error(), "creating request") {
		t.Errorf("error should mention creating request: %v", err)
	}
}

func TestClient_Get_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{}")); err != nil {
			// Context was canceled, write may fail
			return
		}
	}))
	defer server.Close()

	client := New(Config{BaseURL: server.URL})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	var result map[string]interface{}
	err := client.Get(ctx, server.URL+"/test", &result)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestClient_BaseURL(t *testing.T) {
	client := New(Config{BaseURL: "https://api.example.com"})

	if got := client.BaseURL(); got != "https://api.example.com" {
		t.Errorf("BaseURL() = %q, want %q", got, "https://api.example.com")
	}
}
