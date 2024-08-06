/*
Copyright © 2024 Stany Helberty stanyhelberth@gmail.com
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
		fmt.Println("0.9.0")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
