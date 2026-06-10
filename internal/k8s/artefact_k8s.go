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

// ArtefactReconciler reconciles an Artefact object
type ArtefactReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

const (
	errorUpdatingArtefactStatusMsg = ">>> [Artefact] Error Updating resource status"
	unexpectedStatusArtefactMsg    = ">>> [Artefact] Unexpected Status Code"
)

func (r *ArtefactReconciler) CreateArtefact(ctx context.Context, a *v1beta1.Artefact, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [Artefact][K8s] Using Kubernetes API to create artefact in federation host cluster")
	log.Info(">>> [Artefact][K8s] Retrieving kubeconfig from secret")
	kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, feder.Labels[v1beta1.FederationSecretNameLabel], a.Namespace)
	if err != nil {
		log.Error(err, ">>> [Artefact][K8s] Error getting kubeconfig from secret")
		return err
	}
	log.Info(">>> [Artefact][K8s] Building dynamic client with kubeconfig")
	hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, feder.Labels[v1beta1.FederationHostOPLabel])
	if err != nil {
		log.Error(err, ">>> [Artefact][K8s] Error building client with kubeconfig")
		return err
	}
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(a)
	if err != nil {
		log.Error(err, ">>> [Artefact][K8s] Error converting Artefact to unstructured")
		return err
	}
	spec, found, err := unstructured.NestedFieldCopy(unstructuredMap, "spec")
	if err != nil || !found {

		log.Error(err, ">>> [Artefact][K8s] Spec extraction failed")
		return err
	}
	artefactObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "opg.ewbi.nby.one/v1beta1",
			"kind":       "Artefact",
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
	appliedRes, err := ApplyK8sResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "artefacts", feder.Labels[v1beta1.FederationNamespaceLabel], artefactObj, "artefact-controller")
	if err != nil {
		log.Error(err, ">>> [Artefact][K8s] Error applying resource")
		return err
	}
	log.Info(">>> [Artefact][K8s] Successfully applied resource to remote cluster, starting watcher", "name", appliedRes.GetName())
	StartRemoteResourceWatcher(ctx, hostClient, feder.Labels[v1beta1.FederationNamespaceLabel], a.Name, a.Namespace, "opg.ewbi.nby.one", "v1beta1", "artefacts")
	return nil
}

func (r *ArtefactReconciler) SyncStatusWithHost(ctx context.Context, a *v1beta1.Artefact, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [Artefact][K8s] Using Kubernetes API to update artefact status from federation host cluster")
	log.Info(">>> [Artefact][K8s] Retrieving kubeconfig from secret")
	kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, feder.Labels[v1beta1.FederationSecretNameLabel], a.Namespace)
	if err != nil {
		log.Error(err, ">>> [Artefact][K8s] Error getting kubeconfig from secret")
		return err
	}
	log.Info(">>> [Artefact][K8s] Building dynamic client with kubeconfig")
	hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, feder.Labels[v1beta1.FederationHostOPLabel])
	if err != nil {
		log.Error(err, ">>> [Artefact][K8s] Error building client with kubeconfig")
		return err
	}
	log.Info(">>> [Artefact][K8s] Retrieve current state from host federation")
	hostArtefact, err := GetK8sResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "artefacts", feder.Labels[v1beta1.FederationNamespaceLabel], a.Name)
	if err != nil {
		log.Error(err, ">>> [Artefact][K8s] Failed to get federation resource from target cluster")
		return err
	}
	state, found, err := unstructured.NestedString(hostArtefact.Object, "status", "state")
	if err != nil {
		log.Error(err, ">>> [Artefact][K8s] Error parsing status.state")
		return err
	}
	if !found {
		log.Info(">>> [Artefact][K8s] Status.state field not found in the resource", "artefact", a.Name)
		state = string(v1beta1.ArtefactStateReconciling) // Default to Reconciling if status is not yet set
	}
	log.Info(">>> [Artefact][K8s] Successfully retrieved state", "state", state)
	a.Status.State = v1beta1.ArtefactState(state)
	upErr := r.Status().Update(ctx, a.DeepCopy())
	if upErr != nil {
		log.Error(upErr, errorUpdatingArtefactStatusMsg)
		return upErr
	}
	return nil
}

func (r *ArtefactReconciler) DeleteArtefact(ctx context.Context, a *v1beta1.Artefact, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [Artefact][K8s] Deleting external artefact via Kubernetes API")
	kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, feder.Labels[v1beta1.FederationSecretNameLabel], a.Namespace)
	if err != nil {
		log.Error(err, ">>> [Artefact][K8s] Error getting kubeconfig from secret")
		return err
	}
	log.Info(">>> [Artefact][K8s] Building dynamic client with kubeconfig")
	hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, feder.Labels[v1beta1.FederationHostOPLabel])
	if err != nil {
		log.Error(err, ">>> [Artefact][K8s] Error building client with kubeconfig")
		return err
	}
	err = DeleteK8sResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "artefacts", feder.Labels[v1beta1.FederationNamespaceLabel], a.Name)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, ">>> [Artefact][K8s] Failed to delete artefact resource from target cluster")
		return err
	}
	log.Info(">>> [Artefact][K8s] Stopping background watcher for remote host")
	StopRemoteResourceWatcher(feder.Labels[v1beta1.FederationHostIdLabel], a.Name)
	return nil
}
