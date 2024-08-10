/*
Copyright © 2024 Stany Helberty stanyhelberth@gmail.com
*/

package utils

var ValidTypes = []string{"envs", "secrets"}

var ProjectProviders = []ProjectProvider{
	{Name: "collection-back-end-v2.1", CloudProvider: []string{"OCI"}},
	{Name: "gollection-elastic", CloudProvider: []string{"OCI"}},
	{Name: "app-memorial-collection-v2", CloudProvider: []string{"AWS", "DGO"}},
	{Name: `app-biblioteca-collection-v2`, CloudProvider: []string{"AWS", "DGO"}},
	{Name: "collection-front-end-v2.1", CloudProvider: []string{"AWS", "DGO"}},
	{Name: "app-admin-collection-v2", CloudProvider: []string{"AWS"}},
}

var ValidProjects = []string{"collection-back-end-v2.1", "gollection-elastic", "app-memorial-collection-v2", "app-admin-collection-v2", "collection-front-end-v2.1", "app-biblioteca-collection-v2"}
var ValidEnvs = []string{"prod", "beta", "homolog", "dev"}
var ValidAppComponents = []string{"collection-memorial-white-screen", "service", "collection-home-white-screen"}

const (
	// Bucket name
	BucketName = "collection-kubernetes-files"
)
