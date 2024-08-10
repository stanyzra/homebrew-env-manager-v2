/*
Copyright © 2024 Stany Helberty stanyhelberth@gmail.com
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/stanyzra/env-manager-v2/internal/utils"
)

var validTypes = []string{"envs", "secrets"}

var projectProviders = []utils.ProjectProvider{
	{Name: "collection-back-end-v2.1", CloudProvider: []string{"OCI"}},
	{Name: "gollection-elastic", CloudProvider: []string{"OCI"}},
	{Name: "app-memorial-collection-v2", CloudProvider: []string{"AWS", "DGO"}},
	{Name: `app-biblioteca-collection-v2`, CloudProvider: []string{"AWS", "DGO"}},
	{Name: "collection-front-end-v2.1", CloudProvider: []string{"AWS", "DGO"}},
	{Name: "app-admin-collection-v2", CloudProvider: []string{"AWS"}},
}

var validProjects = []string{"collection-back-end-v2.1", "gollection-elastic", "app-memorial-collection-v2", "app-admin-collection-v2", "collection-front-end-v2.1", "app-biblioteca-collection-v2"}
var validEnvs = []string{"prod", "beta", "homolog", "dev"}
var validAppComponents = []string{"collection-memorial-white-screen", "service", "collection-home-white-screen"}

const (
	// Bucket name
	bucketName = "collection-kubernetes-files"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "env-manager-v2",
	Short: "CLI Application to manage environment variables in Kubernetes Cluster",
	Long: `A CLI Application to manage environment variables and secrets in 
Kubernetes Cluster. It can be used to create, update, delete and list
environment variables and secrets in Kubernetes Cluster.

The environment variables and secrets are stored in ConfigMap and Secret in the
Kubernetes Cluster and stored as a key-value pair in Object Storage in OCI.
	`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.env-manager-v2.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
}
