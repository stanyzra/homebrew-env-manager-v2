/*
Copyright © 2025 Stany Helberth stanyhelberth@gmail.com
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/amplify"
	"github.com/digitalocean/godo"
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
		isK8s, err := cmd.Flags().GetBool("k8s")
		if err != nil {
			log.Fatalf("Error reading option flag: %v", err)
		}

		project, err := utils.GetFlagString(cmd, "project", utils.ValidProjects, false)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		envType, err := utils.GetFlagString(cmd, "type", utils.ValidTypes, false)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		projEnvironment, err := utils.GetFlagString(cmd, "environment", utils.ValidEnvs, true)
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

		// provider, err := utils.GetConfigProperty(project, projEnvironment, "provider")
		provider, err := utils.GetConfigProperty("\""+project+"\"", projEnvironment+".provider")

		if err != nil {
			fmt.Println("Error getting provider: ", err)
			return
		}

		projEnvironmentList := []string{projEnvironment}

		if projEnvironment == "all" {
			projEnvironmentList = utils.ValidEnvs
		}

		for _, projEnv := range projEnvironmentList {
			switch provider {
			case "OCI":
				fileName := fmt.Sprintf("%s_%s", projEnv, envType)

				configProvider, _, err := utils.GetConfigProviderOCI()

				if err != nil {
					fmt.Println("Error getting config provider: ", err)
					return
				}

				ociNamespace, err := utils.GetConfigProperty("OCI", "namespace")

				if err != nil {
					fmt.Println("Error getting namespace: ", err)
					return
				}

				client, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(configProvider)
				helpers.FatalIfError(err)

				if filePath != "" {
					fmt.Printf("Deleting from file: %s\n", filePath)
					DeleteFromFile(client, ociNamespace, project, projEnv, envType, filePath, fileName, isQuiet, isK8s)
				} else {
					DeleteFromArgs(client, ociNamespace, project, projEnv, envType, args, fileName, isQuiet, isK8s)
				}
			case "AWS":
				// projEnv, err = utils.GetConfigProperty(project, projEnv, "branch_name")
				projEnv, err = utils.GetConfigProperty("\""+project+"\"", projEnvironment+".branch_name")

				if err != nil {
					fmt.Println("Error getting project environment: ", err)
					return
				}

				configProvider, _, err := utils.GetConfigProviderAWS()
				if err != nil {
					fmt.Println("Error getting config provider: ", err)
					return
				}

				client := amplify.NewFromConfig(configProvider)
				utils.HandleAWS(client, project, projEnv, false, filePath, args, "", "", isQuiet, cmd.Name())

			case "DGO":
				client, err := utils.GetClientDGO()
				if err != nil {
					fmt.Println("Error getting client: ", err)
					return
				}

				DeleteDGOEnv(client, project, projEnv, filePath, args, isQuiet)

			default:
				fmt.Println("Invalid provider")
				return
			}
		}
	},
}

func DeleteDGOEnv(client *godo.Client, project string, projEnvironment string, filePath string, envNames []string, isQuiet bool) {
	// dgoAppName, err := utils.GetConfigProperty(project, projEnvironment, "app_name")
	dgoAppName, err := utils.GetConfigProperty("\""+project+"\"", projEnvironment+".app_name")

	if err != nil {
		fmt.Println("Error getting app name: ", err)
		return
	}

	dgoApp := utils.GetDGOApp(client, dgoAppName)
	isSaved := false

	deleteEnvs := func(component *godo.AppStaticSiteSpec) (bool, error) {
		envsAsIni := utils.GetDGOEnvsAsIni(component.Envs)

		if filePath == "" {
			isSaved = utils.DeleteEnvironmentVariables(envsAsIni, envNames)
		} else {
			userEnvFile, err := ini.Load(filePath)
			if err != nil {
				fmt.Println("Error loading file: ", err)
				if _, err := os.Stat(filePath); err == nil {
					fmt.Println("Are you sure the file are in INI format (<key>=<value>)?")
				}
				return false, nil
			}

			isSaved = utils.DeleteEnvironmentVariables(envsAsIni, userEnvFile.Section("").KeyStrings())
		}

		if isSaved {
			component.Envs = utils.GetDGOEnvsFromIni(envsAsIni)
		}

		return isSaved, nil
	}

	if isQuiet || utils.GetUserPermission("Are you sure you want to delete the environment variables?") {
		err := utils.UpdateDGOApp(client, project, dgoApp, deleteEnvs)
		if err != nil {
			fmt.Println("Error updating app: ", err)
			return
		}

		if isSaved {
			fmt.Printf("Environment variables deleted in project \"%s\" in \"prod\" environment\n", project)
		}
	}

}

func ConfirmAndSave(client objectstorage.ObjectStorageClient, namespace, project, fileName, projEnvironment string, envFile *ini.File, isQuiet bool) {
	if isQuiet || utils.GetUserPermission("Are you sure you want to delete the environment variables?") {
		utils.SaveEnvFile(client, namespace, project, fileName, envFile, utils.BucketName)
		fmt.Printf("Environment variables deleted in project \"%s\" in \"%s\" environment\n", project, projEnvironment)
	}
}

func DeleteFromArgs(client objectstorage.ObjectStorageClient, namespace string, project string, projEnvironment string, envType string, envNames []string, fileName string, isQuiet bool, isK8s bool) {
	fmt.Println("Deleting from args")

	envFile, err := utils.GetEnvsFileAsIni(project, fileName, client, namespace, utils.BucketName)
	if err != nil {
		fmt.Println("Error getting environment file: ", err)
		return
	}

	if utils.DeleteEnvironmentVariables(envFile, envNames) {
		if isK8s {
			k8sClient, err := utils.GetK8sClient()
			if err != nil {
				log.Fatalf("Error getting Kubernetes client: %v", err)
			}
			manager, resourceName := utils.GetK8sResourceDataParams(k8sClient, project, projEnvironment, envType)

			err = utils.DeleteK8sResourceKey(manager, resourceName, envNames)

			if err != nil {
				log.Fatalf("Failed to update resource data: %v", err)
			}
		}
		ConfirmAndSave(client, namespace, project, fileName, projEnvironment, envFile, isQuiet)
	}
}

func DeleteFromFile(client objectstorage.ObjectStorageClient, namespace string, project string, projEnvironment string, envType string, filePath string, fileName string, isQuiet bool, isK8s bool) {
	userEnvFile, err := ini.Load(filePath)
	if err != nil {
		fmt.Println("Error loading file: ", err)
		if _, err := os.Stat(filePath); err == nil {
			fmt.Println("Are you sure the file are in INI format (<key>=<value>)?")
		}
		return
	}

	envFile, err := utils.GetEnvsFileAsIni(project, fileName, client, namespace, utils.BucketName)

	if err != nil {
		fmt.Println("Error loading file: ", err)
		return
	}

	if utils.DeleteEnvironmentVariables(envFile, userEnvFile.Section("").KeyStrings()) {
		if isK8s {
			k8sClient, err := utils.GetK8sClient()
			if err != nil {
				log.Fatalf("Error getting Kubernetes client: %v", err)
			}
			manager, resourceName := utils.GetK8sResourceDataParams(k8sClient, project, projEnvironment, envType)

			err = utils.DeleteK8sResourceKey(manager, resourceName, userEnvFile.Section("").KeyStrings())

			if err != nil {
				log.Fatalf("Failed to update resource data: %v", err)
			}
		}
		ConfirmAndSave(client, namespace, project, fileName, projEnvironment, envFile, isQuiet)
	}
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	validTypesStr := strings.Join(utils.ValidTypes, ", ")
	validProjectsStr := strings.Join(utils.ValidProjects, ", ")
	validProjectEnvsStr := strings.Join(append(utils.ValidEnvs, "all"), ", ")

	deleteCmd.Flags().StringP("type", "t", "envs", fmt.Sprintf("Specify the environment variable type (options: %s)", validTypesStr))
	deleteCmd.Flags().StringP("project", "p", "", fmt.Sprintf("Specify the project name (options: %s)", validProjectsStr))
	deleteCmd.Flags().StringP("environment", "e", "", fmt.Sprintf("Specify the project environment (options: %s)", validProjectEnvsStr))
	deleteCmd.Flags().StringP("file", "f", "", "Specify a file containing a list of environment variables or secrets. The file should be in INI format.")
	deleteCmd.Flags().Bool("quiet", false, "Don't ask for confirmation before deleting the environment variable or secret")
	deleteCmd.Flags().BoolP("k8s", "k", false, "Delete the environment variable or secret from the Kubernetes cluster")

	deleteCmd.MarkFlagRequired("project")
	deleteCmd.MarkFlagRequired("environment")
}
