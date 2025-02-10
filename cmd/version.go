/*
Copyright © 2025 Stany Helberth stanyhelberth@gmail.com
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version number of env-manager-v2",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("1.2.1")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
