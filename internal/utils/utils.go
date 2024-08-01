/*
Copyright © 2024 Stany Helberty stanyhelberth@gmail.com
*/

package utils

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/amplify"

	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/oracle/oci-go-sdk/example/helpers"
	"github.com/oracle/oci-go-sdk/v49/common"
	"github.com/oracle/oci-go-sdk/v49/objectstorage"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

// StringInSlice checks if a string is in a slice of strings
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// IniToString converts an ini.File to a string
func IniToString(iniFile *ini.File) (string, error) {
	var buffer bytes.Buffer
	_, err := iniFile.WriteTo(&buffer)
	if err != nil {
		fmt.Println("Error writing to buffer: ", err)
		return "", err
	}

	iniString := buffer.String()

	var result []string
	for _, line := range strings.Split(iniString, "\n") {
		// Removendo espaços em branco ao redor dos '='
		result = append(result, strings.ReplaceAll(line, " ", ""))
	}
	finalString := strings.Join(result, "\n")

	return finalString, nil

}

// GetEnvsFileAsIni reads an environment file from OCI Object Storage and returns it as an ini.File
func GetEnvsFileAsIni(project string, fileName string, client objectstorage.ObjectStorageClient, namespace string, bucketName string) (*ini.File, error) {
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
		return nil, err
	}

	envFile, err := ini.Load(content)

	if err != nil {
		fmt.Println("Error loading file: ", err)
		return nil, err
	}

	return envFile, nil
}

// GetFlagString reads and validates a string flag
func GetFlagString(cmd *cobra.Command, name string, validOptions []string) (string, error) {
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		return "", fmt.Errorf("error reading --%s flag: %w", name, err)
	}

	if !StringInSlice(value, validOptions) {
		return "", fmt.Errorf("invalid %s \"%s\". Options are: %v", name, value, validOptions)
	}

	return value, nil
}

// GetConfigProviderOCI returns a ConfigurationProvider for OCI
func GetConfigProviderOCI() (common.ConfigurationProvider, string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting user home directory: ", err)
		return nil, "", err
	}
	configFileName := fmt.Sprintf("%s/%s", userHome, ".env-manager/config")

	return common.CustomProfileConfigProvider(fmt.Sprintf("%s/.env-manager/config", userHome), "OCI"), configFileName, nil
}

// GetConfigProviderAWS returns a ConfigurationProvider for AWS
func GetConfigProviderAWS() (aws.Config, string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting user home directory: ", err)
		return aws.Config{}, "", err
	}

	configFileName := fmt.Sprintf("%s/%s", userHome, ".env-manager/config")

	configFile, err := ini.Load(configFileName)
	if err != nil {
		fmt.Println("Error loading config file: ", err)
		return aws.Config{}, "", err
	}

	awsConfig := configFile.Section("AWS")

	awsAccessKeyID := awsConfig.Key("aws_access_key_id").String()
	awsSecretAccessKey := awsConfig.Key("aws_secret_access_key").String()
	awsRegion := awsConfig.Key("region").String()

	awsCreds := credentials.NewStaticCredentialsProvider(awsAccessKeyID, awsSecretAccessKey, "")

	configProvider, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(awsRegion),
		config.WithCredentialsProvider(awsCreds))

	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	return configProvider, configFileName, nil
}

// GetCloudProvider returns the cloud provider for a given project
func GetCloudProvider(project string, projectProviders []ProjectProvider) []string {
	for _, provider := range projectProviders {
		if provider.Name == project {
			return provider.CloudProvider
		}
	}
	return nil
}

// Cast the project environment to a valid branch name for AWS Amplify
func CastBranchName(branchName string) string {
	switch branchName {
	case "dev":
		return "development"
	case "prod":
		return "production"
	default:
		return branchName
	}
}

// SaveEnvFile cast a ini.File to a string and saves it in OCI Object Storage
func SaveEnvFile(client objectstorage.ObjectStorageClient, namespace string, project string, fileName string, envFile *ini.File, bucketName string) {
	envFileContent, err := IniToString(envFile)
	if err != nil {
		fmt.Println("Error converting file to string: ", err)
		return
	}

	// Save file
	saveRequest := objectstorage.PutObjectRequest{
		NamespaceName: &namespace,
		BucketName:    common.String(bucketName),
		ObjectName:    common.String(fmt.Sprintf("%s/env-files/.%s", project, fileName)),
		PutObjectBody: io.NopCloser(strings.NewReader(envFileContent)),
	}

	_, err = client.PutObject(context.Background(), saveRequest)
	if err != nil {
		fmt.Println("Error saving file: ", err)
		return
	}
}

// GetUserPermission asks the user for permission to proceed
func GetUserPermission(message string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s (y/n): ", message)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input, please try again.")
			continue
		}

		response = strings.TrimSpace(response)

		switch response {
		case "y":
			return true
		case "n":
			return false
		default:
			fmt.Println("Invalid response. Please type 'y' or 'n'.")
		}
	}
}

// HandleAWS handles the AWS Amplify environment variables and controlls the command function
func HandleAWS(client *amplify.Client, project, projEnvironment string, isGetAll bool, filePath string, args []string, envName string, envValue string, command string) {
	apps, err := client.ListApps(context.Background(), &amplify.ListAppsInput{})
	var appId string
	if err != nil {
		fmt.Println("Error getting apps: ", err)
		return
	}

	var branchInfos *amplify.GetBranchOutput

	for _, app := range apps.Apps {
		if *app.Name == project {
			branchInfos, err = client.GetBranch(context.Background(), &amplify.GetBranchInput{
				AppId:      common.String(*app.AppId),
				BranchName: common.String(projEnvironment),
			})
			appId = *app.AppId
			if err != nil {
				fmt.Printf("Error getting app in branch \"%s\": %s", projEnvironment, err)
				return
			}
		}
	}

	switch command {
	case "create":
		CreateAWSEnvs(branchInfos, client, project, projEnvironment, filePath, envName, envValue, appId)
	case "get":
		PrintAWSEnvs(branchInfos, project, projEnvironment, isGetAll, args)
	default:
		fmt.Println("Invalid command")
	}

}

// PrintAWSEnvs reads the environment variables from AWS Amplify app
func PrintAWSEnvs(branchInfos *amplify.GetBranchOutput, project string, projEnvironment string, isGetAll bool, args []string) {
	if isGetAll {
		for envName, envValue := range branchInfos.Branch.EnvironmentVariables {
			fmt.Printf("%s=%s\n", envName, envValue)
		}
	} else {
		envNames := args
		for _, envName := range envNames {
			envValue, ok := branchInfos.Branch.EnvironmentVariables[envName]
			if !ok {
				fmt.Printf("Environment variable \"%s\" not found in project \"%s\" in \"%s\" environment\n", envName, project, projEnvironment)
			} else {
				fmt.Printf("%s=%s\n", envName, envValue)
			}
		}
	}
}

// CreateAWSEnvs creates environment variables in AWS Amplify app
func CreateAWSEnvs(branchInfos *amplify.GetBranchOutput, client *amplify.Client, project string, projEnvironment string, filePath string, envName string, envValue string, appId string) {
	iniAWS := ini.Empty()
	var isSaved bool

	for envName, envValue := range branchInfos.Branch.EnvironmentVariables {
		iniAWS.Section("").Key(envName).SetValue(envValue)
	}

	if filePath != "" {
		userEnvFile, err := ini.Load(filePath)
		if err != nil {
			fmt.Println("Error loading file: ", err)
			return
		}

		isSaved = CreateEnvironmentVariables(iniAWS, userEnvFile)

	} else {
		if iniAWS.Section("").HasKey(envName) {
			fmt.Printf("[WARNING] Environment variable \"%s\" already exists in project \"%s\" in \"%s\" environment\n", envName, project, projEnvironment)
			return
		}

		iniAWS.Section("").Key(envName).SetValue(envValue)
		isSaved = true
	}

	if isSaved {
		_, err := client.UpdateBranch(context.Background(), &amplify.UpdateBranchInput{
			AppId:                common.String(appId),
			BranchName:           branchInfos.Branch.BranchName,
			EnvironmentVariables: iniAWS.Section("").KeysHash(),
		})

		if err != nil {
			fmt.Println("Error updating branch: ", err)
			return
		}

		fmt.Printf("Environment variables saved in project \"%s\" in \"%s\" environment\n", project, projEnvironment)
	}
}

// CreateEnvironmentVariables creates environment variables in a ini.File
func CreateEnvironmentVariables(envFile *ini.File, userEnvsFile *ini.File) bool {
	isSaved := false
	for _, key := range userEnvsFile.Section("").Keys() {
		if envFile.Section("").HasKey(key.Name()) {
			fmt.Printf("[WARNING] Environment variable \"%s\" already exists\n", key.Name())
			continue
		}

		isSaved = true
		envFile.Section("").Key(key.Name()).SetValue(key.Value())
	}

	return isSaved
}
