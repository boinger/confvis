package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/boinger/confvis/internal/baseline"
)

// configKeySourcesPrefix is the viper config key prefix for source settings.
const configKeySourcesPrefix = "sources."

// initConfig initializes the viper configuration from config files and environment.
// It searches for config files in the following order:
// 1. .confvis.yaml or .confvis.yml in current directory
// 2. config.yaml in ~/.config/confvis/
//
// Precedence is: config file < environment variables < command-line flags.
func initConfig() {
	// Set config file name (without extension)
	viper.SetConfigName(".confvis")
	viper.SetConfigType("yaml")

	// Search paths (in order of precedence - later paths override earlier)
	// Current directory first
	viper.AddConfigPath(".")

	// Then user config directory
	if home, err := os.UserHomeDir(); err == nil {
		viper.AddConfigPath(filepath.Join(home, ".config", "confvis"))
	}

	// Read config (ignore if not found)
	_ = viper.ReadInConfig()

	// Enable environment variable support
	// Environment variables use CONFVIS_ prefix and replace . with _
	// e.g., gauge.width becomes CONFVIS_GAUGE_WIDTH
	viper.SetEnvPrefix("CONFVIS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}

// bindGaugeFlags binds gauge command flags to viper configuration keys.
// This allows config file values to be overridden by environment variables and flags.
func bindGaugeFlags(cmd *cobra.Command) {
	// Bind flags to viper keys
	must(viper.BindPFlag("gauge.width", cmd.Flags().Lookup("width")))
	must(viper.BindPFlag("gauge.height", cmd.Flags().Lookup("height")))
	must(viper.BindPFlag("gauge.style", cmd.Flags().Lookup("style")))
	must(viper.BindPFlag("gauge.dark", cmd.Flags().Lookup("dark")))
	must(viper.BindPFlag("gauge.fail_under", cmd.Flags().Lookup("fail-under")))
	must(viper.BindPFlag("gauge.badge_type", cmd.Flags().Lookup("badge-type")))
	must(viper.BindPFlag("gauge.history_file", cmd.Flags().Lookup("history-file")))
	must(viper.BindPFlag("gauge.history_count", cmd.Flags().Lookup("history-count")))
	must(viper.BindPFlag("gauge.history_ref", cmd.Flags().Lookup("history-ref")))
	must(viper.BindPFlag("gauge.history_auto", cmd.Flags().Lookup("history-auto")))
	must(viper.BindPFlag("gauge.green_above", cmd.Flags().Lookup("green-above")))
	must(viper.BindPFlag("gauge.yellow_above", cmd.Flags().Lookup("yellow-above")))
	must(viper.BindPFlag("gauge.compare_baseline", cmd.Flags().Lookup("compare-baseline")))
	must(viper.BindPFlag("gauge.baseline_ref", cmd.Flags().Lookup("baseline-ref")))
	must(viper.BindPFlag("gauge.baseline_file", cmd.Flags().Lookup("baseline-file")))
}

// getGateFailUnder returns the gate fail-under from CLI flag, then config/env fallback.
// Gate doesn't use BindPFlag (would conflict with gauge's binding to the same viper key).
func getGateFailUnder() int {
	if gateFailUnder > 0 {
		return gateFailUnder
	}
	return viper.GetInt("gauge.fail_under")
}

// getGateCompareBaseline returns compare-baseline from CLI flag, then config/env fallback.
func getGateCompareBaseline() bool {
	if gateCompareBaseline {
		return true
	}
	return viper.GetBool("gauge.compare_baseline")
}

// getGateBaselineRef returns baseline ref from CLI flag, then config/env fallback.
func getGateBaselineRef() string {
	if gateBaselineRef != "" {
		return gateBaselineRef
	}
	if v := viper.GetString("gauge.baseline_ref"); v != "" {
		return v
	}
	return baseline.DefaultBaselineRef
}

// getGateBaselineFile returns baseline file from CLI flag, then config/env fallback.
func getGateBaselineFile() string {
	if gateBaselineFile != "" {
		return gateBaselineFile
	}
	return viper.GetString("gauge.baseline_file")
}

// getCommentGitHubCompareBaseline returns compare-baseline from CLI flag, then config/env fallback.
func getCommentGitHubCompareBaseline() bool {
	if commentGitHubCompareBaseline {
		return true
	}
	return viper.GetBool("gauge.compare_baseline")
}

// getCommentGitHubBaselineRef returns baseline ref from CLI flag, then config/env fallback.
func getCommentGitHubBaselineRef() string {
	if commentGitHubBaselineRef != "" {
		return commentGitHubBaselineRef
	}
	if v := viper.GetString("gauge.baseline_ref"); v != "" {
		return v
	}
	return baseline.DefaultBaselineRef
}

// getCommentGitHubBaselineFile returns baseline file from CLI flag, then config/env fallback.
func getCommentGitHubBaselineFile() string {
	if commentGitHubBaselineFile != "" {
		return commentGitHubBaselineFile
	}
	return viper.GetString("gauge.baseline_file")
}

// bindFetchFlags binds fetch command flags to viper configuration keys.
func bindFetchFlags(cmd *cobra.Command) {
	must(viper.BindPFlag("fetch.timeout", cmd.Flags().Lookup("timeout")))
	must(viper.BindPFlag("fetch.threshold", cmd.Flags().Lookup("threshold")))
}

// getGaugeWidth returns the gauge width from config/env/flag with defaults.
func getGaugeWidth() int {
	if v := viper.GetInt("gauge.width"); v > 0 {
		return v
	}
	return 200 // default
}

// getGaugeHeight returns the gauge height from config/env/flag with defaults.
func getGaugeHeight() int {
	if v := viper.GetInt("gauge.height"); v > 0 {
		return v
	}
	return 120 // default
}

// getGaugeStyle returns the gauge style from config/env/flag with defaults.
func getGaugeStyle() string {
	if v := viper.GetString("gauge.style"); v != "" {
		return v
	}
	return "github" // default
}

// getGaugeDark returns the dark mode setting from config/env/flag.
func getGaugeDark() bool {
	return viper.GetBool("gauge.dark")
}

// getGaugeFailUnder returns the fail-under threshold from config/env/flag.
func getGaugeFailUnder() int {
	return viper.GetInt("gauge.fail_under")
}

// getGaugeBadgeType returns the badge type from config/env/flag with defaults.
func getGaugeBadgeType() string {
	if v := viper.GetString("gauge.badge_type"); v != "" {
		return v
	}
	return "gauge" // default
}

// getGaugeHistoryFile returns the history file path from config/env/flag.
func getGaugeHistoryFile() string {
	return viper.GetString("gauge.history_file")
}

// getGaugeHistoryCount returns the history count from config/env/flag with defaults.
func getGaugeHistoryCount() int {
	if v := viper.GetInt("gauge.history_count"); v > 0 {
		return v
	}
	return 10 // default
}

// getGaugeHistoryRef returns the history git ref from config/env/flag.
func getGaugeHistoryRef() string {
	return viper.GetString("gauge.history_ref")
}

// getGaugeHistoryAuto returns the history auto-detect setting from config/env/flag.
func getGaugeHistoryAuto() bool {
	return viper.GetBool("gauge.history_auto")
}

// getGaugeGreenAbove returns the green threshold from config/env/flag.
func getGaugeGreenAbove() int {
	return viper.GetInt("gauge.green_above")
}

// getGaugeYellowAbove returns the yellow threshold from config/env/flag.
func getGaugeYellowAbove() int {
	return viper.GetInt("gauge.yellow_above")
}

// getGaugeCompareBaseline returns the compare-baseline setting from config/env/flag.
func getGaugeCompareBaseline() bool {
	return viper.GetBool("gauge.compare_baseline")
}

// getGaugeBaselineRef returns the baseline git ref from config/env/flag with defaults.
func getGaugeBaselineRef() string {
	if v := viper.GetString("gauge.baseline_ref"); v != "" {
		return v
	}
	return baseline.DefaultBaselineRef
}

// getGaugeBaselineFile returns the baseline file path from config/env/flag.
func getGaugeBaselineFile() string {
	return viper.GetString("gauge.baseline_file")
}

// getFetchTimeout returns the fetch timeout from config/env/flag with defaults.
func getFetchTimeout() int {
	if v := viper.GetInt("fetch.timeout"); v > 0 {
		return v
	}
	return 30 // default
}

// getFetchThreshold returns the fetch threshold from config/env/flag with defaults.
func getFetchThreshold() int {
	if v := viper.GetInt("fetch.threshold"); v > 0 {
		return v
	}
	return 75 // default
}

// getSourceURL returns the source-specific URL from config.
func getSourceURL(source string) string {
	return viper.GetString(configKeySourcesPrefix + source + ".url")
}

// getSourceOrg returns the source-specific org from config.
func getSourceOrg(source string) string {
	return viper.GetString(configKeySourcesPrefix + source + ".org")
}

// getSourceService returns the source-specific service from config.
func getSourceService(source string) string {
	return viper.GetString(configKeySourcesPrefix + source + ".service")
}

// GetConfigFile returns the config file being used, if any.
func GetConfigFile() string {
	return viper.ConfigFileUsed()
}

// getGaugeFactorThresholds returns factor thresholds from config.
// Config format:
//
//	gauge:
//	  factor_thresholds:
//	    Coverage: 80
//	    Security: 90
func getGaugeFactorThresholds() map[string]int {
	result := make(map[string]int)
	thresholds := viper.GetStringMap("gauge.factor_thresholds")
	for name, v := range thresholds {
		switch val := v.(type) {
		case int:
			result[name] = val
		case float64:
			result[name] = int(val)
		case int64:
			result[name] = int(val)
		}
	}
	return result
}
