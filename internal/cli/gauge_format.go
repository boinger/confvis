package cli

import (
	"fmt"
	"io"

	"github.com/boinger/confvis/internal/confidence"
)

// writeFormatOutput dispatches to the appropriate format writer.
func writeFormatOutput(w io.Writer, format string, report *confidence.Report, baselineReport *confidence.Report, delta int, deps *GaugeDeps) error {
	switch format {
	case "svg":
		return generateSVGBadge(w, report, deps)
	case "json":
		return writeJSON(w, report, baselineReport, delta)
	case "text":
		return writeText(w, report.ScoreValue(), baselineReport, delta)
	case "markdown":
		return writeMarkdown(w, report, baselineReport, delta)
	case "github-comment":
		return writeGitHubComment(w, report, baselineReport)
	}
	return nil
}

// writeJSON generates JSON output for the report.
func writeJSON(w io.Writer, report *confidence.Report, baselineReport *confidence.Report, delta int) error {
	output := struct {
		Title       string `json:"title"`
		Score       int    `json:"score"`
		Threshold   int    `json:"threshold"`
		Passed      bool   `json:"passed"`
		Version     string `json:"version,omitempty"`
		GeneratedAt string `json:"generatedAt,omitempty"`
		Source      string `json:"source,omitempty"`
		Baseline    *int   `json:"baseline,omitempty"`
		Delta       *int   `json:"delta,omitempty"`
	}{
		Title:       report.Title,
		Score:       report.ScoreValue(),
		Threshold:   report.Threshold,
		Passed:      report.Passed(),
		Version:     report.Version,
		GeneratedAt: report.GeneratedAt,
		Source:      report.Source,
	}
	if baselineReport != nil {
		baselineScore := baselineReport.ScoreValue()
		output.Baseline = &baselineScore
		output.Delta = &delta
	}
	return encodeJSONIndented(w, output)
}

// writeText generates plain text output for the report.
func writeText(w io.Writer, score int, baselineReport *confidence.Report, delta int) error {
	if baselineReport != nil {
		sign := "+"
		if delta < 0 {
			sign = ""
		}
		if _, err := fmt.Fprintf(w, "%d (%s%d)\n", score, sign, delta); err != nil {
			return fmt.Errorf("writing text output: %w", err)
		}
	} else {
		if _, err := fmt.Fprintf(w, "%d\n", score); err != nil {
			return fmt.Errorf("writing text output: %w", err)
		}
	}
	return nil
}

// writeMarkdown generates markdown output for the report.
func writeMarkdown(w io.Writer, report *confidence.Report, baselineReport *confidence.Report, delta int) error {
	status := report.EffectivePassLabel()
	if !report.Passed() {
		status = report.EffectiveFailLabel()
	}

	// Header with title, score, and status
	if baselineReport != nil {
		sign := "+"
		if delta < 0 {
			sign = ""
		}
		if _, err := fmt.Fprintf(w, "## %s: %d%% (%s) [%s%d from %d%%]\n\n", report.Title, report.ScoreValue(), status, sign, delta, baselineReport.ScoreValue()); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(w, "## %s: %d%% (%s)\n\n", report.Title, report.ScoreValue(), status); err != nil {
			return err
		}
	}

	// Description if present
	if report.Description != "" {
		if _, err := fmt.Fprintf(w, "%s\n\n", report.Description); err != nil {
			return err
		}
	}

	// Factors table if present
	if len(report.Factors) > 0 {
		if _, err := fmt.Fprintln(w, "| Factor | Score | Weight |"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "|--------|------:|-------:|"); err != nil {
			return err
		}
		for _, f := range report.Factors {
			name := f.Name
			if f.URL != "" {
				name = fmt.Sprintf("[%s](%s)", f.Name, f.URL)
			}
			if _, err := fmt.Fprintf(w, "| %s | %d%% | %d%% |\n", name, f.Score, f.Weight); err != nil {
				return err
			}
		}
	}

	return nil
}

// writeGitHubComment generates GitHub-flavored markdown output for PR comments.
func writeGitHubComment(w io.Writer, report *confidence.Report, baselineReport *confidence.Report) error {
	if err := writeGitHubCommentHeader(w, report); err != nil {
		return err
	}

	if err := writeGitHubCommentBaseline(w, report, baselineReport); err != nil {
		return err
	}

	if err := writeGitHubCommentFactors(w, report.Factors); err != nil {
		return err
	}

	version := rootCmd.Version
	if version == "" {
		version = report.Version
	}
	return writeGitHubCommentFooter(w, version)
}

func writeGitHubCommentHeader(w io.Writer, report *confidence.Report) error {
	statusEmoji := ":white_check_mark:"
	statusText := "Passed"
	if !report.Passed() {
		statusEmoji = ":x:"
		statusText = "Failed"
	}

	if _, err := fmt.Fprintf(w, "## Confidence Report: %s\n\n", report.Title); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "| Metric | Value |"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "|--------|-------|"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "| Score | **%d%%** %s |\n", report.ScoreValue(), statusEmoji); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "| Threshold | %d%% |\n", report.Threshold); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "| Status | %s |\n", statusText)
	return err
}

func writeGitHubCommentBaseline(w io.Writer, report *confidence.Report, baselineReport *confidence.Report) error {
	if baselineReport == nil {
		return nil
	}
	delta := report.ScoreValue() - baselineReport.ScoreValue()
	deltaEmoji := ":left_right_arrow:"
	if delta > 0 {
		deltaEmoji = ":arrow_up:"
	} else if delta < 0 {
		deltaEmoji = ":arrow_down:"
	}
	_, err := fmt.Fprintf(w, "| Change | %+d %s |\n", delta, deltaEmoji)
	return err
}

func writeGitHubCommentFactors(w io.Writer, factors []confidence.Factor) error {
	if len(factors) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, "\n<details>"); err != nil {
		return err
	}
	if _, err := fmt.Fprint(w, "<summary>Factor Breakdown</summary>\n\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "| Factor | Score | Weight | Description |"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "|--------|-------|--------|-------------|"); err != nil {
		return err
	}
	for _, f := range factors {
		desc := f.Description
		if desc == "" {
			desc = "-"
		}
		if _, err := fmt.Fprintf(w, "| %s | %d | %d | %s |\n", f.Name, f.Score, f.Weight, desc); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w, "\n</details>")
	return err
}

func writeGitHubCommentFooter(w io.Writer, version string) error {
	if version == "" {
		version = "unknown"
	}
	_, err := fmt.Fprintf(w, "\n---\n<sub>Generated by confvis v%s</sub>\n", version)
	return err
}
