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

// ApplicationInstanceReconciler reconciles an Artefact object
type ApplicationInstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

const (
	errorUpdatingApplicationInstanceStatusMsg = ">>> [AppInst][K8s] Error Updating resource status"
)

func (r *ApplicationInstanceReconciler) CreateApplicationInstance(ctx context.Context, a *v1beta1.ApplicationInstance, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [AppInst][K8s] Using Kubernetes API to create application instance in federation host cluster")
	log.Info(">>> [AppInst][K8s] Retrieving kubeconfig from secret")
	kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, feder.Labels[v1beta1.FederationSecretNameLabel], a.Namespace)
	if err != nil {
		log.Error(err, ">>> [AppInst][K8s] Error getting kubeconfig from secret")
		return err
	}
	log.Info(">>> [AppInst][K8s] Building dynamic client with kubeconfig")
	hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, feder.Labels[v1beta1.FederationHostOPLabel])
	if err != nil {
		log.Error(err, ">>> [AppInst][K8s] Error building client with kubeconfig")
		return err
	}
	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(a)
	if err != nil {
		log.Error(err, ">>> [AppInst][K8s] Error converting ApplicationInstance to unstructured")
		return err
	}
	spec, found, err := unstructured.NestedFieldCopy(unstructuredMap, "spec")
	if err != nil || !found {

		log.Error(err, ">>> [AppInst][K8s] Spec extraction failed")
		return err
	}
	applicationObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "opg.ewbi.nby.one/v1beta1",
			"kind":       "ApplicationInstance",
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
	appliedRes, err := ApplyK8sResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "applicationinstances", feder.Labels[v1beta1.FederationNamespaceLabel], applicationObj, "applicationinstance-controller")
	if err != nil {
		log.Error(err, ">>> [AppInst][K8s] Error applying resource")
		return err
	}
	log.Info(">>> [AppInst][K8s] Successfully applied resource to remote cluster, starting watcher", "name", appliedRes.GetName())
	StartRemoteResourceWatcher(ctx, hostClient, feder.Labels[v1beta1.FederationNamespaceLabel], a.Name, a.Namespace, "opg.ewbi.nby.one", "v1beta1", "applicationinstances")
	return nil
}

func (r *ApplicationInstanceReconciler) SyncStatusWithHost(ctx context.Context, a *v1beta1.ApplicationInstance, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [AppInst][K8s] Using Kubernetes API to update application instance from federation host cluster")
	log.Info(">>> [AppInst][K8s] Retrieving kubeconfig from secret")
	kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, feder.Labels[v1beta1.FederationSecretNameLabel], a.Namespace)
	if err != nil {
		log.Error(err, ">>> [AppInst][K8s] Error getting kubeconfig from secret")
		return err
	}
	log.Info(">>> [AppInst][K8s] Building dynamic client with kubeconfig")
	hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, feder.Labels[v1beta1.FederationHostOPLabel])
	if err != nil {
		log.Error(err, ">>> [AppInst][K8s] Error building client with kubeconfig")
		return err
	}
	log.Info(">>> [AppInst][K8s] Retrieve current state from host federation")
	hostApplication, err := GetK8sResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "applicationinstances", feder.Labels[v1beta1.FederationNamespaceLabel], a.Name)
	if err != nil {
		log.Error(err, ">>> [AppInst][K8s] Failed to get federation resource from target cluster")
		return err
	}
	state, found, err := unstructured.NestedString(hostApplication.Object, "status", "state")
	if err != nil {
		log.Error(err, ">>> [AppInst][K8s] Error parsing status.state")
		return err
	}
	if !found {
		log.Info(">>> [AppInst][K8s] Status.state field not found in the resource", "application", a.Name)
		state = string(v1beta1.ApplicationInstanceStatePending) // Default to Pending if status is not yet set
	}
	log.Info(">>> [AppInst][K8s] Successfully retrieved state", "state", state)
	a.Status.State = v1beta1.ApplicationInstanceState(state)
	if state == string(v1beta1.ApplicationInstanceStateReady) {
		rawSlice, found, err := unstructured.NestedSlice(hostApplication.Object, "status", "accessPointInfo")
		if err != nil {
			log.Error(err, ">>> [AppInst][K8s] Error parsing status.accessPointInfo")
			return err
		}
		if found {
			var accessPoints []v1beta1.AccessPointInfo
			for _, rawItem := range rawSlice {
				if itemMap, ok := rawItem.(map[string]interface{}); ok {
					var ap v1beta1.AccessPointInfo
					err = runtime.DefaultUnstructuredConverter.FromUnstructured(itemMap, &ap)
					if err != nil {
						log.Error(err, ">>> [AppInst][K8s] Error converting single accessPointInfo item")
						return err
					}
					accessPoints = append(accessPoints, ap)
				}
			}
			a.Status.AccessPointInfo = accessPoints
		}
	}
	upErr := r.Status().Update(ctx, a.DeepCopy())
	if upErr != nil {
		log.Error(upErr, errorUpdatingApplicationInstanceStatusMsg)
		return upErr
	}
	return nil
}

func (r *ApplicationInstanceReconciler) DeleteApplicationInstance(ctx context.Context, a *v1beta1.ApplicationInstance, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [AppInst][K8s] Deleting external application instance via Kubernetes API")
	kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, feder.Labels[v1beta1.FederationSecretNameLabel], a.Namespace)
	if err != nil {
		log.Error(err, ">>> [AppInst][K8s] Error getting kubeconfig from secret")
		return err
	}
	log.Info(">>> [AppInst][K8s] Building dynamic client with kubeconfig")
	hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, feder.Labels[v1beta1.FederationHostOPLabel])
	if err != nil {
		log.Error(err, ">>> [AppInst][K8s] Error building client with kubeconfig")
		return err
	}
	err = DeleteK8sResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "applicationinstances", feder.Labels[v1beta1.FederationNamespaceLabel], a.Name)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, ">>> [AppInst][K8s] Failed to delete application instance resource from target cluster")
		return err
	}
	log.Info(">>> [AppInst][K8s] Stopping background watcher for remote host")
	StopRemoteResourceWatcher(feder.Labels[v1beta1.FederationHostIdLabel], a.Name)
	return nil
}
