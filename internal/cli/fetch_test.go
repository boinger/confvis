package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// sonarqubeHandler creates a handler that returns SonarQube mock responses.
func sonarqubeHandler(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/measures/component":
			resp := map[string]interface{}{
				"component": map[string]interface{}{
					"key":  r.URL.Query().Get("component"),
					"name": "Test Project",
					"measures": []map[string]string{
						{"metric": "coverage", "value": "85.0"},
						{"metric": "reliability_rating", "value": "1.0"},
						{"metric": "security_rating", "value": "1.0"},
						{"metric": "sqale_rating", "value": "2.0"},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Errorf("encoding response: %v", err)
			}

		case "/api/qualitygates/project_status":
			resp := map[string]interface{}{
				"projectStatus": map[string]interface{}{
					"status": "OK",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Errorf("encoding response: %v", err)
			}

		default:
			http.NotFound(w, r)
		}
	})
}

func TestFetch_SonarQube_Success(t *testing.T) {
	server := httptest.NewServer(sonarqubeHandler(t))
	defer server.Close()

	bin := buildBinary(t)
	outputPath := filepath.Join(t.TempDir(), "report.json")

	cmd := exec.Command(bin, "fetch", "sonarqube",
		"--url", server.URL,
		"--project", "myproject",
		"-o", outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fetch sonarqube failed: %v\n%s", err, output)
	}

	// Verify output file
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	var report map[string]interface{}
	if err := json.Unmarshal(content, &report); err != nil {
		t.Fatalf("parsing JSON: %v", err)
	}

	if report["title"] != "Test Project" {
		t.Errorf("title = %v, want %q", report["title"], "Test Project")
	}
	if report["source"] != "sonarqube" {
		t.Errorf("source = %v, want %q", report["source"], "sonarqube")
	}
	if report["threshold"] != float64(75) {
		t.Errorf("threshold = %v, want %v", report["threshold"], 75)
	}

	factors, ok := report["factors"].([]interface{})
	if !ok {
		t.Fatalf("factors is not an array: %T", report["factors"])
	}
	if len(factors) != 4 {
		t.Errorf("len(factors) = %d, want %d", len(factors), 4)
	}
}

func TestFetch_SonarQube_Stdout(t *testing.T) {
	server := httptest.NewServer(sonarqubeHandler(t))
	defer server.Close()

	bin := buildBinary(t)

	cmd := exec.Command(bin, "fetch", "sonarqube",
		"--url", server.URL,
		"--project", "myproject",
		"-o", "-",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fetch sonarqube stdout failed: %v\n%s", err, output)
	}

	var report map[string]interface{}
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("parsing JSON output: %v", err)
	}

	if report["source"] != "sonarqube" {
		t.Errorf("source = %v, want %q", report["source"], "sonarqube")
	}
}

func TestFetch_SonarQube_CustomTitle(t *testing.T) {
	server := httptest.NewServer(sonarqubeHandler(t))
	defer server.Close()

	bin := buildBinary(t)

	cmd := exec.Command(bin, "fetch", "sonarqube",
		"--url", server.URL,
		"--project", "myproject",
		"--title", "Custom Title",
		"-o", "-",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fetch with custom title failed: %v\n%s", err, output)
	}

	var report map[string]interface{}
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("parsing JSON: %v", err)
	}

	if report["title"] != "Custom Title" {
		t.Errorf("title = %v, want %q", report["title"], "Custom Title")
	}
}

func TestFetch_SonarQube_CustomThreshold(t *testing.T) {
	server := httptest.NewServer(sonarqubeHandler(t))
	defer server.Close()

	bin := buildBinary(t)

	cmd := exec.Command(bin, "fetch", "sonarqube",
		"--url", server.URL,
		"--project", "myproject",
		"--threshold", "90",
		"-o", "-",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fetch with custom threshold failed: %v\n%s", err, output)
	}

	var report map[string]interface{}
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("parsing JSON: %v", err)
	}

	if report["threshold"] != float64(90) {
		t.Errorf("threshold = %v, want %v", report["threshold"], 90)
	}
}

func TestFetch_SonarQube_EnvVarURL(t *testing.T) {
	server := httptest.NewServer(sonarqubeHandler(t))
	defer server.Close()

	bin := buildBinary(t)

	cmd := exec.Command(bin, "fetch", "sonarqube",
		"--project", "myproject",
		"-o", "-",
	)
	cmd.Env = append(os.Environ(), "SONARQUBE_URL="+server.URL)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fetch with env var URL failed: %v\n%s", err, output)
	}

	var report map[string]interface{}
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("parsing JSON: %v", err)
	}

	if report["source"] != "sonarqube" {
		t.Errorf("source = %v, want %q", report["source"], "sonarqube")
	}
}

func TestFetch_SonarQube_PipeToGauge(t *testing.T) {
	server := httptest.NewServer(sonarqubeHandler(t))
	defer server.Close()

	bin := buildBinary(t)

	// Fetch to stdout
	fetchCmd := exec.Command(bin, "fetch", "sonarqube",
		"--url", server.URL,
		"--project", "myproject",
		"-o", "-",
	)

	fetchOutput, err := fetchCmd.Output()
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}

	// Pipe to gauge
	gaugeCmd := exec.Command(bin, "gauge", "-c", "-", "-o", "-")
	gaugeCmd.Stdin = strings.NewReader(string(fetchOutput))

	gaugeOutput, err := gaugeCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gauge failed: %v\n%s", err, gaugeOutput)
	}

	if !strings.Contains(string(gaugeOutput), "<svg") {
		t.Error("gauge output does not contain SVG")
	}
}

func TestFetch_UnknownSource(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "fetch", "unknown",
		"--project", "myproject",
		"-o", "-",
	)

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for unknown source")
	}

	if !strings.Contains(string(output), "unknown source") {
		t.Errorf("error message should mention unknown source: %s", output)
	}
}

func TestFetch_MissingProject(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "fetch", "sonarqube",
		"--url", "http://localhost:9000",
		"-o", "-",
	)

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for missing project")
	}

	if !strings.Contains(string(output), "required flag") || !strings.Contains(string(output), "project") {
		t.Errorf("error message should mention required project flag: %s", output)
	}
}

func TestFetch_MissingOutput(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "fetch", "sonarqube",
		"--url", "http://localhost:9000",
		"--project", "myproject",
	)

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for missing output")
	}

	if !strings.Contains(string(output), "required flag") || !strings.Contains(string(output), "output") {
		t.Errorf("error message should mention required output flag: %s", output)
	}
}

func TestFetch_SonarQube_MissingURL(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "fetch", "sonarqube",
		"--project", "myproject",
		"-o", "-",
	)
	// Explicitly clear the env var
	cmd.Env = []string{}
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "SONARQUBE_URL=") {
			cmd.Env = append(cmd.Env, e)
		}
	}

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for missing URL")
	}

	if !strings.Contains(string(output), "URL required") {
		t.Errorf("error message should mention URL required: %s", output)
	}
}

func TestFetch_SonarQube_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	bin := buildBinary(t)

	cmd := exec.Command(bin, "fetch", "sonarqube",
		"--url", server.URL,
		"--project", "myproject",
		"-o", "-",
	)

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for server error")
	}

	if !strings.Contains(string(output), "500") {
		t.Errorf("error message should mention status code: %s", output)
	}
}

func TestFetch_VerboseOutput(t *testing.T) {
	server := httptest.NewServer(sonarqubeHandler(t))
	defer server.Close()

	bin := buildBinary(t)
	outputPath := filepath.Join(t.TempDir(), "report.json")

	cmd := exec.Command(bin, "fetch", "sonarqube",
		"--url", server.URL,
		"--project", "myproject",
		"-o", outputPath,
		"-v",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fetch verbose failed: %v\n%s", err, output)
	}

	// Verbose output should include score info
	if !strings.Contains(string(output), "Score:") {
		t.Errorf("verbose output should contain Score: %s", output)
	}
}

func TestNopWriteCloser_Close(t *testing.T) {
	var buf strings.Builder
	nwc := nopWriteCloser{&buf}

	// Write some data
	n, err := nwc.Write([]byte("test data"))
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
	if n != 9 {
		t.Errorf("Write() = %d, want 9", n)
	}

	// Close should return nil
	if err := nwc.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	// Verify written data
	if buf.String() != "test data" {
		t.Errorf("written data = %q, want %q", buf.String(), "test data")
	}
}
