/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package k8s

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetKubeconfigFromSecret(ctx context.Context, client client.Client, secretName string, namespace string) ([]byte, error) {
	var secret corev1.Secret
	if err := client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, &secret); err != nil {
		return nil, err
	}
	kubeconfigData, exists := secret.Data["kubeconfig"]
	if !exists {
		return nil, errors.New("Kubeconfig not found in secret")
	}
	return kubeconfigData, nil
}

func BuildClientWithKubeconfig(kubeconfigBytes []byte, contextName string) (dynamic.Interface, error) {
	// Load the configuration from the kubeconfig bytes
	config, err := clientcmd.NewClientConfigFromBytes(kubeconfigBytes)
	if err != nil {
		return nil, err
	}
	if contextName != "" {
		rawConfig, err := config.RawConfig()
		if err != nil {
			return nil, err
		}
		if _, exists := rawConfig.Contexts[contextName]; !exists {
			return nil, errors.New("Context not found in kubeconfig")
		}
		rawConfig.CurrentContext = contextName
		config = clientcmd.NewDefaultClientConfig(rawConfig, &clientcmd.ConfigOverrides{})
	}
	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return dynamicClient, nil
}

func ApplyK8sResource(
	ctx context.Context,
	dynamicClient dynamic.Interface,
	group, version, resourcePlural string,
	namespace string,
	resource *unstructured.Unstructured,
	fieldManager string,
) (*unstructured.Unstructured, error) {
	// Create the GroupVersionResource (GVR) for the target resource
	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resourcePlural,
	}

	var resourceInterface dynamic.ResourceInterface
	// Cluster-scoped or namespace-scoped
	if namespace != "" {
		resourceInterface = dynamicClient.Resource(gvr).Namespace(namespace)
	} else {
		resourceInterface = dynamicClient.Resource(gvr)
	}

	//The Server-Side Apply requires the object to be sent as JSON (Patch)
	data, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("Error during marshal of resource %s: %w", resource.GetName(), err)
	}

	// Set the fieldManager (required for Server-Side Apply) and force the apply
	patchOptions := metav1.PatchOptions{
		FieldManager: fieldManager, // e.g., "my-custom-controller"
		Force:        func(b bool) *bool { return &b }(true),
	}

	// Perform the Patch operation using ApplyPatchType
	appliedRes, err := resourceInterface.Patch(
		ctx,
		resource.GetName(),
		types.ApplyPatchType,
		data,
		patchOptions,
	)
	if err != nil {
		return nil, fmt.Errorf("Error during generic apply on %s/%s: %w", resourcePlural, resource.GetName(), err)
	}

	return appliedRes, nil
}

func GetK8sResource(
	ctx context.Context,
	dynamicClient dynamic.Interface,
	group, version, resourcePlural string,
	namespace, resourceName string,
) (*unstructured.Unstructured, error) {
	// Create the GroupVersionResource (GVR) for the target resource
	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resourcePlural,
	}
	var resourceInterface dynamic.ResourceInterface

	// Namespace-scoped or cluster-scoped resource
	if namespace != "" {
		resourceInterface = dynamicClient.Resource(gvr).Namespace(namespace)
	} else {
		resourceInterface = dynamicClient.Resource(gvr)
	}
	// Execute the get operation
	unstructuredRes, err := resourceInterface.Get(ctx, resourceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Error during generic get on %s/%s: %w", resourcePlural, resourceName, err)
	}
	return unstructuredRes, nil
}

func PatchK8sResource(
	ctx context.Context,
	dynamicClient dynamic.Interface,
	group, version, resourcePlural string,
	namespace, resourceName string,
	patchType types.PatchType,
	patchData []byte,
) error {

	// Creamo the GroupVersionResource for the target resource
	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resourcePlural,
	}

	var resourceInterface dynamic.ResourceInterface

	// Namespace-scoped or cluster-scoped resource
	if namespace != "" {
		resourceInterface = dynamicClient.Resource(gvr).Namespace(namespace)
	} else {
		resourceInterface = dynamicClient.Resource(gvr)
	}

	// Execute the patch operation
	_, err := resourceInterface.Patch(
		ctx,
		resourceName,
		patchType,
		patchData,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("error during generic patch on %s/%s: %w", resourcePlural, resourceName, err)
	}

	return nil
}

func DeleteK8sResource(
	ctx context.Context,
	dynamicClient dynamic.Interface,
	group, version, resourcePlural string,
	namespace, resourceName string,
) error {
	// Create the GroupVersionResource (GVR) for the target resource
	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resourcePlural,
	}
	var resourceInterface dynamic.ResourceInterface

	// Namespace-scoped or cluster-scoped resource
	if namespace != "" {
		resourceInterface = dynamicClient.Resource(gvr).Namespace(namespace)
	} else {
		resourceInterface = dynamicClient.Resource(gvr)
	}

	// Execute the delete operation
	err := resourceInterface.Delete(ctx, resourceName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("Error during generic delete on %s/%s: %w", resourcePlural, resourceName, err)
	}

	return nil
}
