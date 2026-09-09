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
	"errors"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CompareSameAZs compares two slices of ZoneDetails and returns true if they contain the same elements, regardless of order.
func CompareSameAZs(s1, s2 []v1beta1.ZoneDetails) bool {
	if len(s1) != len(s2) {
		return false
	}
	set := make(map[string]bool)
	for _, v := range s1 {
		set[v.ZoneId] = true
	}
	for _, v := range s2 {
		if !set[v.ZoneId] {
			return false
		}
	}
	return true
}

// Build a Kubernetes client and dynamic client for the host cluster using the kubeconfig stored in the secret referenced by the Federation resource.
func buildHostClient(ctx context.Context, fed *v1beta1.Federation, r client.Client, scheme *runtime.Scheme) (client.Client, dynamic.Interface, error) {
	kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r, fed.Spec.FederationData.K8sOptions.SecretName, fed.Namespace)
	if err != nil {
		return nil, nil, err
	}
	return BuildClientWithKubeconfig(kubeconfigBytes, fed.Spec.FederationData.K8sOptions.ContextName, scheme)
}

// GetKubeconfigFromSecret retrieves the kubeconfig from the specified secret in the given namespace.
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

// BuildClientWithKubeconfig builds a Kubernetes client and dynamic client from the provided kubeconfig bytes and context name.
func BuildClientWithKubeconfig(kubeconfigBytes []byte, contextName string, scheme *runtime.Scheme) (client.Client, dynamic.Interface, error) {
	// Load the configuration from the kubeconfig bytes
	config, err := clientcmd.NewClientConfigFromBytes(kubeconfigBytes)
	if err != nil {
		return nil, nil, err
	}
	if contextName != "" {
		rawConfig, err := config.RawConfig()
		if err != nil {
			return nil, nil, err
		}
		if _, exists := rawConfig.Contexts[contextName]; !exists {
			return nil, nil, errors.New("Context not found in kubeconfig")
		}
		rawConfig.CurrentContext = contextName
		config = clientcmd.NewDefaultClientConfig(rawConfig, &clientcmd.ConfigOverrides{})
	}
	restConfig, err := config.ClientConfig()
	if err != nil {
		return nil, nil, err
	}
	k8sClient, err := client.New(restConfig, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, nil, err
	}

	// Create a dynamic client for unstructured resources
	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}

	return k8sClient, dynClient, nil
}

// ApplyRemoteResource applies or updates a remote Kubernetes resource in the host cluster based on the provided object, and starts a watcher if it's a new resource.
func ApplyRemoteResource(
	ctx context.Context,
	localClient client.Client,
	scheme *runtime.Scheme,
	fed *v1beta1.Federation,
	remoteObj client.Object,
	emptyCheckObj client.Object,
	localName string,
	localNamespace string,
	group string,
	version string,
	plural string,
	fieldOwner string,
	logPrefix string,
) error {
	log := ctrl.Log
	remoteNamespace := remoteObj.GetNamespace()
	log.Info(">>> "+logPrefix+"[APPLY] Retrieving KUBECONFIG.", "name", localName, "namespace", localNamespace)
	k8sClient, dynClient, err := buildHostClient(ctx, fed, localClient, scheme)
	if err != nil {
		log.Error(err, ">>> "+logPrefix+"[APPLY] Error building K8s Client.", "name", localName, "namespace", localNamespace)
		return err
	}
	reqKey := types.NamespacedName{
		Name:      remoteObj.GetName(),
		Namespace: remoteNamespace,
	}
	isNewResource := false

	if checkErr := k8sClient.Get(ctx, reqKey, emptyCheckObj); checkErr != nil {
		if apierrors.IsNotFound(checkErr) {
			isNewResource = true
		} else {
			log.Error(checkErr, ">>> "+logPrefix+"[APPLY] Failed to check remote resource existence via GET.", "name", localName, "namespace", localNamespace)
			return checkErr
		}
	}
	if err := k8sClient.Patch(ctx, remoteObj, client.Apply, client.ForceOwnership, client.FieldOwner(fieldOwner)); err != nil {
		log.Error(err, ">>> "+logPrefix+"[APPLY] Failed to APPLY/UPDATE.", "name", localName, "namespace", localNamespace)
		return err
	}
	log.Info(">>> "+logPrefix+"[APPLY] SUCCESSFULLY APPLIED/UPDATED.", "name", localName, "namespace", localNamespace)
	if isNewResource {
		StartRemoteResourceWatcher(ctx, dynClient, remoteNamespace, localName, localNamespace, group, version, plural)
		log.Info(">>> "+logPrefix+"[APPLY] SUCCESSFULLY STARTED background WATCHER.", "name", localName, "namespace", localNamespace)
	}
	return nil
}

// GetRemoteResource retrieves a remote Kubernetes resource from the host cluster based on the provided object and updates the local object with its status.
func GetRemoteResource(
	ctx context.Context,
	localClient client.Client,
	scheme *runtime.Scheme,
	fed *v1beta1.Federation,
	remoteObj client.Object,
	remoteName string,
	localName string,
	localNamespace string,
	logPrefix string,
) error {
	log := ctrl.Log
	log.Info(">>> "+logPrefix+"[GET] Retrieving KUBECONFIG.", "name", localName, "namespace", localNamespace)
	k8sClient, _, err := buildHostClient(ctx, fed, localClient, scheme)
	if err != nil {
		log.Error(err, ">>> "+logPrefix+"[GET] Error building K8s Client.", "name", localName, "namespace", localNamespace)
		return err
	}
	reqKey := types.NamespacedName{
		Name:      remoteName,
		Namespace: fed.Spec.FederationData.K8sOptions.Namespace,
	}
	if err := k8sClient.Get(ctx, reqKey, remoteObj); err != nil {
		log.Error(err, ">>> "+logPrefix+"[GET] Failed to GET remote resource.", "name", localName, "namespace", localNamespace)
		return err
	}
	return nil
}

// PatchRemoteResource applies or updates a remote Kubernetes resource using a MergePatch.
func PatchRemoteResource(
	ctx context.Context,
	localClient client.Client,
	scheme *runtime.Scheme,
	fed *v1beta1.Federation,
	targetObj client.Object,
	localName string,
	localNamespace string,
	group string,
	version string,
	plural string,
	logPrefix string,
) error {
	log := ctrl.Log
	log.Info(">>> "+logPrefix+"[PATCH] Retrieving KUBECONFIG.", "name", localName, "namespace", localNamespace)
	k8sClient, _, err := buildHostClient(ctx, fed, localClient, scheme)
	if err != nil {
		log.Error(err, ">>> "+logPrefix+"[PATCH] Error building K8s Client.", "name", localName, "namespace", localNamespace)
		return err
	}
	reqKey := types.NamespacedName{
		Name:      targetObj.GetName(),
		Namespace: targetObj.GetNamespace(),
	}
	// Download the CURRENT state of the remote object from the cluster
	if err := k8sClient.Get(ctx, reqKey, targetObj); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info(">>> "+logPrefix+"[PATCH] Remote resource not found.", "name", localName, "namespace", localNamespace)
		} else {
			log.Error(err, ">>> "+logPrefix+"[PATCH] Failed to check remote resource existence.", "name", localName, "namespace", localNamespace)
			return err
		}
	}
	// Prepare the base for the Patch (save an exact copy of the current state)
	var patchBase = client.MergeFrom(targetObj.DeepCopyObject().(client.Object))
	// Execute the Mutator function!
	// This function will modify `targetObj` by adding or changing ONLY the fields you care about.
	// Perform a standard MergePatch. K8s will understand exactly which fields you changed by comparing targetObj with patchBase.
	if err := k8sClient.Patch(ctx, targetObj, patchBase); err != nil {
		log.Error(err, ">>> "+logPrefix+"[PATCH] Failed to PATCH the SPEC.", "name", localName, "namespace", localNamespace)
		return err
	}
	log.Info(">>> "+logPrefix+"[PATCH] SUCCESSFULLY PATCHED the SPEC.", "name", localName, "namespace", localNamespace)
	return nil
}

// DeleteRemoteResource deletes a remote Kubernetes resource from the host cluster based on the provided object and stops the watcher if it was running.
func DeleteRemoteResource(
	ctx context.Context,
	localClient client.Client,
	scheme *runtime.Scheme,
	fed *v1beta1.Federation,
	targetObj client.Object,
	remoteName string,
	localName string,
	localNamespace string,
	logPrefix string,
) error {
	log := ctrl.Log
	remoteNamespace := fed.Spec.FederationData.K8sOptions.Namespace
	log.Info(">>> "+logPrefix+"[DELETE] Retrieving KUBECONFIG", "name", localName, "namespace", localNamespace)
	k8sClient, _, err := buildHostClient(ctx, fed, localClient, scheme)
	if err != nil {
		log.Error(err, ">>> "+logPrefix+"[DELETE] Error building K8s Client.", "name", localName, "namespace", localNamespace)
		return err
	}
	targetObj.SetName(remoteName)
	targetObj.SetNamespace(remoteNamespace)
	if err := k8sClient.Delete(ctx, targetObj); err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, ">>> "+logPrefix+"[DELETE] Failed to DELETE.", "name", localName, "namespace", localNamespace)
		return err
	}
	log.Info(">>> "+logPrefix+"[DELETE] SUCCESSFULLY DELETED.", "name", localName, "namespace", localNamespace)
	if !StopRemoteResourceWatcher(remoteNamespace, localName) {
		log.Error(nil, ">>> "+logPrefix+"[DELETE] Problem during the stopping of the WATCHER.", "name", localName, "namespace", localNamespace)
	} else {
		log.Info(">>> "+logPrefix+"[DELETE] SUCCESSFULLY STOPPED background WATCHER.", "name", localName, "namespace", localNamespace)
	}
	return nil
}

func AddRemoteAnnotation(
	ctx context.Context,
	localClient client.Client,
	scheme *runtime.Scheme,
	fed *v1beta1.Federation,
	targetObj client.Object,
	localName string,
	localNamespace string,
	group string,
	version string,
	plural string,
	logPrefix string,
	annotationKey string,
	annotationValue string,
) error {
	log := ctrl.Log
	remoteNamespace := targetObj.GetNamespace()
	remoteName := targetObj.GetName()
	log.Info(">>> "+logPrefix+"[ANNOTATE] Retrieving KUBECONFIG.", "name", localName, "namespace", localNamespace)
	k8sClient, _, err := buildHostClient(ctx, fed, localClient, scheme)
	if err != nil {
		log.Error(err, ">>> "+logPrefix+"[ANNOTATE] Error building K8s Client.", "name", localName, "namespace", localNamespace)
		return err
	}
	reqKey := types.NamespacedName{
		Name:      remoteName,
		Namespace: remoteNamespace,
	}
	// Download the CURRENT state of the remote object from the cluster
	if err := k8sClient.Get(ctx, reqKey, targetObj); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info(">>> "+logPrefix+"[ANNOTATE] Remote resource not found.", "name", localName, "namespace", localNamespace)
		} else {
			log.Error(err, ">>> "+logPrefix+"[ANNOTATE] Failed to check remote resource existence.", "name", localName, "namespace", localNamespace)
			return err
		}
	}
	// Prepare the base for the Patch (save an exact copy of the current state)
	var patchBase = client.MergeFrom(targetObj.DeepCopyObject().(client.Object))
	currentAnnotations := targetObj.GetAnnotations()
	if currentAnnotations == nil {
		currentAnnotations = make(map[string]string)
	}
	currentAnnotations[annotationKey] = annotationValue
	targetObj.SetAnnotations(currentAnnotations)
	if err := k8sClient.Patch(ctx, targetObj, patchBase); err != nil {
		log.Error(err, ">>> "+logPrefix+"[ANNOTATE] Failed to PATCH remote resource.", "name", localName, "namespace", localNamespace)
		return err
	}
	log.Info(">>> "+logPrefix+"[ANNOTATE] SUCCESSFULLY PATCHED remote resource.", "name", localName, "namespace", localNamespace)
	return nil
}
