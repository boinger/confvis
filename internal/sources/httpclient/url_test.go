package httpclient

import "testing"

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		defaultURL string
		want       string
	}{
		{
			name:       "empty URL uses default",
			baseURL:    "",
			defaultURL: "https://api.example.com",
			want:       "https://api.example.com",
		},
		{
			name:       "removes trailing slash",
			baseURL:    "https://api.example.com/",
			defaultURL: "https://default.com",
			want:       "https://api.example.com",
		},
		{
			name:       "removes multiple trailing slashes",
			baseURL:    "https://api.example.com///",
			defaultURL: "https://default.com",
			want:       "https://api.example.com",
		},
		{
			name:       "preserves URL without trailing slash",
			baseURL:    "https://api.example.com",
			defaultURL: "https://default.com",
			want:       "https://api.example.com",
		},
		{
			name:       "default URL with trailing slash",
			baseURL:    "",
			defaultURL: "https://api.example.com/",
			want:       "https://api.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeBaseURL(tt.baseURL, tt.defaultURL)
			if got != tt.want {
				t.Errorf("NormalizeBaseURL(%q, %q) = %q, want %q", tt.baseURL, tt.defaultURL, got, tt.want)
			}
		})
	}
}
