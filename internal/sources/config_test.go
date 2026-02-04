package sources

import (
	"strings"
	"testing"
	"time"
)

func TestConfigResolver_Resolve_AllFromOptions(t *testing.T) {
	resolver := &ConfigResolver{
		SourceName:     "testsrc",
		TokenEnvVar:    "TESTSRC_TOKEN",
		URLEnvVar:      "TESTSRC_URL",
		TokenRequired:  true,
		URLRequired:    true,
		DefaultTimeout: 30 * time.Second,
	}

	opts := Options{
		Token:   "my-token",
		URL:     "https://api.example.com",
		Timeout: 60,
	}

	cfg, err := resolver.Resolve(opts)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if cfg.Token != "my-token" {
		t.Errorf("Token = %q, want %q", cfg.Token, "my-token")
	}
	if cfg.URL != "https://api.example.com" {
		t.Errorf("URL = %q, want %q", cfg.URL, "https://api.example.com")
	}
	if cfg.Timeout != 60*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 60*time.Second)
	}
}

func TestConfigResolver_Resolve_TokenFromEnv(t *testing.T) {
	const envKey = "CONFVIS_TEST_TOKEN_123"
	t.Setenv(envKey, "env-token")

	resolver := &ConfigResolver{
		SourceName:    "testsrc",
		TokenEnvVar:   envKey,
		TokenRequired: true,
	}

	opts := Options{} // Token not provided

	cfg, err := resolver.Resolve(opts)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if cfg.Token != "env-token" {
		t.Errorf("Token = %q, want %q", cfg.Token, "env-token")
	}
}

func TestConfigResolver_Resolve_URLFromEnv(t *testing.T) {
	const envKey = "CONFVIS_TEST_URL_123"
	t.Setenv(envKey, "https://env.example.com")

	resolver := &ConfigResolver{
		SourceName:  "testsrc",
		URLEnvVar:   envKey,
		URLRequired: true,
	}

	opts := Options{} // URL not provided

	cfg, err := resolver.Resolve(opts)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if cfg.URL != "https://env.example.com" {
		t.Errorf("URL = %q, want %q", cfg.URL, "https://env.example.com")
	}
}

func TestConfigResolver_Resolve_OptionOverridesEnv(t *testing.T) {
	const tokenEnvKey = "CONFVIS_TEST_TOKEN_456"
	const urlEnvKey = "CONFVIS_TEST_URL_456"

	t.Setenv(tokenEnvKey, "env-token")
	t.Setenv(urlEnvKey, "https://env.example.com")

	resolver := &ConfigResolver{
		SourceName:    "testsrc",
		TokenEnvVar:   tokenEnvKey,
		URLEnvVar:     urlEnvKey,
		TokenRequired: true,
		URLRequired:   true,
	}

	opts := Options{
		Token: "opt-token",
		URL:   "https://opt.example.com",
	}

	cfg, err := resolver.Resolve(opts)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if cfg.Token != "opt-token" {
		t.Errorf("Token = %q, want %q", cfg.Token, "opt-token")
	}
	if cfg.URL != "https://opt.example.com" {
		t.Errorf("URL = %q, want %q", cfg.URL, "https://opt.example.com")
	}
}

func TestConfigResolver_Resolve_TokenRequired_Missing(t *testing.T) {
	// Use a unique env key that won't be set in the environment
	const envKey = "CONFVIS_TEST_TOKEN_MISSING_XYZ"

	resolver := &ConfigResolver{
		SourceName:    "testsrc",
		TokenEnvVar:   envKey,
		TokenRequired: true,
	}

	opts := Options{}

	_, err := resolver.Resolve(opts)
	if err == nil {
		t.Fatal("expected error for missing required token")
	}

	if !strings.Contains(err.Error(), "testsrc token required") {
		t.Errorf("error should mention source name: %v", err)
	}
	if !strings.Contains(err.Error(), envKey) {
		t.Errorf("error should mention env var: %v", err)
	}
}

func TestConfigResolver_Resolve_URLRequired_Missing(t *testing.T) {
	// Use a unique env key that won't be set in the environment
	const envKey = "CONFVIS_TEST_URL_MISSING_XYZ"

	resolver := &ConfigResolver{
		SourceName:  "testsrc",
		URLEnvVar:   envKey,
		URLRequired: true,
	}

	opts := Options{}

	_, err := resolver.Resolve(opts)
	if err == nil {
		t.Fatal("expected error for missing required URL")
	}

	if !strings.Contains(err.Error(), "testsrc URL required") {
		t.Errorf("error should mention source name: %v", err)
	}
	if !strings.Contains(err.Error(), envKey) {
		t.Errorf("error should mention env var: %v", err)
	}
}

func TestConfigResolver_Resolve_TokenOptional(t *testing.T) {
	resolver := &ConfigResolver{
		SourceName:    "testsrc",
		TokenEnvVar:   "NONEXISTENT_TOKEN_VAR",
		TokenRequired: false, // optional
	}

	opts := Options{}

	cfg, err := resolver.Resolve(opts)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if cfg.Token != "" {
		t.Errorf("Token = %q, want empty", cfg.Token)
	}
}

func TestConfigResolver_Resolve_URLOptional(t *testing.T) {
	resolver := &ConfigResolver{
		SourceName:  "testsrc",
		URLEnvVar:   "NONEXISTENT_URL_VAR",
		URLRequired: false, // optional
	}

	opts := Options{}

	cfg, err := resolver.Resolve(opts)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if cfg.URL != "" {
		t.Errorf("URL = %q, want empty", cfg.URL)
	}
}

func TestConfigResolver_Resolve_DefaultTimeout(t *testing.T) {
	resolver := &ConfigResolver{
		SourceName:     "testsrc",
		DefaultTimeout: 45 * time.Second,
	}

	opts := Options{Timeout: 0} // Zero means use default

	cfg, err := resolver.Resolve(opts)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if cfg.Timeout != 45*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 45*time.Second)
	}
}

func TestConfigResolver_Resolve_FallbackTimeout(t *testing.T) {
	resolver := &ConfigResolver{
		SourceName:     "testsrc",
		DefaultTimeout: 0, // No default
	}

	opts := Options{Timeout: 0}

	cfg, err := resolver.Resolve(opts)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 30*time.Second)
	}
}

func TestConfigResolver_Resolve_NegativeTimeout(t *testing.T) {
	resolver := &ConfigResolver{
		SourceName:     "testsrc",
		DefaultTimeout: 20 * time.Second,
	}

	opts := Options{Timeout: -5}

	cfg, err := resolver.Resolve(opts)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if cfg.Timeout != 20*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 20*time.Second)
	}
}

func TestConfigResolver_Resolve_NoEnvVarConfigured(t *testing.T) {
	resolver := &ConfigResolver{
		SourceName: "testsrc",
		// No env vars configured
		TokenRequired: false,
		URLRequired:   false,
	}

	opts := Options{}

	cfg, err := resolver.Resolve(opts)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if cfg.Token != "" {
		t.Errorf("Token = %q, want empty", cfg.Token)
	}
	if cfg.URL != "" {
		t.Errorf("URL = %q, want empty", cfg.URL)
	}
}
