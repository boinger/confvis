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

	// Create fresh registry with multiple sources (inserted out of order)
	Registry = make(map[string]Source)
	Registry["gamma"] = &mockSource{name: "gamma"}
	Registry["alpha"] = &mockSource{name: "alpha"}
	Registry["beta"] = &mockSource{name: "beta"}

	got := Names()
	if len(got) != 3 {
		t.Errorf("Names() returned %d names, want 3", len(got))
	}

	// Verify names are sorted alphabetically
	expected := []string{"alpha", "beta", "gamma"}
	for i, name := range expected {
		if got[i] != name {
			t.Errorf("Names()[%d] = %q, want %q (should be sorted)", i, got[i], name)
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

func TestResolveCommand_FromExtra(t *testing.T) {
	opts := Options{
		Extra: map[string]string{"grype-cmd": "/usr/local/bin/grype"},
	}
	got := ResolveCommand(opts, "grype-cmd", "GRYPE_CMD")
	if got != "/usr/local/bin/grype" {
		t.Errorf("ResolveCommand() = %q, want %q", got, "/usr/local/bin/grype")
	}
}

func TestResolveCommand_FromEnvVar(t *testing.T) {
	t.Setenv("TEST_CMD", "/opt/bin/tool")
	opts := Options{}
	got := ResolveCommand(opts, "tool-cmd", "TEST_CMD")
	if got != "/opt/bin/tool" {
		t.Errorf("ResolveCommand() = %q, want %q", got, "/opt/bin/tool")
	}
}

func TestResolveCommand_ExtraTakesPrecedence(t *testing.T) {
	t.Setenv("TEST_CMD", "/opt/bin/tool")
	opts := Options{
		Extra: map[string]string{"tool-cmd": "/custom/tool"},
	}
	got := ResolveCommand(opts, "tool-cmd", "TEST_CMD")
	if got != "/custom/tool" {
		t.Errorf("ResolveCommand() = %q, want %q (Extra should take precedence)", got, "/custom/tool")
	}
}

func TestResolveCommand_NilExtra(t *testing.T) {
	t.Setenv("TEST_CMD", "/opt/bin/tool")
	opts := Options{Extra: nil}
	got := ResolveCommand(opts, "tool-cmd", "TEST_CMD")
	if got != "/opt/bin/tool" {
		t.Errorf("ResolveCommand() = %q, want %q", got, "/opt/bin/tool")
	}
}

func TestResolveCommand_EmptyExtraFallsThrough(t *testing.T) {
	t.Setenv("TEST_CMD", "/opt/bin/tool")
	opts := Options{
		Extra: map[string]string{"tool-cmd": ""},
	}
	got := ResolveCommand(opts, "tool-cmd", "TEST_CMD")
	if got != "/opt/bin/tool" {
		t.Errorf("ResolveCommand() = %q, want %q (empty Extra should fall through)", got, "/opt/bin/tool")
	}
}

func TestResolveCommand_NothingSet(t *testing.T) {
	t.Setenv("UNSET_CMD", "")
	opts := Options{}
	got := ResolveCommand(opts, "tool-cmd", "UNSET_CMD")
	if got != "" {
		t.Errorf("ResolveCommand() = %q, want empty", got)
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

func TestDeriveTitleFromPath_ExplicitTitle(t *testing.T) {
	got := DeriveTitleFromPath("/some/path", "My Title")
	if got != "My Title" {
		t.Errorf("DeriveTitleFromPath() = %q, want %q", got, "My Title")
	}
}

func TestDeriveTitleFromPath_DerivedFromPath(t *testing.T) {
	got := DeriveTitleFromPath("/path/to/myproject", "")
	if got != "myproject" {
		t.Errorf("DeriveTitleFromPath() = %q, want %q", got, "myproject")
	}
}

func TestDeriveTitleFromPath_RelativePath(t *testing.T) {
	got := DeriveTitleFromPath("./...", "")
	// filepath.Abs resolves relative paths; Base returns the last element
	if got == "" {
		t.Error("DeriveTitleFromPath() returned empty string for relative path")
	}
	// "./..." resolves to cwd/... so Base should be "..."
	if got != "..." {
		t.Errorf("DeriveTitleFromPath() = %q, want %q", got, "...")
	}
}

func TestResolveTitle_FirstNonEmpty(t *testing.T) {
	got := ResolveTitle("", "fallback", "last")
	if got != "fallback" {
		t.Errorf("ResolveTitle() = %q, want %q", got, "fallback")
	}
}

func TestResolveTitle_AllEmpty(t *testing.T) {
	got := ResolveTitle("", "")
	if got != "" {
		t.Errorf("ResolveTitle() = %q, want %q", got, "")
	}
}

func TestResolveTitle_FirstNonEmptyUsed(t *testing.T) {
	got := ResolveTitle("first", "second")
	if got != "first" {
		t.Errorf("ResolveTitle() = %q, want %q", got, "first")
	}
}

func TestGetExtra_Found(t *testing.T) {
	opts := Options{
		Extra: map[string]string{"key": "value"},
	}
	got := GetExtra(opts, "key", "default")
	if got != "value" {
		t.Errorf("GetExtra() = %q, want %q", got, "value")
	}
}

func TestGetExtra_Missing(t *testing.T) {
	opts := Options{}
	got := GetExtra(opts, "key", "default")
	if got != "default" {
		t.Errorf("GetExtra() = %q, want %q", got, "default")
	}
}

func TestGetExtra_EmptyValue(t *testing.T) {
	opts := Options{
		Extra: map[string]string{"key": ""},
	}
	got := GetExtra(opts, "key", "default")
	if got != "default" {
		t.Errorf("GetExtra() = %q, want %q", got, "default")
	}
}
