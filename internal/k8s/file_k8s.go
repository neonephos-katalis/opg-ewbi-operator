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
	"fmt"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// FileReconciler reconciles a File object
type FileReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

const (
	errorUpdatingFileStatusMsg = ">>> [File][K8s] Error Updating resource status"
	unexpectedStatusFileMsg    = ">>> [File][K8s] Unexpected Status Code"
)

func (r *FileReconciler) CreateFile(ctx context.Context, f *v1beta1.File, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [File][K8s] Using Kubernetes API to create file in federation host cluster")
	log.Info(">>> [File][K8s] Retrieving kubeconfig from secret")
	kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, feder.Labels[v1beta1.FederationSecretNameLabel], f.Namespace)
	if err != nil {
		log.Error(err, ">>> [File][K8s] Error getting kubeconfig from secret")
		return err
	}
	log.Info(">>> [File][K8s] Building dynamic client with kubeconfig")
	hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, feder.Labels[v1beta1.FederationHostOPLabel])
	if err != nil {
		log.Error(err, ">>> [File][K8s] Error building client with kubeconfig")
		return err
	}
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(f)
	if err != nil {
		log.Error(err, ">>> [File][K8s] Error converting File to unstructured")
		return err
	}
	spec, found, err := unstructured.NestedFieldCopy(unstructuredMap, "spec")
	if err != nil || !found {
		err := fmt.Errorf("impossibile trovare o estrarre la spec dalla risorsa: %w", err)
		log.Error(err, ">>> [File][K8s] Spec extraction failed")
		return err
	}
	fileObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "opg.ewbi.nby.one/v1beta1",
			"kind":       "File",
			"metadata": map[string]interface{}{
				"name":      f.Name,
				"namespace": feder.Labels[v1beta1.FederationNamespaceLabel],
				"labels": map[string]interface{}{
					"opg.ewbi.nby.one/federation-relation":   "host",
					"opg.ewbi.nby.one/federation-context-id": feder.Status.FederationContextId,
				},
			},
			"spec": spec,
		},
	}
	appliedRes, err := ApplyK8sResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "files", feder.Labels[v1beta1.FederationNamespaceLabel], fileObj, "file-controller")
	if err != nil {
		log.Error(err, ">>> [File][K8s] Error applying resource")
		return err
	}
	log.Info(">>> [File][K8s] Successfully applied resource to remote cluster, starting watcher", "name", appliedRes.GetName())
	StartRemoteResourceWatcher(ctx, hostClient, feder.Labels[v1beta1.FederationNamespaceLabel], f.Name, f.Namespace, "opg.ewbi.nby.one", "v1beta1", "files")
	return nil
}

func (r *FileReconciler) SyncStatusWithHost(ctx context.Context, f *v1beta1.File, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [File][K8s] Using Kubernetes API to update file status from federation host cluster")
	log.Info(">>> [File][K8s] Retrieving kubeconfig from secret")
	kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, feder.Labels[v1beta1.FederationSecretNameLabel], f.Namespace)
	if err != nil {
		log.Error(err, ">>> [File][K8s] Error getting kubeconfig from secret")
		return err
	}
	log.Info(">>> [File][K8s] Building dynamic client with kubeconfig")
	hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, feder.Labels[v1beta1.FederationHostOPLabel])
	if err != nil {
		log.Error(err, ">>> [File][K8s] Error building client with kubeconfig")
		return err
	}
	log.Info(">>> [File][K8s] Retrieve current state from host federation")
	hostFile, err := GetK8sResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "files", feder.Labels[v1beta1.FederationNamespaceLabel], f.Name)
	if err != nil {
		log.Error(err, ">>> [File][K8s] Failed to get federation resource from target cluster")
		return err
	}
	state, found, err := unstructured.NestedString(hostFile.Object, "status", "state")
	if err != nil {
		log.Error(err, ">>> [File][K8s] Error parsing status.state")
		return err
	}
	if !found {
		log.Info(">>> [File][K8s] stats.state field not found in the resource", "file", f.Name)
		state = string(v1beta1.FileStatePending)
	}
	log.Info(">>> [File][K8s] Successfully retrieved state", "state", state)
	f.Status.State = v1beta1.FileState(state)
	upErr := r.Status().Update(ctx, f.DeepCopy())
	if upErr != nil {
		log.Error(upErr, errorUpdatingFileStatusMsg)
		return upErr
	}
	return nil
}
func (r *FileReconciler) DeleteFile(ctx context.Context, f *v1beta1.File, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [File][K8s] Deleting external file via Kubernetes API")
	kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, feder.Labels[v1beta1.FederationSecretNameLabel], f.Namespace)
	if err != nil {
		log.Error(err, ">>> [File][K8s] Error getting kubeconfig from secret")
		return err
	}
	log.Info(">>> [File][K8s] Building dynamic client with kubeconfig")
	hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, feder.Labels[v1beta1.FederationHostOPLabel])
	if err != nil {
		log.Error(err, ">>> [File][K8s] Error building client with kubeconfig")
		return err
	}
	err = DeleteK8sResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "files", feder.Labels[v1beta1.FederationNamespaceLabel], f.Name)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, ">>> [File][K8s] Failed to delete file resource from target cluster")
		return err
	}
	log.Info(">>> [File][K8s] Stopping background watcher for remote host")
	StopRemoteResourceWatcher(f.Labels[v1beta1.FederationHostIdLabel], f.Name)
	return nil
}
