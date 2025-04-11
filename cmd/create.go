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

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use: "create [flags] -p <project-name> -e <project-environment> (-n <name> -v <value>|--file <file>)",
	Example: `env-manager-v2 create -p collection-back-end-v2.1 -e dev -t envs -n foo -v bar
env-manager-v2 create -p gollection-elastic -e homolog -t secrets -n moo -v baz
env-manager-v2 create -p collection-back-end-v2.1 -e dev -t envs -f /path/to/file`,
	Short: "Create a new environment variable or secret for a project",
	Long: `Create a new environment variable or secret for a configured project. The project and
	environment flags are required. If the file flag is used, the name and value flags are ignored.
	If a environment variable or secret with the same name already exists, it will not be created.
	Use the update command to update an existing environment variable or secret.`,
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

		environments, err := utils.GetConfigProperty(project, "environments")
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		utils.ValidEnvs = strings.Split(strings.ReplaceAll(environments, " ", ""), ",")

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

		projEnvironmentList := []string{projEnvironment}

		if projEnvironment == "all" {
			projEnvironmentList = utils.ValidEnvs
		}

		for _, projEnv := range projEnvironmentList {
			provider, err := utils.GetConfigProperty(project, projEnv+".provider")
			if err != nil {
				fmt.Println("Error getting provider: ", err)
				return
			}

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
					CreateEnvFromFile(client, ociNamespace, project, projEnv, envType, fileName, filePath, isK8s)
				} else {
					CreateSingleEnv(client, ociNamespace, project, projEnv, envType, envName, envValue, fileName, isK8s)
				}
			case "DGO":
				client, err := utils.GetClientDGO()
				if err != nil {
					fmt.Println("Error getting client: ", err)
					return
				}
				CreateDGOEnv(client, project, projEnv, filePath, envName, envValue)

			case "AWS":
				projEnv, err = utils.GetConfigProperty(project, projEnv+".branch_name")

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

			default:
				fmt.Println("Invalid provider")
				return
			}
		}
	},
}

func CreateDGOEnv(client *godo.Client, project string, projEnvironment string, filePath string, envName string, envValue string) {
	dgoAppName, err := utils.GetConfigProperty(project, projEnvironment+".app_name")

	if err != nil {
		fmt.Println("Error getting app name: ", err)
		return
	}
	dgoApp := utils.GetDGOApp(client, dgoAppName)
	isSaved := false

	createEnvs := func(component *godo.AppStaticSiteSpec) (bool, error) {
		envsAsIni := utils.GetDGOEnvsAsIni(component.Envs)

		if filePath == "" {
			if envsAsIni.Section("").HasKey(envName) {
				fmt.Printf("[WARNING] Environment variable \"%s\" already exists in project \"%s\" in \"%s\" environment\n", envName, project, projEnvironment)
				return false, nil
			}

			envsAsIni.Section("").Key(envName).SetValue(envValue)
			isSaved = true
		} else {
			userEnvFile, err := ini.Load(filePath)
			if err != nil {
				fmt.Println("Error loading file: ", err)
				return false, err
			}

			isSaved, _ = utils.CreateEnvironmentVariables(envsAsIni, userEnvFile)
		}

		if isSaved {
			component.Envs = utils.GetDGOEnvsFromIni(envsAsIni)
		}

		return isSaved, nil
	}

	err = utils.UpdateDGOApp(client, project, dgoApp, createEnvs)
	if err != nil {
		fmt.Println("Error updating app: ", err)
		return
	}

	if isSaved {
		fmt.Printf("Environment variables saved in project \"%s\" in \"%s\" environment\n", project, projEnvironment)
	}
}

func CreateEnvFromFile(client objectstorage.ObjectStorageClient, ociNamespace string, project string, projEnvironment string, envType string, fileName string, filePath string, isK8s bool) {
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

	isSaved, createdEnvs := utils.CreateEnvironmentVariables(envFile, userEnvFile)

	if isSaved {
		if isK8s {
			k8sClient, err := utils.GetK8sClient()
			if err != nil {
				log.Fatalf("Error getting Kubernetes client: %v", err)
			}
			manager, resourceName := utils.GetK8sResourceDataParams(k8sClient, project, projEnvironment, envType)

			err = utils.UpdateK8sResourceData(manager, createdEnvs, resourceName)

			if err != nil {
				log.Fatalf("Failed to update resource data: %v", err)
			}
		}
		utils.SaveEnvFile(client, ociNamespace, project, fileName, envFile, utils.BucketName)
		fmt.Printf("Environment variables saved in project \"%s\" in \"%s\" environment\n", project, projEnvironment)
	}
}

func CreateSingleEnv(client objectstorage.ObjectStorageClient, ociNamespace string, project string, projEnvironment string, envType string, envName string, envValue string, fileName string, isK8s bool) {
	envFile, err := utils.GetEnvsFileAsIni(project, fileName, client, ociNamespace, utils.BucketName)

	if err != nil {
		fmt.Println("Error loading file: ", err)
		return
	}

	if envFile.Section("").HasKey(envName) {
		fmt.Printf("[WARNING] Environment variable \"%s\" already exists in project \"%s\" in \"%s\" environment\n", envName, project, projEnvironment)
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
	rootCmd.AddCommand(createCmd)

	createCmd.Flags().StringP("type", "t", "envs", "Specify the environment variable type")
	createCmd.Flags().StringP("project", "p", "", "Specify the project name")
	createCmd.Flags().StringP("environment", "e", "", "Specify the project environment")
	createCmd.Flags().StringP("name", "n", "", "Specify the environment variable or secret name")
	createCmd.Flags().StringP("value", "v", "", "Specify the environment variable or secret value")
	createCmd.Flags().StringP("file", "f", "", "Specify a file containing a list of environment variables or secrets. The file should be in INI format.")
	createCmd.Flags().BoolP("k8s", "k", false, "Create the environment variable or secret in the Kubernetes cluster")

	createCmd.MarkFlagRequired("project")
	createCmd.MarkFlagRequired("environment")

	createCmd.RegisterFlagCompletionFunc("project", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		projects := []cobra.Completion{}
		projects = append(projects, utils.ValidProjects...)
		return projects, cobra.ShellCompDirectiveDefault
	})

	createCmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		types := []cobra.Completion{}
		types = append(types, utils.ValidTypes...)
		return types, cobra.ShellCompDirectiveDefault
	})

	createCmd.RegisterFlagCompletionFunc("environment", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		project, err := cmd.Flags().GetString("project")
		if err != nil {
			return nil, cobra.ShellCompDirectiveDefault
		}
		envs, err := utils.GetConfigProperty(project, "environments")
		if err != nil {
			return nil, cobra.ShellCompDirectiveDefault
		}

		validEnvs := []cobra.Completion{}
		validEnvs = append(validEnvs, strings.Split(envs, ",")...)
		return validEnvs, cobra.ShellCompDirectiveDefault
	})

	createCmd.RegisterFlagCompletionFunc("name", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	})

	createCmd.RegisterFlagCompletionFunc("value", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	})

	createCmd.RegisterFlagCompletionFunc("file", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveDefault
	})

	createCmd.RegisterFlagCompletionFunc("k8s", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	})
}
