// Package sources provides a modular framework for fetching metrics from external systems.
package sources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/boinger/confvis/internal/confidence"
)

// Source fetches metrics from an external system and produces a confidence report.
type Source interface {
	// Name returns the unique identifier for this source (e.g., "sonarqube").
	Name() string

	// Fetch retrieves metrics and converts them to a confidence report.
	// The opts map contains source-specific configuration (URL, project, token, etc.).
	Fetch(ctx context.Context, opts Options) (*confidence.Report, error)
}

// Options holds configuration for fetching from a source.
type Options struct {
	// Common options
	URL       string
	Project   string
	Token     string
	Branch    string
	Title     string
	Threshold int
	Timeout   int // seconds

	// Source-specific options stored by key
	Extra map[string]string
}

// Registry holds available sources by name.
var Registry = make(map[string]Source)

// Register adds a source to the registry.
// Panics if a source with the same name already exists.
func Register(s Source) {
	name := s.Name()
	if _, exists := Registry[name]; exists {
		panic(fmt.Sprintf("source %q already registered", name))
	}
	Registry[name] = s
}

// Get returns a source by name, or nil if not found.
func Get(name string) Source {
	return Registry[name]
}

// ResolveCommand resolves a command path from Extra options or an environment variable.
func ResolveCommand(opts Options, extraKey, envVar string) string {
	command := ""
	if opts.Extra != nil {
		command = opts.Extra[extraKey]
	}
	if command == "" {
		command = os.Getenv(envVar)
	}
	return command
}

// Names returns a sorted list of registered source names.
func Names() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DeriveTitleFromPath returns a friendly title from a file path.
// If an explicit title is provided, it returns that.
// Otherwise, resolves the path to an absolute path and returns the base name.
func DeriveTitleFromPath(path, explicitTitle string) string {
	if explicitTitle != "" {
		return explicitTitle
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	return filepath.Base(absPath)
}

// GetExtra retrieves a value from opts.Extra, returning defaultValue if not found or empty.
func GetExtra(opts Options, key, defaultValue string) string {
	if opts.Extra != nil {
		if v, ok := opts.Extra[key]; ok && v != "" {
			return v
		}
	}
	return defaultValue
}

// ResolveTitle returns the first non-empty string from the provided values.
// Typically used as ResolveTitle(opts.Title, derivedTitle, fallback).
func ResolveTitle(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
