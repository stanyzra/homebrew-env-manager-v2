/*
Copyright © 2024 Stany Helberty stanyhelberth@gmail.com
*/

package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/oracle/oci-go-sdk/example/helpers"
	"github.com/oracle/oci-go-sdk/v49/common"
	"github.com/oracle/oci-go-sdk/v49/objectstorage"
	"github.com/spf13/cobra"
	"github.com/stanyzra/env-manager-v2/internal/utils"
	"gopkg.in/ini.v1"
)

// Define the valid options as a slice
var validTypes = []string{"envs", "secrets"}
var validProjects = []string{"collection-back-end-v2.1", "gollection-elastic"}
var validEnvs = []string{"prod", "beta", "homolog", "dev"}

const (
	// Bucket name
	bucketName = "collection-kubernetes-files"
)

var getCmd = &cobra.Command{
	Use: "get [flags] -p <project-name> -e <project-environment> (<env-name>|--get-all)",
	Example: `env-manager-v2 get -p collection-back-end-v2.1 -e dev -t secrets -A
env-manager-v2 get -p gollection-elastic -e homolog foo
env-manager-v2 get -p collection-back-end-v2.1 -e dev bar moo baz`,
	Short: "Get a list of environment variables and secrets from OCI Object Storage",
	Long: `Get a list of environment variables and secrets (only names) from OCI Object
Storage. Flags can be used to filter the list of environment variables and secrets. The
project flag is required. You can specify one or more environment variable names in the arguments.`,
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

		isGetAll, err := cmd.Flags().GetBool("get-all")
		if err != nil {
			log.Fatalf("Error reading option flag: %v", err)
		}

		configProvider, configFileName, err := utils.GetConfigProviderOCI()

		if err != nil {
			fmt.Println("Error getting config provider: ", err)
			return
		}

		client, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(configProvider)
		helpers.FatalIfError(err)

		ini_config, err := ini.Load(configFileName)
		if err != nil {
			fmt.Println("Error loading config file: ", err)
			return
		}

		sec := ini_config.Section("OCI")
		namespace := sec.Key("namespace").String()

		if isGetAll {
			ReadFullObject(client, namespace, project, projEnvironment, envType)
			return
		} else {
			envNames := args
			GetInputedEnv(client, namespace, project, projEnvironment, envType, envNames)
		}

	},
}

func GetInputedEnv(client objectstorage.ObjectStorageClient, namespace string, project string, projEnvironment string, envType string, envNames []string) {
	fileName := fmt.Sprintf("%s_%s", projEnvironment, envType)

	// Get the object
	getRequest := objectstorage.GetObjectRequest{
		NamespaceName: &namespace,
		BucketName:    common.String(bucketName),
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

	envFile, err := ini.Load(content)

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

func ReadFullObject(client objectstorage.ObjectStorageClient, namespace string, project string, projEnvironment string, envType string) {
	fileName := fmt.Sprintf("%s_%s", projEnvironment, envType)

	// Get the object
	getRequest := objectstorage.GetObjectRequest{
		NamespaceName: &namespace,
		BucketName:    common.String(bucketName),
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
	validTypesStr := strings.Join(validTypes, ", ")
	validProjectsStr := strings.Join(validProjects, ", ")
	validProjectEnvsStr := strings.Join(validEnvs, ", ")

	getCmd.Flags().BoolP("get-all", "A", false, "List all environment variables")
	getCmd.Flags().StringP("type", "t", "envs", fmt.Sprintf("Specify the environment variable type (options: %s)", validTypesStr))
	getCmd.Flags().StringP("project", "p", "", fmt.Sprintf("Specify the project name (options: %s)", validProjectsStr))
	getCmd.Flags().StringP("environment", "e", "", fmt.Sprintf("Specify the project environment (options: %s)", validProjectEnvsStr))

	getCmd.MarkFlagRequired("project")
	getCmd.MarkFlagRequired("environment")
}
