// Package cli provides the command-line interface for confvis.
package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	verbose bool
	quiet   bool
)

var rootCmd = &cobra.Command{
	Use:   "confvis",
	Short: "Generate confidence visualization badges and dashboards",
	Long: `confvis generates visual representations of confidence scores.

It reads JSON confidence reports and produces:
- SVG gauge badges showing the overall score
- HTML dashboards with detailed factor breakdowns`,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress non-error output")
}
