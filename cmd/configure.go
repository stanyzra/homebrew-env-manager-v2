/*
Copyright © 2024 Stany Helberty stanyhelberth@gmail.com
*/
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	// Identity or any other service you wish to make requests to

	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

var cloudProviders = []string{"OCI", "AWS", "DGO"}

// configureCmd represents the configure command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure Cloud credentials",
	Long: `Configure Cloud credentials to be used by the CLI. The credentials are stored in
<home-directory>/.env-manager-v2/config file. Accepted cloud providers are: OCI, AWS, and DigitalOcean.`,
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Select a cloud provider: ")
		for i, provider := range cloudProviders {
			fmt.Printf("%d. %s\n", i+1, provider)
		}

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		userChoice, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Error converting input to integer")
		}

		ManageCloudProvider(userChoice - 1)
	},
}

func ManageConfigProperties(questions []string, credentialKeys []string) map[string]string {
	credentials := make(map[string]string)

	i := 0
	reader := bufio.NewReader(os.Stdin)
	for {
		if i == len(questions) {
			break
		}
		fmt.Println(questions[i])
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			fmt.Println("Input cannot be empty")
			continue
		}
		credentials[credentialKeys[i]] = input
		i++
	}

	return credentials

}

func SaveCredentials(configFileName string, credentials map[string]string, provider int) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting user home directory: ", err)
		return
	}
	ini_config := ini.Empty()

	sec, err := ini_config.NewSection(cloudProviders[provider])
	if err != nil {
		fmt.Println("Error creating section: ", err)
		return
	}

	// check if the path exists
	if _, err := os.Stat(configFileName); os.IsNotExist(err) {
		// fmt.Println("File does not exist, creating it in: ", configFileName)
		os.MkdirAll(fmt.Sprintf("%s/%s", userHome, ".env-manager"), os.ModePerm)
	} else {
		// fmt.Println("File exists, loading it in: ", configFileName)
		ini_config, err = ini.Load(configFileName)
		if err != nil {
			fmt.Println("Error loading config file: ", err)
			return
		}

		sec = ini_config.Section(cloudProviders[provider])
	}

	for key, value := range credentials {
		sec.NewKey(key, value)
	}

	err = ini_config.SaveTo(configFileName)
	if err != nil {
		fmt.Println("Error saving credentials: ", err)
	}

	fmt.Println("Credentials saved successfully in: ", configFileName)

}

func ManageCloudProvider(provider int) {
	var credentials map[string]string
	var questions []string
	var credentialKeys []string
	userHome, err := os.UserHomeDir()
	configFileName := fmt.Sprintf("%s/%s", userHome, ".env-manager/config")

	if err != nil {
		fmt.Println("Error getting user home directory: ", err)
		return
	}
	switch provider {
	case 0:
		fmt.Println("Configuring OCI")
		questions = []string{
			"Enter user OCID: ",
			"Enter fingerprint: ",
			"Enter tenancy OCID: ",
			"Enter region: ",
			"Enter private key path: ",
			"Enter OCI OS namespace: ",
		}
		credentialKeys = []string{"user", "fingerprint", "tenancy", "region", "key_file", "namespace"}
	case 1:
		fmt.Println("Configuring AWS")
		questions = []string{
			"Enter Access Key ID: ",
			"Enter Secret Access Key: ",
			"Enter region: ",
		}
		credentialKeys = []string{"aws_access_key_id", "aws_secret_access_key", "region"}
	case 2:
		fmt.Println("Configuring DigitalOcean")
		questions = []string{
			"Enter DGO API Key: ",
		}
		credentialKeys = []string{"dgo_api_token"}
	default:
		fmt.Println("Invalid choice")
	}

	credentials = ManageConfigProperties(questions, credentialKeys)
	SaveCredentials(configFileName, credentials, provider)
}

func init() {
	rootCmd.AddCommand(configureCmd)
}
