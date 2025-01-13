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
	"github.com/digitalocean/godo"

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
		if strings.TrimSpace(line) != "" {
			// Remove extra spaces around '='
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				result = append(result, strings.TrimSpace(parts[0])+"="+strings.TrimSpace(parts[1]))
			}
		}
	}
	finalString := strings.Join(result, "\n")

	return finalString, nil
}

// GetEnvsFileAsIni reads an environment file from OCI Object Storage and returns it as an ini.File
func GetEnvsFileAsIni(project string, fileName string, client objectstorage.ObjectStorageClient, namespace string, BucketName string) (*ini.File, error) {
	// Get the object
	getRequest := objectstorage.GetObjectRequest{
		NamespaceName: &namespace,
		BucketName:    common.String(BucketName),
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
func GetFlagString(cmd *cobra.Command, name string, validOptions []string, isGetAllAvailable bool) (string, error) {
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		return "", fmt.Errorf("error reading --%s flag: %w", name, err)
	}

	if isGetAllAvailable && value == "all" {
		return value, nil
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
func GetCloudProvider(project string, ProjectProviders []ProjectProvider) []string {
	for _, provider := range ProjectProviders {
		if provider.Name == project {
			return provider.CloudProvider
		}
	}
	return nil
}

// Cast the project name to a valid name for DGO AppPlatform
func CastDGOProjectName(projectName string) string {
	switch projectName {
	case "app-memorial-collection-v2":
		return "collection-memorial-app"
	case "collection-front-end-v2.1":
		return "collection-home-app"
	case "app-biblioteca-collection-v2":
		return "collection-library-app"
	default:
		return projectName
	}
}

// Cast the project environment to a valid branch name for AWS Amplify
func CastBranchName(branchName string, projectName string) string {
	switch branchName {
	case "dev":
		return "development"
	case "prod":
		if projectName == "app-admin-collection-v2" {
			return "master"
		} else {
			return "main"
		}
	default:
		return branchName
	}
}

// SaveEnvFile cast a ini.File to a string and saves it in OCI Object Storage
func SaveEnvFile(client objectstorage.ObjectStorageClient, namespace string, project string, fileName string, envFile *ini.File, BucketName string) {
	envFileContent, err := IniToString(envFile)
	if err != nil {
		fmt.Println("Error converting file to string: ", err)
		return
	}

	// Save file
	saveRequest := objectstorage.PutObjectRequest{
		NamespaceName: &namespace,
		BucketName:    common.String(BucketName),
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
func HandleAWS(client *amplify.Client, project, projEnvironment string, isGetAll bool, filePath string, args []string, envName string, envValue string, isQuiet bool, command string) {
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
	case "delete":
		DeleteAWSEnvs(branchInfos, client, project, projEnvironment, filePath, args, isQuiet, appId)
	case "update":
		UpdateAWSEnvs(branchInfos, client, project, projEnvironment, filePath, envName, envValue, appId)
	default:
		fmt.Println("Invalid command")
	}
}

// GetClientDGO returns a client for DGO
func GetClientDGO() (*godo.Client, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting user home directory: ", err)
		return nil, err
	}

	configFileName := fmt.Sprintf("%s/%s", userHome, ".env-manager/config")

	configFile, err := ini.Load(configFileName)
	if err != nil {
		fmt.Println("Error loading config file: ", err)
		return nil, err

	}

	dgoConfig := configFile.Section("DGO")
	dgoToken := dgoConfig.Key("dgo_api_token").String()
	client := godo.NewFromToken(dgoToken)

	return client, nil
}

// UpdateAWSEnvs updates environment variables in a AWS Amplify app
func UpdateAWSEnvs(branchInfos *amplify.GetBranchOutput, client *amplify.Client, project string, projEnvironment string, filePath string, envName string, envValue string, appId string) {
	iniAWS := ini.Empty()
	isSaved := false

	for envName, envValue := range branchInfos.Branch.EnvironmentVariables {
		iniAWS.Section("").Key(envName).SetValue(envValue)
	}

	if filePath != "" {
		userEnvFile, err := ini.Load(filePath)
		if err != nil {
			fmt.Println("Error loading file: ", err)
			return
		}

		isSaved = UpdateEnvironmentVariables(iniAWS, userEnvFile)

	} else {
		if !iniAWS.Section("").HasKey(envName) {
			fmt.Printf("[WARNING] Environment variable \"%s\" not found in project \"%s\" in \"%s\" environment\n", envName, project, projEnvironment)
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

		fmt.Printf("Environment variables updated in project \"%s\" in \"%s\" environment\n", project, projEnvironment)
	}
}

// DeleteAWSEnvs deletes environment variables in a AWS Amplify app
func DeleteAWSEnvs(branchInfos *amplify.GetBranchOutput, client *amplify.Client, project string, projEnvironment string, filePath string, envNames []string, isQuiet bool, appId string) {
	iniAWS := ini.Empty()

	for envName, envValue := range branchInfos.Branch.EnvironmentVariables {
		iniAWS.Section("").Key(envName).SetValue(envValue)
	}

	if filePath != "" {
		userEnvFile, err := ini.Load(filePath)
		if err != nil {
			fmt.Println("Error loading file: ", err)
			if _, err := os.Stat(filePath); err == nil {
				fmt.Println("Are you sure the file are in INI format (<key>=<value>)?")
			}
			return
		}
		envNames = userEnvFile.Section("").KeyStrings()
		fmt.Printf("Deleting from file: %s\n", filePath)
	}

	if DeleteEnvironmentVariables(iniAWS, envNames) {
		if isQuiet || GetUserPermission("Are you sure you want to delete the environment variables?") {
			_, err := client.UpdateBranch(context.Background(), &amplify.UpdateBranchInput{
				AppId:                common.String(appId),
				BranchName:           branchInfos.Branch.BranchName,
				EnvironmentVariables: iniAWS.Section("").KeysHash(),
			})

			if err != nil {
				fmt.Println("Error updating branch: ", err)
				return
			}
			fmt.Printf("Environment variables deleted in project \"%s\" in \"%s\" environment\n", project, projEnvironment)
		}
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

// CreateAWSEnvs creates environment variables in a AWS Amplify app
func CreateAWSEnvs(branchInfos *amplify.GetBranchOutput, client *amplify.Client, project string, projEnvironment string, filePath string, envName string, envValue string, appId string) {
	iniAWS := ini.Empty()
	isSaved := false

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

// GetDGOApp gets a DGO App by its project name
func GetDGOApp(client *godo.Client, project string) *godo.App {
	castedProject := CastDGOProjectName(project)
	ctx := context.TODO()

	apps, _, err := client.Apps.List(ctx, nil)
	if err != nil {
		log.Fatalf("Error getting apps: %v", err)
	}

	var specificApp *godo.App
	for _, app := range apps {
		if app.Spec.Name == castedProject {
			specificApp, _, err = client.Apps.Get(ctx, app.ID)
			if err != nil {
				log.Fatalf("Error getting app: %v", err)
			}
			break
		}
	}

	if specificApp == nil {
		log.Fatalf("App with project name %s not found", project)
	}

	return specificApp
}

// GetDGOEnvsAsIni converts a slice of AppVariableDefinition to an ini.File
func GetDGOEnvsAsIni(appEnvs []*godo.AppVariableDefinition) *ini.File {
	envsAsIni := ini.Empty()

	for _, envVar := range appEnvs {
		envsAsIni.Section("").NewKey(envVar.Key, envVar.Value)
	}

	return envsAsIni
}

// UpdateDGOApp updates environment variables in a DGO App
func UpdateDGOApp(client *godo.Client, project string, dgoApp *godo.App, updateFunc func(*godo.AppStaticSiteSpec) (bool, error)) error {
	isSaved := false
	var err error
	err = godo.ForEachAppSpecComponent(dgoApp.Spec, func(component *godo.AppStaticSiteSpec) error {
		if StringInSlice(component.Name, ValidAppComponents) {
			isSaved, err = updateFunc(component)
			if err != nil {
				return err
			}
			if isSaved {
				return nil
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error updating app components: %w", err)
	}

	if isSaved {
		_, _, err = client.Apps.Update(context.TODO(), dgoApp.ID, &godo.AppUpdateRequest{
			Spec: dgoApp.Spec,
		})
	}

	if err != nil {
		return fmt.Errorf("error updating app: %w", err)
	}

	return nil
}

// DeleteFromFile deletes environment variables in a ini.File
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

// GetDGOEnvsFromIni converts an ini.File to a slice of AppVariableDefinition
func GetDGOEnvsFromIni(envsAsIni *ini.File) []*godo.AppVariableDefinition {
	var appEnvs []*godo.AppVariableDefinition

	for _, key := range envsAsIni.Section("").Keys() {
		appEnvs = append(appEnvs, &godo.AppVariableDefinition{
			Key:   key.Name(),
			Value: key.Value(),
		})
	}

	return appEnvs
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

// UpdateEnvironmentVariables updates environment variables in a ini.File
func UpdateEnvironmentVariables(envFile *ini.File, userEnvsFile *ini.File) bool {
	isSaved := false
	for _, key := range userEnvsFile.Section("").Keys() {
		if !envFile.Section("").HasKey(key.Name()) {
			fmt.Printf("[WARNING] Key \"%s\" not found in environment file\n", key.Name())
			continue
		}

		isSaved = true
		envFile.Section("").Key(key.Name()).SetValue(key.Value())
	}

	return isSaved
}
