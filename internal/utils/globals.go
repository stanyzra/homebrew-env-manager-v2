/*
Copyright © 2025 Stany Helberth stanyhelberth@gmail.com
*/

package utils

import (
	"log"
	"slices"
	"strings"

	"gopkg.in/ini.v1"
)

func GetProjects() []string {
	configFileName := GetConfigFileName()

	cfg, err := ini.Load(configFileName)
	if err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	projects := strings.Split(strings.ReplaceAll(cfg.Section("PROJECTS").Key("projects").Value(), " ", ""), ",")
	return projects
}

func GetEnvironments() []string {
	configFileName := GetConfigFileName()

	cfg, err := ini.Load(configFileName)
	if err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	environemnts := strings.Split(strings.ReplaceAll(cfg.Section("ENVIRONMENTS").Key("environments").Value(), " ", ""), ",")
	return environemnts
}

func GetProjectProviders(projects []string, environments []string) []ProjectProvider {
	configFileName := GetConfigFileName()

	cfg, err := ini.Load(configFileName)
	if err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	var projectProviders []ProjectProvider
	for _, project := range projects {
		var providers []string
		for _, environment := range environments {
			provider := cfg.Section("\"" + project + "\"").Key(environment + ".provider").Value()
			providers = append(providers, provider)
		}

		slices.Sort(providers)
		providers = slices.Compact(providers)

		projectProviders = append(projectProviders, ProjectProvider{Name: project, CloudProvider: providers})
	}
	return projectProviders
}

func GetAppComponents() []string {
	configFileName := GetConfigFileName()

	cfg, err := ini.Load(configFileName)
	if err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	appComponents := strings.Split(strings.ReplaceAll(cfg.Section("DGO.APP_COMPONENTS").Key("app_components").Value(), " ", ""), ",")
	return appComponents
}

func GetBucketName() string {
	configFileName := GetConfigFileName()

	cfg, err := ini.Load(configFileName)
	if err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	return cfg.Section("OCI").Key("bucket_name").Value()
}

var ValidTypes = []string{"envs", "secrets"}

var ValidProjects = GetProjects()
var ValidEnvs = GetEnvironments()
var ValidAppComponents = GetAppComponents()
var ProjectProviders = GetProjectProviders(ValidProjects, ValidEnvs)
var BucketName = GetBucketName()
