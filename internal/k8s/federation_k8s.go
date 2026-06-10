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

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// FederationReconciler reconciles a Federation object
type FederationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

func (r *FederationReconciler) CreateFederation(ctx context.Context, f *v1beta1.Federation) (statusChanged bool, err error) {
	log := log.FromContext(ctx)
	log.Info(">>> [Federation][K8s] Using Kubernetes API to create federation")
	log.Info(">>> [Federation][K8s] Retrieving kubeconfig from secret")
	kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, f.Labels[v1beta1.FederationSecretNameLabel], f.Namespace)
	if err != nil {
		log.Error(err, ">>> [Federation][K8s] Error getting kubeconfig from secret")
		return false, err
	}
	log.Info(">>> [Federation][K8s] Building dynamic client with kubeconfig")
	hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, f.Labels[v1beta1.FederationHostOPLabel])
	if err != nil {
		log.Error(err, ">>> [Federation][K8s] Error building client with kubeconfig")
		return false, err
	}
	fedPatch, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"initialDate": f.Spec.InitialDate.Format("2006-01-02T15:04:05Z"),
			"originOP": map[string]interface{}{
				"countryCode":       f.Spec.OriginOP.CountryCode,
				"fixedNetworkCodes": f.Spec.OriginOP.FixedNetworkCodes,
				"mobileNetworkCodes": map[string]interface{}{
					"mcc":  f.Spec.OriginOP.MobileNetworkCodes.MCC,
					"mncs": f.Spec.OriginOP.MobileNetworkCodes.MNC,
				},
			},
			"partner": map[string]interface{}{
				"callbackCredentials": map[string]interface{}{
					"clientId": f.Spec.Partner.CallbackCredentials.ClientId,
				},
				"statusLink": f.Spec.Partner.StatusLink,
			},
		},
	})
	if err != nil {
		log.Error(err, ">>> [Federation][K8s] Error marshaling federation patch")
		return false, err
	}
	log.Info(">>> [Federation][K8s] Patching Federation resource in host cluster")
	err = PatchK8sResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "federations", f.Labels[v1beta1.FederationNamespaceLabel], f.Labels[v1beta1.FederationHostIdLabel], k8stypes.PatchType("application/merge-patch+json"), fedPatch)
	if err != nil {
		log.Error(err, ">>> [Federation][K8s] Failed to apply patch to target cluster resource")
		return false, err
	}
	log.Info(">>> [Federation][K8s] Successfully patched Federation resource in host cluster")
	StartRemoteResourceWatcher(ctx, hostClient, f.Labels[v1beta1.FederationNamespaceLabel], f.Name, f.Namespace, "opg.ewbi.nby.one", "v1beta1", "federations")
	return true, nil
}

func (r *FederationReconciler) SyncStatusWithHost(ctx context.Context, f *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [Federation][K8s] Syncing federation status with host cluster")
	log.Info(">>> [Federation][K8s] Retrieve current state from host federation")
	log.Info(">>> [Federation][K8s] Using Kubernetes API to create federation")
	log.Info(">>> [Federation][K8s] Retrieving kubeconfig from secret")
	kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, f.Labels[v1beta1.FederationSecretNameLabel], f.Namespace)
	if err != nil {
		log.Error(err, ">>> [Federation][K8s] Error getting kubeconfig from secret")
		return err
	}
	log.Info(">>> [Federation][K8s] Building dynamic client with kubeconfig")
	hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, f.Labels[v1beta1.FederationHostOPLabel])
	if err != nil {
		log.Error(err, ">>> [Federation][K8s] Error building client with kubeconfig")
		return err
	}
	hostFed, err := GetK8sResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "federations", f.Labels[v1beta1.FederationNamespaceLabel], f.Labels[v1beta1.FederationHostIdLabel])
	if err != nil {
		log.Error(err, ">>> [Federation][K8s] Failed to get federation resource from target cluster")
		return err // Errore di rete, riproviamo
	}
	offeredAZs, azFound, _ := unstructured.NestedSlice(hostFed.Object, "spec", "offeredAvailabilityZones")
	// log.Info(">>> [Federation][K8s] Zones:", "offeredAZs", offeredAZs)
	federationContextId, ctxFound, _ := unstructured.NestedString(hostFed.Object, "status", "federationContextId")
	if !azFound || !ctxFound {
		log.Info(">>> [Federation][K8s] Remote data not yet ready. Waiting for watch events...")
		return nil
	}

	zones := []v1beta1.ZoneDetails{}
	for _, zRaw := range offeredAZs {
		// Il parsing da interface{} dipende dalla struttura esatta della tua AZ
		zMap := zRaw.(map[string]interface{})
		zones = append(zones, v1beta1.ZoneDetails{
			// Estrai i dati mappandoli correttamente, es:
			// ZoneId: zMap["zoneId"].(string),
			ZoneId:           zMap["zoneId"].(string),
			Geolocation:      zMap["geolocation"].(string),
			GeographyDetails: zMap["geographyDetails"].(string),
		})
	}

	if compareSameAZs(f.Status.OfferedAvailabilityZones, zones) &&
		f.Status.State == v1beta1.FederationStateAvailable {
		return nil // Tutto è già allineato
	}

	f.Status.OfferedAvailabilityZones = zones
	f.Status.State = v1beta1.FederationStateAvailable
	f.Status.FederationContextId = federationContextId

	upErr := r.Status().Update(ctx, f.DeepCopy())
	if upErr != nil {
		log.Error(upErr, ">>> [Federation][K8s] Error Updating resource status", "federation", f.Name)
		return upErr
	}
	return nil
}

func compareSameAZs(s1, s2 []v1beta1.ZoneDetails) bool {
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

func (r *FederationReconciler) AcceptExternalAZ(ctx context.Context, f *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	if len(f.Status.OfferedAvailabilityZones) == 0 {
		log.Info(">>> [Federation][K8s] No offered AZs discovered from host yet, skipping acceptance for now")
		return nil
	}
	az := f.Status.OfferedAvailabilityZones[0].ZoneId
	if len(f.Spec.AcceptedAvailabilityZones) > 0 && f.Spec.AcceptedAvailabilityZones[0] == az {
		log.Info(">>> [Federation][K8s] AZ already accepted locally, skipping patch")
		return nil
	}
	log.Info(">>> [Federation][K8s] Accepting AZ in Kubernetes federation via cross-cluster patch", "az", az)
	kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, f.Labels[v1beta1.FederationSecretNameLabel], f.Namespace)
	if err != nil {
		log.Error(err, ">>> [Federation][K8s] Error getting kubeconfig from secret")
		return err
	}
	log.Info(">>> [Federation][K8s] Building dynamic client with kubeconfig")
	hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, f.Labels[v1beta1.FederationHostOPLabel])
	if err != nil {
		log.Error(err, ">>> [Federation][K8s] Error building client with kubeconfig")
		return err
	}
	fedPatch, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"acceptedAvailabilityZones": []string{az},
		},
	})
	if err != nil {
		log.Error(err, ">>> [Federation][K8s] Error marshaling patch")
		return err
	}
	err = PatchK8sResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "federations", f.Labels[v1beta1.FederationNamespaceLabel], f.Labels[v1beta1.FederationHostIdLabel], k8stypes.PatchType("application/merge-patch+json"), fedPatch)
	if err != nil {
		log.Error(err, ">>> [Federation][K8s] Failed to apply patch to target cluster resource")
		return err
	}
	log.Info(">>> [Federation][K8s] Successfully patched Federation resource in host cluster")
	f.Spec.AcceptedAvailabilityZones = []string{az}
	if err := r.Update(ctx, f.DeepCopy()); err != nil {
		log.Error(err, ">>> [Federation][K8s] Failed to update local spec with accepted AZ")
		return err
	}
	return nil
}

func (r *FederationReconciler) DeleteFederation(ctx context.Context, f *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [Federation][K8s] Deleting external federation via Kubernetes API")
	kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, f.Labels[v1beta1.FederationSecretNameLabel], f.Namespace)
	if err != nil {
		log.Error(err, ">>> [Federation][K8s] Error getting kubeconfig from secret")
		return err
	}
	log.Info(">>> [Federation][K8s] Building dynamic client with kubeconfig")
	hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, f.Labels[v1beta1.FederationHostOPLabel])
	if err != nil {
		log.Error(err, ">>> [Federation][K8s] Error building client with kubeconfig")
		return err
	}
	err = DeleteK8sResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "federations", f.Labels[v1beta1.FederationNamespaceLabel], f.Labels[v1beta1.FederationHostIdLabel])
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, ">>> [Federation][K8s] Failed to delete federation resource from target cluster")
		return err
	}
	log.Info(">>> [Federation][K8s] Stopping background watcher for remote host")
	StopRemoteResourceWatcher(f.Labels[v1beta1.FederationHostIdLabel], f.Name)
	return nil
}
