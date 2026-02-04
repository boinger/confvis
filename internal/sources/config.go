package sources

import (
	"fmt"
	"os"
	"time"
)

// ResolvedConfig holds resolved configuration for a source.
type ResolvedConfig struct {
	Token   string
	URL     string
	Timeout time.Duration
}

// ConfigResolver provides methods for resolving source configuration.
type ConfigResolver struct {
	SourceName     string
	TokenEnvVar    string
	URLEnvVar      string
	TokenRequired  bool
	URLRequired    bool
	DefaultTimeout time.Duration
}

// Resolve resolves token and URL from options or environment variables.
func (r *ConfigResolver) Resolve(opts Options) (*ResolvedConfig, error) {
	cfg := &ResolvedConfig{}

	// Resolve token
	cfg.Token = opts.Token
	if cfg.Token == "" && r.TokenEnvVar != "" {
		cfg.Token = os.Getenv(r.TokenEnvVar)
	}
	if r.TokenRequired && cfg.Token == "" {
		return nil, fmt.Errorf("%s token required: use --token flag or set %s", r.SourceName, r.TokenEnvVar)
	}

	// Resolve URL
	cfg.URL = opts.URL
	if cfg.URL == "" && r.URLEnvVar != "" {
		cfg.URL = os.Getenv(r.URLEnvVar)
	}
	if r.URLRequired && cfg.URL == "" {
		return nil, fmt.Errorf("%s URL required: use --url flag or set %s", r.SourceName, r.URLEnvVar)
	}

	// Resolve timeout
	cfg.Timeout = time.Duration(opts.Timeout) * time.Second
	if cfg.Timeout <= 0 {
		cfg.Timeout = r.DefaultTimeout
		if cfg.Timeout <= 0 {
			cfg.Timeout = 30 * time.Second
		}
	}

	return cfg, nil
}
