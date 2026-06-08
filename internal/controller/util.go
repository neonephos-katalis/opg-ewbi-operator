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

package controller

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	opgmodels "github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func GetFederationByContextId(
	ctx context.Context, c client.Client, fedCtxId string, filterLabels map[string]string,
) (*v1beta1.Federation, error) {
	log := log.FromContext(ctx)

	var federList v1beta1.FederationList

	labelSelector := labels.SelectorFromSet(filterLabels)

	listOpts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(
			v1beta1.FederationStatusContextIDField,
			fedCtxId,
		),
		LabelSelector: labelSelector,
	}

	if err := c.List(ctx, &federList, listOpts); err != nil {
		log.Info("error listing federation objects")
		return nil, err
	}
	if len(federList.Items) != 1 {
		log.Info("unexpected number of federations, should be 1", "actual",
			len(federList.Items))
		return nil, errors.New(
			"unexpected number of federations for this resource, should be 1",
		)
	}
	return &federList.Items[0], nil
}

func handleProblemDetails(log logr.Logger, code int, p *opgmodels.ProblemDetails) {
	log.Info("response with error", "error", code, "details", p)
}

// returns true if LabelValue is v1beta1.FederationRelationGuest
// false otherwise (either label wasn't present or is RelationHost)
func IsGuestResource(labels map[string]string) bool {
	return labels[v1beta1.FederationRelationLabel] == string(v1beta1.FederationRelationGuest)
}

// returns true if LabelValue is v1beta1.FederationTechnologyRest
// false otherwise (either label wasn't present or is another technology)
func IsRestTechnology(labels map[string]string) bool {
	return labels[v1beta1.FederationTechnologyLabel] == string(v1beta1.FederationTechnologyRest)
}

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

// PatchResource any Kubernetes resource, given its group, version, plural name, namespace (empty for cluster-scoped) and name
func PatchResource(
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

func GetResource(
	ctx context.Context,
	dynamicClient dynamic.Interface,
	group, version, resourcePlural string,
	namespace, resourceName string,
) (*unstructured.Unstructured, error) {
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
	// Execute the get operation
	unstructuredRes, err := resourceInterface.Get(ctx, resourceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error during generic get on %s/%s: %w", resourcePlural, resourceName, err)
	}
	return unstructuredRes, nil
}

func DeleteResource(
	ctx context.Context,
	dynamicClient dynamic.Interface,
	group, version, resourcePlural string,
	namespace, resourceName string,
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

	// Execute the delete operation
	err := resourceInterface.Delete(ctx, resourceName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("error during generic delete on %s/%s: %w", resourcePlural, resourceName, err)
	}

	return nil
}
