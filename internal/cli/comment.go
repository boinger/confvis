package cli

import (
	"github.com/spf13/cobra"
)

var commentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Post confidence reports as PR comments",
	Long: `Post confidence reports as comments on pull requests.

Confvis can post formatted confidence reports directly to PRs
without needing external actions like peter-evans/create-or-update-comment.`,
}

func init() {
	rootCmd.AddCommand(commentCmd)
}
