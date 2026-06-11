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

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ApplicationReconciler reconciles an Artefact object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

const (
	errorUpdatingApplicationStatusMsg = ">>> [App][K8s] Error Updating resource status"
)

func (r *ApplicationReconciler) CreateApplication(ctx context.Context, a *v1beta1.Application, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [App][K8s] Using Kubernetes API to create application in federation host cluster")
	log.Info(">>> [App][K8s] Retrieving kubeconfig from secret")
	kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, feder.Labels[v1beta1.FederationSecretNameLabel], a.Namespace)
	if err != nil {
		log.Error(err, ">>> [App][K8s] Error getting kubeconfig from secret")
		return err
	}
	log.Info(">>> [App][K8s] Building dynamic client with kubeconfig")
	hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, feder.Labels[v1beta1.FederationHostOPLabel])
	if err != nil {
		log.Error(err, ">>> [App][K8s] Error building client with kubeconfig")
		return err
	}
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(a)
	if err != nil {
		log.Error(err, ">>> [App][K8s] Error converting Application to unstructured")
		return err
	}
	spec, found, err := unstructured.NestedFieldCopy(unstructuredMap, "spec")
	if err != nil || !found {

		log.Error(err, ">>> [App][K8s] Spec extraction failed")
		return err
	}
	applicationObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "opg.ewbi.nby.one/v1beta1",
			"kind":       "Application",
			"metadata": map[string]interface{}{
				"name":      a.Name,
				"namespace": feder.Labels[v1beta1.FederationNamespaceLabel],
				"labels": map[string]interface{}{
					"opg.ewbi.nby.one/federation-relation":   "host",
					"opg.ewbi.nby.one/federation-context-id": feder.Status.FederationContextId,
				},
			},
			"spec": spec,
		},
	}
	appliedRes, err := ApplyK8sResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "applications", feder.Labels[v1beta1.FederationNamespaceLabel], applicationObj, "application-controller")
	if err != nil {
		log.Error(err, ">>> [App][K8s] Error applying resource")
		return err
	}
	log.Info(">>> [App][K8s] Successfully applied resource to remote cluster, starting watcher", "name", appliedRes.GetName())
	StartRemoteResourceWatcher(ctx, hostClient, feder.Labels[v1beta1.FederationNamespaceLabel], a.Name, a.Namespace, "opg.ewbi.nby.one", "v1beta1", "applications")
	return nil
}

func (r *ApplicationReconciler) SyncStatusWithHost(ctx context.Context, a *v1beta1.Application, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [App][K8s] Using Kubernetes API to update application from federation host cluster")
	log.Info(">>> [App][K8s] Retrieving kubeconfig from secret")
	kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, feder.Labels[v1beta1.FederationSecretNameLabel], a.Namespace)
	if err != nil {
		log.Error(err, ">>> [App][K8s] Error getting kubeconfig from secret")
		return err
	}
	log.Info(">>> [App][K8s] Building dynamic client with kubeconfig")
	hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, feder.Labels[v1beta1.FederationHostOPLabel])
	if err != nil {
		log.Error(err, ">>> [App][K8s] Error building client with kubeconfig")
		return err
	}
	log.Info(">>> [App][K8s] Retrieve current state from host federation")
	hostApplication, err := GetK8sResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "applications", feder.Labels[v1beta1.FederationNamespaceLabel], a.Name)
	if err != nil {
		log.Error(err, ">>> [App][K8s] Failed to get federation resource from target cluster")
		return err
	}
	state, found, err := unstructured.NestedString(hostApplication.Object, "status", "state")
	if err != nil {
		log.Error(err, ">>> [App][K8s] Error parsing status.state")
		return err
	}
	if !found {
		log.Info(">>> [App][K8s] Status.state field not found in the resource", "application", a.Name)
		state = string(v1beta1.ApplicationStatePending) // Default to Pending if status is not yet set
	}
	log.Info(">>> [App][K8s] Successfully retrieved state", "state", state)
	a.Status.State = v1beta1.ApplicationState(state)
	upErr := r.Status().Update(ctx, a.DeepCopy())
	if upErr != nil {
		log.Error(upErr, errorUpdatingApplicationStatusMsg)
		return upErr
	}
	return nil
}

func (r *ApplicationReconciler) DeleteApplication(ctx context.Context, a *v1beta1.Application, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [App][K8s] Deleting external application via Kubernetes API")
	kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, feder.Labels[v1beta1.FederationSecretNameLabel], a.Namespace)
	if err != nil {
		log.Error(err, ">>> [App][K8s] Error getting kubeconfig from secret")
		return err
	}
	log.Info(">>> [App][K8s] Building dynamic client with kubeconfig")
	hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, feder.Labels[v1beta1.FederationHostOPLabel])
	if err != nil {
		log.Error(err, ">>> [App][K8s] Error building client with kubeconfig")
		return err
	}
	err = DeleteK8sResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "applications", feder.Labels[v1beta1.FederationNamespaceLabel], a.Name)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, ">>> [App][K8s] Failed to delete application resource from target cluster")
		return err
	}
	log.Info(">>> [App][K8s] Stopping background watcher for remote host")
	StopRemoteResourceWatcher(feder.Labels[v1beta1.FederationHostIdLabel], a.Name)
	return nil
}
