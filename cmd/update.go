/*
Copyright © 2025 Stany Helberth stanyhelberth@gmail.com
*/
package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/amplify"
	"github.com/digitalocean/godo"
	"github.com/oracle/oci-go-sdk/example/helpers"
	"github.com/oracle/oci-go-sdk/v49/objectstorage"
	"github.com/spf13/cobra"
	"github.com/stanyzra/env-manager-v2/internal/utils"
	"gopkg.in/ini.v1"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update [flags] -p <project-name> -e <project-environment> (-n <name> -v <value>|--file <file>)",
	Short: "Update a environment variable or secret",
	Example: `env-manager-v2 update -p collection-back-end-v2.1 -e dev -t envs -n foo -v bar
env-manager-v2 update -p gollection-elastic -e homolog -t secrets -n moo -v baz
env-manager-v2 update -p collection-back-end-v2.1 -e dev -t envs -f /path/to/file`,
	Long: `Update a environment variable or secret from the environment file in OCI Object Storage.
The project and environment flags are required. You can update multiple environment variables or secrets
using a file. If the file flag is used, the name and valueflags is ignored. The file should be
in INI format WITH keys and values. If a environment variable or secret doesn't exists,
it will not be created. Use the create command to create a new environment variable or secret.`,
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

		envName, err := cmd.Flags().GetString("name")
		if err != nil {
			log.Fatalf("Error reading option flag: %v", err)
		}

		envValue, err := cmd.Flags().GetString("value")
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
					UpdateEnvFromFile(client, ociNamespace, project, projEnv, envType, fileName, filePath, isK8s)
				} else {
					UpdateSingleEnv(client, ociNamespace, project, projEnv, envType, envName, envValue, fileName, isK8s)
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
				utils.HandleAWS(client, project, projEnv, false, filePath, args, envName, envValue, false, cmd.Name())

			case "DGO":
				client, err := utils.GetClientDGO()
				if err != nil {
					fmt.Println("Error getting client: ", err)
					return
				}

				UpdateDGOEnv(client, project, filePath, projEnv, envName, envValue)

			default:
				fmt.Println("Invalid provider")
				return
			}
		}
	},
}

func UpdateDGOEnv(client *godo.Client, project string, filePath string, projEnvironment string, envName string, envValue string) {
	// dgoAppName, err := utils.GetConfigProperty(project, projEnvironment, "app_name")
	dgoAppName, err := utils.GetConfigProperty("\""+project+"\"", projEnvironment+".app_name")

	if err != nil {
		fmt.Println("Error getting app name: ", err)
		return
	}

	dgoApp := utils.GetDGOApp(client, dgoAppName)
	isSaved := false

	updateEnvs := func(component *godo.AppStaticSiteSpec) (bool, error) {
		envsAsIni := utils.GetDGOEnvsAsIni(component.Envs)

		if filePath == "" {
			if !envsAsIni.Section("").HasKey(envName) {
				fmt.Printf("[WARNING] Environment variable \"%s\" doesn't exists in project \"%s\" in \"%s\" environment\n", envName, project, projEnvironment)
				return false, nil
			}

			envsAsIni.Section("").Key(envName).SetValue(envValue)
			isSaved = true
		} else {
			userEnvsAsIni, err := ini.Load(filePath)
			if err != nil {
				fmt.Println("Error loading file: ", err)
				return false, err
			}

			isSaved, _ = utils.UpdateEnvironmentVariables(envsAsIni, userEnvsAsIni)
		}

		if isSaved {
			component.Envs = utils.GetDGOEnvsFromIni(envsAsIni)
		}

		return isSaved, nil
	}

	err = utils.UpdateDGOApp(client, project, dgoApp, updateEnvs)
	if err != nil {
		fmt.Println("Error updating app: ", err)
		return
	}

	if isSaved {
		fmt.Printf("Environment variables saved in project \"%s\" in \"%s\" environment\n", project, projEnvironment)
	}
}

func UpdateEnvFromFile(client objectstorage.ObjectStorageClient, ociNamespace string, project string, projEnvironment string, envType string, fileName string, filePath string, isK8s bool) {
	userEnvFile, err := ini.Load(filePath)
	if err != nil {
		fmt.Println("Error loading file: ", err)
		return
	}

	envFile, err := utils.GetEnvsFileAsIni(project, fileName, client, ociNamespace, utils.BucketName)

	if err != nil {
		fmt.Println("Error loading file: ", err)
		return
	}

	isSaved, updatedEnvs := utils.UpdateEnvironmentVariables(envFile, userEnvFile)

	if isSaved {
		if isK8s {
			k8sClient, err := utils.GetK8sClient()
			if err != nil {
				log.Fatalf("Error getting Kubernetes client: %v", err)
			}
			manager, resourceName := utils.GetK8sResourceDataParams(k8sClient, project, projEnvironment, envType)

			err = utils.UpdateK8sResourceData(manager, updatedEnvs, resourceName)

			if err != nil {
				log.Fatalf("Failed to update resource data: %v", err)
			}
		}
		utils.SaveEnvFile(client, ociNamespace, project, fileName, envFile, utils.BucketName)
		fmt.Printf("Environment variables saved in project \"%s\" in \"%s\" environment\n", project, projEnvironment)
	}

}

func UpdateSingleEnv(client objectstorage.ObjectStorageClient, ociNamespace string, project string, projEnvironment string, envType string, envName string, envValue string, fileName string, isK8s bool) {
	envFile, err := utils.GetEnvsFileAsIni(project, fileName, client, ociNamespace, utils.BucketName)

	if err != nil {
		fmt.Println("Error loading file: ", err)
		return
	}

	if !envFile.Section("").HasKey(envName) {
		fmt.Printf("[WARNING] Environment variable \"%s\" doesn't exists in project \"%s\" in \"%s\" environment\n", envName, project, projEnvironment)
		return
	}

	if isK8s {
		k8sClient, err := utils.GetK8sClient()
		if err != nil {
			log.Fatalf("Error getting Kubernetes client: %v", err)
		}
		manager, resourceName := utils.GetK8sResourceDataParams(k8sClient, project, projEnvironment, envType)

		envsAsIni := ini.Empty()
		envsAsIni.Section("").Key(envName).SetValue(envValue)

		err = utils.UpdateK8sResourceData(manager, envsAsIni, resourceName)

		if err != nil {
			log.Fatalf("Failed to update resource data: %v", err)
		}
	}

	envFile.Section("").Key(envName).SetValue(envValue)
	utils.SaveEnvFile(client, ociNamespace, project, fileName, envFile, utils.BucketName)
	fmt.Printf("Environment variables saved in project \"%s\" in \"%s\" environment\n", project, projEnvironment)

}

func init() {
	rootCmd.AddCommand(updateCmd)

	validTypesStr := strings.Join(utils.ValidTypes, ", ")
	validProjectsStr := strings.Join(utils.ValidProjects, ", ")
	validProjectEnvsStr := strings.Join(append(utils.ValidEnvs, "all"), ", ")

	updateCmd.Flags().StringP("type", "t", "envs", fmt.Sprintf("Specify the environment variable type (options: %s)", validTypesStr))
	updateCmd.Flags().StringP("project", "p", "", fmt.Sprintf("Specify the project name (options: %s)", validProjectsStr))
	updateCmd.Flags().StringP("environment", "e", "", fmt.Sprintf("Specify the project environment (options: %s)", validProjectEnvsStr))
	updateCmd.Flags().StringP("name", "n", "", "Specify the environment variable or secret name")
	updateCmd.Flags().StringP("value", "v", "", "Specify the environment variable or secret value")
	updateCmd.Flags().StringP("file", "f", "", "Specify a file containing a list of environment variables or secrets. The file should be in INI format.")
	updateCmd.Flags().BoolP("k8s", "k", false, "Update the environment variable or secret from the Kubernetes cluster")

	updateCmd.MarkFlagRequired("project")
	updateCmd.MarkFlagRequired("environment")
}
