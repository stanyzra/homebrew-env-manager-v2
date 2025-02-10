/*
Copyright © 2025 Stany Helberth stanyhelberth@gmail.com
*/

package utils

import (
	"context"
	"fmt"

	"gopkg.in/ini.v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

// KubernetesResourceManager is an interface that defines the methods that a Kubernetes resource manager should implement
type KubernetesResourceManager interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (interface{}, error)
	Update(ctx context.Context, obj interface{}, opts metav1.UpdateOptions) error
}

// ConfigMapManager is a struct that holds the client and namespace of a Kubernetes cluster
type ConfigMapManager struct {
	Client    kubernetes.Interface
	Namespace string
}

// Get retrieves a ConfigMap from a Kubernetes cluster
func (c *ConfigMapManager) Get(ctx context.Context, name string, opts metav1.GetOptions) (interface{}, error) {
	return c.Client.CoreV1().ConfigMaps(c.Namespace).Get(ctx, name, opts)
}

// Update updates a ConfigMap in a Kubernetes cluster
func (c *ConfigMapManager) Update(ctx context.Context, obj interface{}, opts metav1.UpdateOptions) error {
	configMap, ok := obj.(*v1.ConfigMap)
	if !ok {
		return fmt.Errorf("invalid object type for ConfigMap update")
	}
	_, err := c.Client.CoreV1().ConfigMaps(c.Namespace).Update(ctx, configMap, opts)
	return err
}

// SecretManager is a struct that holds the client and namespace of a Kubernetes cluster
type SecretManager struct {
	Client    kubernetes.Interface
	Namespace string
}

// Get retrieves a ConfigMap from a Kubernetes cluster
func (c *SecretManager) Get(ctx context.Context, name string, opts metav1.GetOptions) (interface{}, error) {
	return c.Client.CoreV1().Secrets(c.Namespace).Get(ctx, name, opts)
}

// Update updates a Secret in a Kubernetes cluster
func (c *SecretManager) Update(ctx context.Context, obj interface{}, opts metav1.UpdateOptions) error {
	secret, ok := obj.(*v1.Secret)
	if !ok {
		return fmt.Errorf("invalid object type for Secret update")
	}
	_, err := c.Client.CoreV1().Secrets(c.Namespace).Update(ctx, secret, opts)
	return err
}

// UpdateK8sResourceData updates specific key-value pairs inside a Kubernetes ConfigMap or Secret. It can be used to add new keys or update existing ones.
func UpdateK8sResourceData(manager KubernetesResourceManager, envFile *ini.File, resourceName string) error {
	obj, err := manager.Get(context.TODO(), resourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting resource \"%s\": %v", resourceName, err)
	}

	switch resource := obj.(type) {
	case *v1.ConfigMap:
		for _, key := range envFile.Section("").Keys() {
			resource.Data[key.Name()] = key.Value()
		}
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return manager.Update(context.TODO(), resource, metav1.UpdateOptions{})
		})

	case *v1.Secret:
		for _, key := range envFile.Section("").Keys() {
			resource.Data[key.Name()] = []byte(key.Value())
		}
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return manager.Update(context.TODO(), resource, metav1.UpdateOptions{})
		})

	default:
		return fmt.Errorf("unsupported resource type")
	}

	if err != nil {
		return fmt.Errorf("error updating resource \"%s\": %v", resourceName, err)
	}

	return nil
}

// DeleteK8sResourceKey removes specific keys from a Kubernetes ConfigMap or Secret
func DeleteK8sResourceKey(manager KubernetesResourceManager, resourceName string, keys []string) error {
	obj, err := manager.Get(context.TODO(), resourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error getting resource \"%s\": %v", resourceName, err)
	}

	switch resource := obj.(type) {
	case *v1.ConfigMap:
		for _, key := range keys {
			delete(resource.Data, key)
		}
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return manager.Update(context.TODO(), resource, metav1.UpdateOptions{})
		})

	case *v1.Secret:
		for _, key := range keys {
			delete(resource.Data, key)
		}
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return manager.Update(context.TODO(), resource, metav1.UpdateOptions{})
		})

	default:
		return fmt.Errorf("unsupported resource type")
	}

	if err != nil {
		return fmt.Errorf("error deleting keys from resource \"%s\": %v", resourceName, err)
	}

	return nil
}

type ProjectProvider struct {
	Name          string
	CloudProvider []string
}
