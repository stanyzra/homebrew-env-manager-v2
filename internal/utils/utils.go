/*
Copyright © 2024 Stany Helberty stanyhelberth@gmail.com
*/

package utils

import (
	"fmt"
	"os"

	"github.com/oracle/oci-go-sdk/v49/common"
	"github.com/spf13/cobra"
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

// func ReadObject() {

// }

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
