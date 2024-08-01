/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/amplify"
	"github.com/oracle/oci-go-sdk/example/helpers"
	"github.com/oracle/oci-go-sdk/v49/objectstorage"
	"github.com/spf13/cobra"
	"github.com/stanyzra/env-manager-v2/internal/utils"
	"gopkg.in/ini.v1"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use: "create [flags] -p <project-name> -e <project-environment> (-n <name> -v <value>|--file <file>)",
	Example: `env-manager-v2 create -p collection-back-end-v2.1 -e dev -t envs -n foo -v bar
env-manager-v2 create -p gollection-elastic -e homolog -t secrets -n moo -v baz
env-manager-v2 create -p collection-back-end-v2.1 -e dev -t envs -f /path/to/file`,
	Short: "Create a new environment variable or secret",
	Long: `Create a new environment variable or secret and stores it
in the environment file in OCI Object Storage. The project and environment flags are
required. If the file flag is used, the name and value flags are ignored. If a environment
variable or secret with the same name already exists, it will not be created. Use the
update command to update an existing environment variable or secret.`,
	Args: func(cmd *cobra.Command, args []string) error {
		filePath, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatalf("Error reading option flag: %v", err)
		}

		envName, err := cmd.Flags().GetString("name")
		if err != nil {
			log.Fatalf("Error reading option flag: %v", err)
		}

		envValue, err := cmd.Flags().GetString("value")
		if err != nil {
			log.Fatalf("Error reading option flag: %v", err)
		}

		if filePath == "" && (envName == "" || envValue == "") {
			return fmt.Errorf("requires --name and --value flags unless --file is used")
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

		envName, err := cmd.Flags().GetString("name")
		if err != nil {
			log.Fatalf("Error reading option flag: %v", err)
		}

		envValue, err := cmd.Flags().GetString("value")
		if err != nil {
			log.Fatalf("Error reading option flag: %v", err)
		}

		cloudProvider := utils.GetCloudProvider(project, projectProviders)

		if utils.StringInSlice("OCI", cloudProvider) {
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
				CreateEnvFromFile(client, namespace, project, projEnvironment, envType, fileName, filePath)
			} else {
				CreateSingleEnv(client, namespace, project, projEnvironment, envType, envName, envValue, fileName)
			}

		} else if utils.StringInSlice("DGO", cloudProvider) && projEnvironment == "prod" {
			fmt.Println("DGO")

		} else if utils.StringInSlice("AWS", cloudProvider) {
			projEnvironment = utils.CastBranchName(projEnvironment)
			configProvider, _, err := utils.GetConfigProviderAWS()
			if err != nil {
				fmt.Println("Error getting config provider: ", err)
				return
			}

			client := amplify.NewFromConfig(configProvider)
			utils.HandleAWS(client, project, projEnvironment, false, filePath, args, envName, envValue, cmd.Name())
		}

	},
}

func CreateEnvFromFile(client objectstorage.ObjectStorageClient, namespace string, project string, projEnvironment string, envType string, fileName string, filePath string) {
	userEnvFile, err := ini.Load(filePath)
	if err != nil {
		fmt.Println("Error loading file: ", err)
		return
	}

	envFile, err := utils.GetEnvsFileAsIni(project, fileName, client, namespace, bucketName)

	if err != nil {
		fmt.Println("Error loading file: ", err)
		return
	}

	if utils.CreateEnvironmentVariables(envFile, userEnvFile) {
		utils.SaveEnvFile(client, namespace, project, fileName, envFile, bucketName)
		fmt.Printf("Environment variables saved in project \"%s\" in \"%s\" environment\n", project, projEnvironment)
	}
}

func CreateSingleEnv(client objectstorage.ObjectStorageClient, namespace string, project string, projEnvironment string, envType string, envName string, envValue string, fileName string) {
	envFile, err := utils.GetEnvsFileAsIni(project, fileName, client, namespace, bucketName)

	if err != nil {
		fmt.Println("Error loading file: ", err)
		return
	}

	if envFile.Section("").HasKey(envName) {
		fmt.Printf("[WARNING] Environment variable \"%s\" already exists in project \"%s\" in \"%s\" environment\n", envName, project, projEnvironment)
		return
	}

	envFile.Section("").Key(envName).SetValue(envValue)
	utils.SaveEnvFile(client, namespace, project, fileName, envFile, bucketName)
	fmt.Printf("Environment variables saved in project \"%s\" in \"%s\" environment\n", project, projEnvironment)
}

func init() {
	rootCmd.AddCommand(createCmd)

	validTypesStr := strings.Join(validTypes, ", ")
	validProjectsStr := strings.Join(validProjects, ", ")
	validProjectEnvsStr := strings.Join(validEnvs, ", ")

	createCmd.Flags().StringP("type", "t", "envs", fmt.Sprintf("Specify the environment variable type (options: %s)", validTypesStr))
	createCmd.Flags().StringP("project", "p", "", fmt.Sprintf("Specify the project name (options: %s)", validProjectsStr))
	createCmd.Flags().StringP("environment", "e", "", fmt.Sprintf("Specify the project environment (options: %s)", validProjectEnvsStr))
	createCmd.Flags().StringP("name", "n", "", "Specify the environment variable or secret name")
	createCmd.Flags().StringP("value", "v", "", "Specify the environment variable or secret value")
	createCmd.Flags().StringP("file", "f", "", "Specify a file containing a list of environment variables or secrets. The file should be in INI format.")

	createCmd.MarkFlagRequired("project")
	createCmd.MarkFlagRequired("environment")
}
