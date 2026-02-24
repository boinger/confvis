package cli

import (
	"fmt"

	"github.com/boinger/confvis/internal/history"
)

// loadHistoryScores reads historical scores from git ref or file storage.
func loadHistoryScores(deps *GaugeDeps, useGitRef bool, historyRef, historyFile string) ([]int, error) {
	var hist *history.History
	var err error

	if useGitRef && historyRef != "" {
		hist, err = deps.GitRefReader(historyRef)
		if err != nil {
			return nil, fmt.Errorf("reading history from git ref: %w", err)
		}
	} else if historyFile != "" {
		hist, err = deps.HistoryReader(historyFile)
		if err != nil {
			return nil, fmt.Errorf("reading history: %w", err)
		}
	}

	if hist == nil {
		return nil, nil
	}

	var scores []int
	for _, e := range hist.Last(deps.HistoryCount - 1) {
		scores = append(scores, e.Score)
	}
	return scores, nil
}

// appendToHistory writes a score entry to the appropriate history storage.
func appendToHistory(deps *GaugeDeps, useGitRef bool, historyRef, historyFile string, score int) error {
	entry := history.NewEntry(score)
	if useGitRef && historyRef != "" {
		if err := deps.GitRefAppender(historyRef, entry); err != nil {
			return fmt.Errorf("appending to history git ref: %w", err)
		}
	} else if historyFile != "" {
		if err := deps.HistoryAppender(historyFile, entry); err != nil {
			return fmt.Errorf("appending to history: %w", err)
		}
	}
	return nil
}

// resolveHistoryStorage determines which history storage mode to use.
// Returns: useGitRef (bool), historyRef (string), historyFile (string).
// Logic:
//   - If --history-ref is explicitly set, use git ref storage
//   - If --history-auto is set, auto-detect: git ref if in repo, else file
//   - If --history-file is set, use file storage
//   - Otherwise, no history storage
func resolveHistoryStorage(deps *GaugeDeps) (useGitRef bool, historyRef string, historyFile string) {
	// Explicit --history-ref takes precedence
	if deps.HistoryRef != "" {
		return true, deps.HistoryRef, ""
	}

	// --history-auto: detect storage mode
	if deps.HistoryAuto {
		if deps.IsGitRepo != nil && deps.IsGitRepo() {
			// In a git repo, use git ref storage with default ref
			return true, history.DefaultHistoryRef, ""
		}
		// Not in a git repo, fall back to default file
		defaultFile := ".confvis-history.jsonl"
		if deps.HistoryFile != "" {
			defaultFile = deps.HistoryFile
		}
		return false, "", defaultFile
	}

	// Explicit --history-file
	if deps.HistoryFile != "" {
		return false, "", deps.HistoryFile
	}

	// No history storage configured
	return false, "", ""
}

