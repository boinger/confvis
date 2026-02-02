package sources

import (
	"context"
	"testing"

	"github.com/boinger/confvis/internal/confidence"
)

// mockSource implements Source interface for testing.
type mockSource struct {
	name string
}

func (m *mockSource) Name() string {
	return m.name
}

func (m *mockSource) Fetch(_ context.Context, _ Options) (*confidence.Report, error) {
	return &confidence.Report{Title: m.name}, nil
}

func TestRegister_Success(t *testing.T) {
	// Save original registry
	origRegistry := Registry
	defer func() { Registry = origRegistry }()

	// Create fresh registry
	Registry = make(map[string]Source)

	mock := &mockSource{name: "test-source"}
	Register(mock)

	if got := Registry["test-source"]; got != mock {
		t.Errorf("Register() did not add source to registry")
	}
}

func TestRegister_Panic(t *testing.T) {
	// Save original registry
	origRegistry := Registry
	defer func() { Registry = origRegistry }()

	// Create fresh registry
	Registry = make(map[string]Source)

	mock1 := &mockSource{name: "duplicate"}
	mock2 := &mockSource{name: "duplicate"}

	Register(mock1)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Register() should panic on duplicate name")
		}
	}()

	Register(mock2)
}

func TestGet_Found(t *testing.T) {
	// Save original registry
	origRegistry := Registry
	defer func() { Registry = origRegistry }()

	// Create fresh registry with a source
	Registry = make(map[string]Source)
	mock := &mockSource{name: "get-test"}
	Registry["get-test"] = mock

	got := Get("get-test")
	if got != mock {
		t.Errorf("Get() = %v, want %v", got, mock)
	}
}

func TestGet_NotFound(t *testing.T) {
	// Save original registry
	origRegistry := Registry
	defer func() { Registry = origRegistry }()

	// Create fresh empty registry
	Registry = make(map[string]Source)

	got := Get("nonexistent")
	if got != nil {
		t.Errorf("Get() = %v, want nil", got)
	}
}

func TestNames_Empty(t *testing.T) {
	// Save original registry
	origRegistry := Registry
	defer func() { Registry = origRegistry }()

	// Create fresh empty registry
	Registry = make(map[string]Source)

	got := Names()
	if len(got) != 0 {
		t.Errorf("Names() = %v, want empty slice", got)
	}
}

func TestNames_Multiple(t *testing.T) {
	// Save original registry
	origRegistry := Registry
	defer func() { Registry = origRegistry }()

	// Create fresh registry with multiple sources
	Registry = make(map[string]Source)
	Registry["alpha"] = &mockSource{name: "alpha"}
	Registry["beta"] = &mockSource{name: "beta"}
	Registry["gamma"] = &mockSource{name: "gamma"}

	got := Names()
	if len(got) != 3 {
		t.Errorf("Names() returned %d names, want 3", len(got))
	}

	// Check all names are present (order may vary)
	found := make(map[string]bool)
	for _, name := range got {
		found[name] = true
	}

	for _, expected := range []string{"alpha", "beta", "gamma"} {
		if !found[expected] {
			t.Errorf("Names() missing %q", expected)
		}
	}
}

func TestOptions_DefaultValues(t *testing.T) {
	opts := Options{}

	if opts.URL != "" {
		t.Errorf("URL should default to empty")
	}
	if opts.Project != "" {
		t.Errorf("Project should default to empty")
	}
	if opts.Token != "" {
		t.Errorf("Token should default to empty")
	}
	if opts.Timeout != 0 {
		t.Errorf("Timeout should default to 0")
	}
	if opts.Extra != nil {
		t.Errorf("Extra should default to nil")
	}
}

func TestOptions_Extra(t *testing.T) {
	opts := Options{
		Extra: map[string]string{
			"workflow": "ci.yml",
			"event":    "push",
		},
	}

	if opts.Extra["workflow"] != "ci.yml" {
		t.Errorf("Extra[workflow] = %q, want %q", opts.Extra["workflow"], "ci.yml")
	}
	if opts.Extra["event"] != "push" {
		t.Errorf("Extra[event] = %q, want %q", opts.Extra["event"], "push")
	}
}
