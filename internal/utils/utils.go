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
	"os"
	"strings"

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

// GetFlagString reads and validates a string flag
func GetConfigProviderOCI() (common.ConfigurationProvider, string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting user home directory: ", err)
		return nil, "", err
	}
	configFileName := fmt.Sprintf("%s/%s", userHome, ".env-manager/config")

	return common.CustomProfileConfigProvider(fmt.Sprintf("%s/.env-manager/config", userHome), "OCI"), configFileName, nil
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
