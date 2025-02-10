/*
Copyright © 2025 Stany Helberth stanyhelberth@gmail.com
*/

package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/amplify"
	"github.com/digitalocean/godo"
	"github.com/oracle/oci-go-sdk/example/helpers"
	"github.com/oracle/oci-go-sdk/v49/common"
	"github.com/oracle/oci-go-sdk/v49/objectstorage"
	"github.com/spf13/cobra"
	"github.com/stanyzra/env-manager-v2/internal/utils"
)

var getCmd = &cobra.Command{
	Use: "get [flags] -p <project-name> -e <project-environment> (<env-name>|--get-all)",
	Example: `env-manager-v2 get -p collection-back-end-v2.1 -e dev -t secrets -A
env-manager-v2 get -p gollection-elastic -e homolog foo
env-manager-v2 get -p collection-back-end-v2.1 -e dev bar moo baz`,
	Short: "Get a list of environment variables and secrets from OCI Object Storage",
	Long: `Get a list of environment variables and secrets (only names) from OCI Object
Storage. Flags can be used to filter the list of environment variables and secrets. The
project and environment flag is required. You can specify one or more environment
variable names in the arguments.`,
	Args: func(cmd *cobra.Command, args []string) error {
		isGetAll, err := cmd.Flags().GetBool("get-all")
		if err != nil {
			log.Fatalf("Error reading option flag: %v", err)
		}

		if isGetAll && len(args) > 0 {
			return fmt.Errorf("cannot use arguments with --get-all flag")
		}

		if !isGetAll && len(args) == 0 {
			return fmt.Errorf("requires an argument unless --get-all is used")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		project, err := utils.GetFlagString(cmd, "project", utils.ValidProjects, false)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		envType, err := utils.GetFlagString(cmd, "type", utils.ValidTypes, false)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		projEnvironment, err := utils.GetFlagString(cmd, "environment", utils.ValidEnvs, false)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		isGetAll, err := cmd.Flags().GetBool("get-all")
		if err != nil {
			log.Fatalf("Error reading option flag: %v", err)
		}

		// provider, err := utils.GetConfigProperty(project, projEnvironment, "provider")
		provider, err := utils.GetConfigProperty("\""+project+"\"", projEnvironment+".provider")

		if err != nil {
			fmt.Println("Error getting provider: ", err)
			return
		}

		switch provider {
		case "OCI":
			configProvider, _, err := utils.GetConfigProviderOCI()
			if err != nil {
				fmt.Println("Error getting config provider: ", err)
				return
			}

			client, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(configProvider)
			helpers.FatalIfError(err)

			ociNamespace, err := utils.GetConfigProperty("OCI", "namespace")

			if err != nil {
				fmt.Println("Error getting namespace: ", err)
				return
			}

			HandleOCI(client, ociNamespace, project, projEnvironment, envType, isGetAll, args)

		case "AWS":
			// projEnvironment, err = utils.GetConfigProperty(project, projEnvironment, "branch_name")
			projEnvironment, err = utils.GetConfigProperty("\""+project+"\"", projEnvironment+".branch_name")

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

			utils.HandleAWS(client, project, projEnvironment, isGetAll, "", args, "", "", false, cmd.Name())

		case "DGO":
			client, err := utils.GetClientDGO()
			if err != nil {
				fmt.Println("Error getting client: ", err)
				return
			}

			showEnvs(client, project, projEnvironment, isGetAll, args)

		default:
			fmt.Println("Invalid provider")
			return
		}
	},
}

func showEnvs(client *godo.Client, project string, projEnvironment string, isGetAll bool, envNames []string) {
	// dgoAppName, err := utils.GetConfigProperty(project, projEnvironment, "app_name")
	dgoAppName, err := utils.GetConfigProperty("\""+project+"\"", projEnvironment+".app_name")
	fmt.Println("dgoAppName: ", dgoAppName)
	if err != nil {
		log.Fatalf("Error getting app name: %v", err)
	}

	specificApp := utils.GetDGOApp(client, dgoAppName)

	printEnvs := func(component *godo.AppStaticSiteSpec) error {
		if isGetAll {
			for _, envVar := range component.Envs {
				fmt.Printf("%s=%s\n", envVar.Key, envVar.Value)
			}
		} else {
			envsAsIni := utils.GetDGOEnvsAsIni(component.Envs)
			for _, envName := range envNames {
				if envsAsIni.Section("").HasKey(envName) {
					value := envsAsIni.Section("").Key(envName).String()
					fmt.Printf("%s=%s\n", envName, value)
				} else {
					fmt.Printf("Environment variable \"%s\" not found in project \"%s\" in \"%s\" environment\n", envName, project, projEnvironment)
				}
			}
		}
		return nil
	}

	// appComponentName, err := utils.GetConfigProperty(project, projEnvironment, "app_component_name")
	appComponentName, err := utils.GetConfigProperty("\""+project+"\"", projEnvironment+".app_component_name")

	if err != nil {
		log.Fatalf("Error getting app component name: %v", err)
	}

	err = godo.ForEachAppSpecComponent(specificApp.Spec, func(component *godo.AppStaticSiteSpec) error {
		if component.Name == appComponentName {
			return printEnvs(component)
		}
		// if utils.StringInSlice(component.Name, utils.ValidAppComponents) {
		// 	return printEnvs(component)
		// }
		return nil
	})
	if err != nil {
		log.Fatalf("Error iterating over app components: %v", err)
	}
}

func GetInputedEnv(client objectstorage.ObjectStorageClient, namespace string, project string, projEnvironment string, envType string, envNames []string) {
	fileName := fmt.Sprintf("%s_%s", projEnvironment, envType)

	envFile, err := utils.GetEnvsFileAsIni(project, fileName, client, namespace, utils.BucketName)

	if err != nil {
		fmt.Println("Error loading file: ", err)
		return
	}

	for _, envName := range envNames {
		value := envFile.Section("").Key(envName).String()
		if value == "" {
			fmt.Printf("Environment variable \"%s\" not found in project \"%s\" in \"%s\" environment\n", envName, project, projEnvironment)
		} else if envType == "secrets" {
			fmt.Printf("%s=***\n", envName)
		} else {
			fmt.Printf("%s=%s\n", envName, value)
		}
	}

}

func HandleOCI(client objectstorage.ObjectStorageClient, namespace, project, projEnvironment, envType string, isGetAll bool, args []string) {
	if isGetAll {
		ReadFullObject(client, namespace, project, projEnvironment, envType)
	} else {
		envNames := args
		GetInputedEnv(client, namespace, project, projEnvironment, envType, envNames)
	}
}

func ReadFullObject(client objectstorage.ObjectStorageClient, namespace string, project string, projEnvironment string, envType string) {
	fileName := fmt.Sprintf("%s_%s", projEnvironment, envType)

	// Get the object
	getRequest := objectstorage.GetObjectRequest{
		NamespaceName: &namespace,
		BucketName:    common.String(utils.BucketName),
		ObjectName:    common.String(fmt.Sprintf("%s/env-files/.%s", project, fileName)),
	}

	getResponse, err := client.GetObject(context.Background(), getRequest)
	helpers.FatalIfError(err)

	// Read the object content
	content, err := io.ReadAll(getResponse.Content)
	if err != nil {
		fmt.Println("Error reading object content: ", err)
		return
	}

	if envType == "envs" {
		fmt.Println(string(content))
	} else {
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				lines[i] = fmt.Sprintf("%s=***\n", parts[0])
			}
		}

		censoredContent := strings.Join(lines, "")
		fmt.Println(censoredContent)
	}
}

func init() {
	rootCmd.AddCommand(getCmd)
	validTypesStr := strings.Join(utils.ValidTypes, ", ")
	validProjectsStr := strings.Join(utils.ValidProjects, ", ")
	validProjectEnvsStr := strings.Join(utils.ValidEnvs, ", ")

	getCmd.Flags().BoolP("get-all", "A", false, "List all environment variables")
	getCmd.Flags().StringP("type", "t", "envs", fmt.Sprintf("Specify the environment variable type (options: %s)", validTypesStr))
	getCmd.Flags().StringP("project", "p", "", fmt.Sprintf("Specify the project name (options: %s)", validProjectsStr))
	getCmd.Flags().StringP("environment", "e", "", fmt.Sprintf("Specify the project environment (options: %s)", validProjectEnvsStr))

	getCmd.MarkFlagRequired("project")
	getCmd.MarkFlagRequired("environment")
}
