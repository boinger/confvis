package trufflehog

import "testing"

func TestClient_NewClient(t *testing.T) {
	// Default command
	c := NewClient("")
	if c.command != DefaultCommand {
		t.Errorf("NewClient(\"\").command = %q, want %q", c.command, DefaultCommand)
	}

	// Custom command
	c = NewClient("custom-trufflehog")
	if c.command != "custom-trufflehog" {
		t.Errorf("NewClient(custom).command = %q, want %q", c.command, "custom-trufflehog")
	}
}

func TestParseJSONLines(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		wantCount int
		wantErr   bool
	}{
		{
			name:      "single finding",
			input:     []byte(`{"Verified":true,"DetectorName":"AWS","Raw":"AKIA..."}`),
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "multiple findings",
			input: []byte(`{"Verified":true,"DetectorName":"AWS","Raw":"AKIA..."}
{"Verified":false,"DetectorName":"GitHub","Raw":"ghp_..."}`),
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "empty input",
			input:     []byte(""),
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "blank lines ignored",
			input:     []byte("\n\n  \n"),
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "blank lines between findings",
			input: []byte(`{"Verified":true,"DetectorName":"AWS","Raw":"AKIA..."}

{"Verified":false,"DetectorName":"GitHub","Raw":"ghp_..."}`),
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:    "invalid JSON",
			input:   []byte(`not valid json`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings, err := parseJSONLines(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("parseJSONLines() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseJSONLines() error = %v, want nil", err)
			}
			if len(findings) != tt.wantCount {
				t.Errorf("parseJSONLines() returned %d findings, want %d", len(findings), tt.wantCount)
			}
		})
	}
}

func TestCountFindingsByVerification(t *testing.T) {
	tests := []struct {
		name           string
		findings       []Finding
		wantVerified   int
		wantUnverified int
	}{
		{
			name: "mixed verified and unverified",
			findings: []Finding{
				{Verified: true},
				{Verified: false},
				{Verified: true},
			},
			wantVerified:   2,
			wantUnverified: 1,
		},
		{
			name: "all verified",
			findings: []Finding{
				{Verified: true},
				{Verified: true},
			},
			wantVerified:   2,
			wantUnverified: 0,
		},
		{
			name: "all unverified",
			findings: []Finding{
				{Verified: false},
				{Verified: false},
			},
			wantVerified:   0,
			wantUnverified: 2,
		},
		{
			name:           "empty findings",
			findings:       []Finding{},
			wantVerified:   0,
			wantUnverified: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counts := countFindingsByVerification(tt.findings)
			if counts.Verified != tt.wantVerified {
				t.Errorf("Verified = %d, want %d", counts.Verified, tt.wantVerified)
			}
			if counts.Unverified != tt.wantUnverified {
				t.Errorf("Unverified = %d, want %d", counts.Unverified, tt.wantUnverified)
			}
		})
	}
}
