package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze newsletters in your inbox",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("📬 (Coming soon) Analyze newsletters and show stats...")
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
}
