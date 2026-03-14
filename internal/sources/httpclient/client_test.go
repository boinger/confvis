package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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

	result, err := Get[testResponse](client, context.Background(), server.URL+"/test")
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

	_, err := Get[map[string]interface{}](client, context.Background(), server.URL+"/test")
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

	_, err := Get[map[string]interface{}](client, context.Background(), server.URL+"/test")
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

	_, err := Get[map[string]interface{}](client, context.Background(), server.URL+"/test")
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

	_, err := Get[map[string]interface{}](client, context.Background(), server.URL+"/test")
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

	_, err := Get[map[string]interface{}](client, context.Background(), server.URL+"/test")
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

	_, err := Get[map[string]interface{}](client, context.Background(), server.URL+"/test")
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

	_, err := Get[map[string]interface{}](client, context.Background(), server.URL+"/test")
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

	_, err := Get[map[string]interface{}](client, context.Background(), server.URL+"/test")
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

	_, err := Get[map[string]interface{}](client, context.Background(), server.URL+"/test")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "decoding response") {
		t.Errorf("error should mention decoding: %v", err)
	}
}

func TestClient_Get_NetworkError(t *testing.T) {
	client := New(Config{BaseURL: "http://localhost:1", MaxRetries: -1})

	_, err := Get[map[string]interface{}](client, context.Background(), "http://localhost:1/test")
	if err == nil {
		t.Fatal("expected error for network failure")
	}

	if !strings.Contains(err.Error(), "making request") {
		t.Errorf("error should mention making request: %v", err)
	}
}

func TestClient_Get_InvalidURL(t *testing.T) {
	client := New(Config{})

	_, err := Get[map[string]interface{}](client, context.Background(), "://invalid-url")
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

	_, err := Get[map[string]interface{}](client, ctx, server.URL+"/test")
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

// newTestClient creates a client with minimal backoff for fast tests.
func newTestClient(server *httptest.Server) *Client {
	return New(Config{
		BaseURL:        server.URL,
		MaxRetries:     3,
		InitialBackoff: 1 * time.Millisecond,
	})
}

func TestClient_Get_Retry_TransientThenSuccess(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = fmt.Fprint(w, "bad gateway")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(testResponse{Message: "ok", Value: 1})
	}))
	defer server.Close()

	client := newTestClient(server)
	result, err := Get[testResponse](client, context.Background(), server.URL+"/test")
	if err != nil {
		t.Fatalf("Get() error = %v, want success after retries", err)
	}

	if result.Message != "ok" {
		t.Errorf("result.Message = %q, want %q", result.Message, "ok")
	}
	if got := attempts.Load(); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestClient_Get_Retry_AllFail(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprint(w, "service unavailable")
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := Get[map[string]interface{}](client, context.Background(), server.URL+"/test")
	if err == nil {
		t.Fatal("expected error after all retries exhausted")
	}

	if !strings.Contains(err.Error(), "503") {
		t.Errorf("error should contain 503: %v", err)
	}

	// 1 initial + 3 retries = 4 total attempts
	if got := attempts.Load(); got != 4 {
		t.Errorf("attempts = %d, want 4", got)
	}
}

func TestClient_Get_Retry_429WithRetryAfter(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "1") // 1 second — but we use small backoff in test
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = fmt.Fprint(w, "rate limited")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(testResponse{Message: "ok", Value: 2})
	}))
	defer server.Close()

	// Use very small backoff; Retry-After=1s will still be respected
	// but we test the mechanism, not the actual sleep duration
	client := New(Config{
		BaseURL:        server.URL,
		MaxRetries:     2,
		InitialBackoff: 1 * time.Millisecond,
	})

	start := time.Now()
	result, err := Get[testResponse](client, context.Background(), server.URL+"/test")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Get() error = %v, want success after retry", err)
	}

	if result.Message != "ok" {
		t.Errorf("result.Message = %q, want %q", result.Message, "ok")
	}
	if got := attempts.Load(); got != 2 {
		t.Errorf("attempts = %d, want 2", got)
	}

	// Verify Retry-After was respected (should have waited ~1s)
	if elapsed < 900*time.Millisecond {
		t.Errorf("elapsed = %v, want >= 900ms (Retry-After: 1)", elapsed)
	}
}

func TestClient_Get_Retry_NonRetryableStatus(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := Get[map[string]interface{}](client, context.Background(), server.URL+"/test")
	if err == nil {
		t.Fatal("expected error for 403")
	}

	// Should NOT retry a 403 — only 1 attempt
	if got := attempts.Load(); got != 1 {
		t.Errorf("attempts = %d, want 1 (no retry for 403)", got)
	}
}

func TestClient_Get_Retry_DisabledWithNegativeOne(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = fmt.Fprint(w, "bad gateway")
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:        server.URL,
		MaxRetries:     -1,
		InitialBackoff: 1 * time.Millisecond,
	})

	_, err := Get[map[string]interface{}](client, context.Background(), server.URL+"/test")
	if err == nil {
		t.Fatal("expected error for 502 with retries disabled")
	}

	// No retries — exactly 1 attempt
	if got := attempts.Load(); got != 1 {
		t.Errorf("attempts = %d, want 1 (retries disabled)", got)
	}
}

func TestClient_Get_Retry_ContextCancelStopsRetry(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = fmt.Fprint(w, "bad gateway")
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:        server.URL,
		MaxRetries:     5,
		InitialBackoff: 100 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := Get[map[string]interface{}](client, ctx, server.URL+"/test")
	if err == nil {
		t.Fatal("expected error when context expires during retry")
	}

	// Should have attempted at most 2 times (first attempt + maybe one retry
	// before context deadline)
	if got := attempts.Load(); got > 2 {
		t.Errorf("attempts = %d, want <= 2 (context should have stopped retries)", got)
	}
}

func TestClient_Get_Retry_DefaultConfig(t *testing.T) {
	client := New(Config{})
	if client.maxRetries != DefaultMaxRetries {
		t.Errorf("maxRetries = %d, want %d", client.maxRetries, DefaultMaxRetries)
	}
	if client.initialBackoff != DefaultInitialBackoff {
		t.Errorf("initialBackoff = %v, want %v", client.initialBackoff, DefaultInitialBackoff)
	}
}

func TestParseRetryAfter_Seconds(t *testing.T) {
	d := parseRetryAfter("5")
	if d != 5*time.Second {
		t.Errorf("parseRetryAfter(\"5\") = %v, want 5s", d)
	}
}

func TestParseRetryAfter_Zero(t *testing.T) {
	d := parseRetryAfter("0")
	if d != 0 {
		t.Errorf("parseRetryAfter(\"0\") = %v, want 0", d)
	}
}

func TestParseRetryAfter_Invalid(t *testing.T) {
	d := parseRetryAfter("not-a-number")
	if d != 0 {
		t.Errorf("parseRetryAfter(\"not-a-number\") = %v, want 0", d)
	}
}

func TestClient_Get_ResponseHook(t *testing.T) {
	var hookCalled bool
	var capturedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Custom-Info", "test-value")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		MaxRetries: -1,
		OnResponse: func(headers http.Header) {
			hookCalled = true
			capturedHeaders = headers
		},
	})

	if _, err := Get[map[string]interface{}](client, context.Background(), server.URL+"/test"); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if !hookCalled {
		t.Error("OnResponse hook was not called")
	}
	if capturedHeaders.Get("X-Custom-Info") != "test-value" {
		t.Errorf("hook received X-Custom-Info = %q, want %q",
			capturedHeaders.Get("X-Custom-Info"), "test-value")
	}
}

func TestClient_Get_ResponseHook_NotCalledOnError(t *testing.T) {
	var hookCalled bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client := New(Config{
		BaseURL:    server.URL,
		MaxRetries: -1,
		OnResponse: func(_ http.Header) {
			hookCalled = true
		},
	})

	_, _ = Get[map[string]interface{}](client, context.Background(), server.URL+"/test")

	if hookCalled {
		t.Error("OnResponse hook should not be called on error responses")
	}
}

// failingReader is an io.ReadCloser whose Read always returns an error.
type failingReader struct {
	err error
}

func (f *failingReader) Read([]byte) (int, error) { return 0, f.err }
func (f *failingReader) Close() error             { return nil }

// failingBodyTransport returns a fixed HTTP response with a failing body reader.
type failingBodyTransport struct {
	statusCode int
	bodyErr    error
}

func (t *failingBodyTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: t.statusCode,
		Header:     http.Header{},
		Body:       &failingReader{err: t.bodyErr},
	}, nil
}

func TestDoGet_HTTPError_BodyReadFails(t *testing.T) {
	bodyErr := fmt.Errorf("connection reset during read")
	client := New(Config{
		BaseURL:    "http://fake.test",
		MaxRetries: -1,
	})
	client.httpClient = &http.Client{
		Transport: &failingBodyTransport{
			statusCode: http.StatusInternalServerError,
			bodyErr:    bodyErr,
		},
	}

	var result map[string]interface{}
	err := client.doGet(context.Background(), "http://fake.test/api", &result)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "500") {
		t.Errorf("error should contain status code 500: %v", err)
	}
	if !strings.Contains(errMsg, "body read error") {
		t.Errorf("error should mention body read error: %v", err)
	}
	if !strings.Contains(errMsg, "connection reset during read") {
		t.Errorf("error should contain the underlying read error: %v", err)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"plain error", fmt.Errorf("plain"), false},
		{"retryable error", &retryableError{err: fmt.Errorf("transient")}, true},
		{"wrapped retryable", fmt.Errorf("wrap: %w", &retryableError{err: fmt.Errorf("inner")}), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryable(tt.err); got != tt.want {
				t.Errorf("isRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}
