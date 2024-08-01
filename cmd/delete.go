/*
Copyright © 2024 NAME HERE stanyhelberth@gmail.com
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/oracle/oci-go-sdk/example/helpers"
	"github.com/oracle/oci-go-sdk/v49/objectstorage"
	"github.com/spf13/cobra"
	"github.com/stanyzra/env-manager-v2/internal/utils"
	"gopkg.in/ini.v1"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use: "delete [flags] -p <project-name> -e <project-environment> (<name> |--file <file>)",
	Example: `env-manager-v2 delete -p collection-back-end-v2.1 -e dev -t envs foo bar
env-manager-v2 delete -p gollection-elastic -e homolog -t secrets -f /path/to/file`,
	Short: "Delete a environment variable or secret",
	Long: `Delete a environment variable or secret from the environment file in OCI Object Storage.
The project and environment flags are required. You can delete multiple environment variables or secrets
by passing multiple names in the command's arguments or using a file. If the file flag is used, the name
flag is ignored. The file should be in INI format WITH keys and values, even though the values are not used.`,
	Args: func(cmd *cobra.Command, args []string) error {
		filePath, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatalf("Error reading option flag: %v", err)
		}

		if filePath == "" && len(args) == 0 {
			return fmt.Errorf("requires at least one name argument unless --file is used")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		project, err := utils.GetFlagString(cmd, "project", validProjects)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		envType, err := utils.GetFlagString(cmd, "type", validTypes)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		projEnvironment, err := utils.GetFlagString(cmd, "environment", validEnvs)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		filePath, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatalf("Error reading option flag: %v", err)
		}

		isQuiet, err := cmd.Flags().GetBool("quiet")
		if err != nil {
			log.Fatalf("Error reading option flag: %v", err)
		}

		fileName := fmt.Sprintf("%s_%s", projEnvironment, envType)

		configProvider, configFileName, err := utils.GetConfigProviderOCI()

		if err != nil {
			fmt.Println("Error getting config provider: ", err)
			return
		}

		ini_config, err := ini.Load(configFileName)
		if err != nil {
			fmt.Println("Error loading config file: ", err)
			return
		}

		sec := ini_config.Section("OCI")
		namespace := sec.Key("namespace").String()

		if err != nil {
			fmt.Println("Error getting config provider: ", err)
			return
		}

		client, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(configProvider)
		helpers.FatalIfError(err)

		if filePath != "" {
			fmt.Printf("Deleting from file: %s\n", filePath)
			DeleteFromFile(client, namespace, project, projEnvironment, envType, filePath, fileName, isQuiet)
		} else {
			DeleteFromArgs(client, namespace, project, projEnvironment, envType, args, fileName, isQuiet)
		}
	},
}

func DeleteEnvironmentVariables(envFile *ini.File, envNames []string) bool {
	isSaved := false
	for _, envName := range envNames {
		sec := envFile.Section("")
		if !sec.HasKey(envName) {
			fmt.Printf("Environment variable %s not found\n", envName)
			continue
		}
		sec.DeleteKey(envName)
		isSaved = true
	}
	return isSaved
}

func ConfirmAndSave(client objectstorage.ObjectStorageClient, namespace, project, fileName, projEnvironment string, envFile *ini.File, isQuiet bool) {
	if isQuiet || utils.GetUserPermission("Are you sure you want to delete the environment variables?") {
		utils.SaveEnvFile(client, namespace, project, fileName, envFile, bucketName)
		fmt.Printf("Environment variables deleted in project \"%s\" in \"%s\" environment\n", project, projEnvironment)
	}
}

func DeleteFromArgs(client objectstorage.ObjectStorageClient, namespace, project, projEnvironment, envType string, envNames []string, fileName string, isQuiet bool) {
	fmt.Println("Deleting from args")

	envFile, err := utils.GetEnvsFileAsIni(project, fileName, client, namespace, bucketName)
	if err != nil {
		fmt.Println("Error getting environment file: ", err)
		return
	}

	if DeleteEnvironmentVariables(envFile, envNames) {
		ConfirmAndSave(client, namespace, project, fileName, projEnvironment, envFile, isQuiet)
	}
}

func DeleteFromFile(client objectstorage.ObjectStorageClient, namespace, project, projEnvironment, envType, filePath, fileName string, isQuiet bool) {
	userEnvFile, err := ini.Load(filePath)
	if err != nil {
		fmt.Println("Error loading file: ", err)
		if _, err := os.Stat(filePath); err == nil {
			fmt.Println("Are you sure the file are in INI format (<key>=<value>)?")
		}
		return
	}

	envFile, err := utils.GetEnvsFileAsIni(project, fileName, client, namespace, bucketName)

	if err != nil {
		fmt.Println("Error loading file: ", err)
		return
	}

	if DeleteEnvironmentVariables(envFile, userEnvFile.Section("").KeyStrings()) {
		ConfirmAndSave(client, namespace, project, fileName, projEnvironment, envFile, isQuiet)
	}
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	validTypesStr := strings.Join(validTypes, ", ")
	validProjectsStr := strings.Join(validProjects, ", ")
	validProjectEnvsStr := strings.Join(validEnvs, ", ")

	deleteCmd.Flags().StringP("type", "t", "envs", fmt.Sprintf("Specify the environment variable type (options: %s)", validTypesStr))
	deleteCmd.Flags().StringP("project", "p", "", fmt.Sprintf("Specify the project name (options: %s)", validProjectsStr))
	deleteCmd.Flags().StringP("environment", "e", "", fmt.Sprintf("Specify the project environment (options: %s)", validProjectEnvsStr))
	deleteCmd.Flags().StringP("file", "f", "", "Specify a file containing a list of environment variables or secrets. The file should be in INI format.")
	deleteCmd.Flags().Bool("quiet", false, "Don't ask for confirmation before deleting the environment variable or secret")

	deleteCmd.MarkFlagRequired("project")
	deleteCmd.MarkFlagRequired("environment")
}
