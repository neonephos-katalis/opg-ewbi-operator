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
	"encoding/json"

	opgmodels "github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/watcher"
	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/uuid"
)

const (
	errorCreatingFederationMsg = "error creating federation"
)

// FederationReconciler reconciles a Federation object
type FederationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=federations,verbs=*,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=federations/status,verbs=get;update;patch,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=federations/finalizers,verbs=update,namespace=foo

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Modify the Reconcile function to compare the state specified by
// the Federation object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *FederationReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)

	log.Info(">>> [Federation] Starting reconcile function for federation")
	defer log.Info(">>> [Federation] End reconcile for federation")

	// Getting main federation or requeue
	var f v1beta1.Federation
	if err := r.Get(ctx, req.NamespacedName, &f); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info(">>> [Federation] Object not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "XXX [Federation] Error getting federation object")
		return ctrl.Result{}, err
	}
	log.Info(">>> [Federation] Object obtained", "name", f.Name, "originOP", f.Spec.OriginOP)
	isGuest := IsGuestResource(f.Labels)
	isRest := IsRestTechnology(f.Labels)
	log.Info(">>> [Federation] Resource type evaluation", "isGuest", isGuest, "isRest", isRest)
	if f.GetDeletionTimestamp().IsZero() {
		if controllerutil.AddFinalizer(&f, v1beta1.FederationFinalizer) {
			log.Info(">>> [Federation] Added finalizer to Federation")
			if err := r.Update(ctx, f.DeepCopy()); err != nil {
				log.Info(">>> [Federation] Unable to Update Federation with finalizer")
				return ctrl.Result{}, err
			}
			log.Info(">>> [Federation] Successfully added finalizer to federation")
			return ctrl.Result{}, nil
		}
	} else {
		if isGuest {
			if err := r.handleExternalFederationDeletion(ctx, &f, isRest); err != nil {
				log.Error(err, "XXX [Federation] Error deleting federation")
				f.Status.State = v1beta1.FederationStateNotAvailable
				upErr := r.Status().Update(ctx, f.DeepCopy())
				if upErr != nil {
					log.Error(upErr, errorUpdatingResourceStatusMsg)
				}
				return ctrl.Result{}, err
			}
		}
		// if external federation is correctly deleted, we can remove the finalizer
		if controllerutil.RemoveFinalizer(&f, v1beta1.FederationFinalizer) {
			log.Info("Removed basic finalizer for Federation")
			if err := r.Update(ctx, f.DeepCopy()); err != nil {
				log.Error(err, "update failed while removing finalizers")
				return ctrl.Result{}, err
			}
			log.Info("removed all finalizers, exiting...")
			return ctrl.Result{}, nil
		}
	}

	// if federation is guest, send OPG API request
	if isGuest {
		updated, err := r.handleExternalFederationCreation(ctx, &f, isRest)
		if err != nil {
			log.Error(err, errorCreatingFederationMsg)
			f.Status.State = v1beta1.FederationStateNotAvailable
			upErr := r.Status().Update(ctx, f.DeepCopy())
			if upErr != nil {
				log.Error(upErr, errorUpdatingResourceStatusMsg)
			}
			return ctrl.Result{}, err
		}
		if updated {
			// return, we will accept the AZ at the next reconcile
			return ctrl.Result{}, nil
		}

		if err := r.handleAcceptExternalAZ(ctx, &f, isRest); err != nil {
			log.Error(err, "error accepting az federation")
			f.Status.State = v1beta1.FederationStateNotAvailable
			upErr := r.Status().Update(ctx, f.DeepCopy())
			if upErr != nil {
				log.Error(upErr, errorUpdatingResourceStatusMsg)
			}
			return ctrl.Result{}, err
		}
	} else {
		if f.Status.State == "" {
			f.Status.State = v1beta1.FederationStateNotAvailable
			if !isRest {
				f.Status.FederationContextId = uuid.V5(f.Spec.GuestPartnerCredentials.ClientId)
			}
		}
		if f.Spec.AcceptedAvailabilityZones != nil {
			f.Status.State = v1beta1.FederationStateAvailable
		}
		upErr := r.Status().Update(ctx, f.DeepCopy())
		if upErr != nil {
			log.Error(upErr, errorUpdatingResourceStatusMsg)
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FederationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Federation{}).
		Named("federation").
		// Add watch on the channel receiving events from remote clusters, to trigger reconciliation when an event is received
		WatchesRawSource(
			source.Channel(
				watcher.RemoteFederationEvents,
				&handler.EnqueueRequestForObject{},
			),
		).
		Complete(r)
}

func (r *FederationReconciler) handleExternalFederationCreation(
	ctx context.Context, f *v1beta1.Federation, isRest bool) (statusChanged bool, err error) {
	log := log.FromContext(ctx)
	if isRest {
		log.Info(">>> [Federation] Using OPG API to create federation")
		fedReq := opgmodels.FederationRequestData{
			InitialDate:             f.Spec.InitialDate.Time,
			OrigOPCountryCode:       &f.Spec.OriginOP.CountryCode,
			OrigOPFederationId:      f.Labels[v1beta1.ExternalIdLabel],
			OrigOPFixedNetworkCodes: &f.Spec.OriginOP.FixedNetworkCodes,
			OrigOPMobileNetworkCodes: &opgmodels.MobileNetworkIds{
				Mcc:  &f.Spec.OriginOP.MobileNetworkCodes.MCC,
				Mncs: &f.Spec.OriginOP.MobileNetworkCodes.MNC,
			},
			PartnerCallbackCredentials: &opgmodels.CallbackCredentials{
				ClientId: f.Spec.Partner.CallbackCredentials.ClientId,
				TokenUrl: f.Spec.Partner.CallbackCredentials.TokenUrl,
			},
			PartnerStatusLink: f.Spec.Partner.StatusLink,
		}

		res, err := r.GetOPGClient(
			f.Labels[v1beta1.ExternalIdLabel],
			f.Spec.GuestPartnerCredentials.TokenUrl,
			f.Spec.GuestPartnerCredentials.ClientId,
		).CreateFederationWithResponse(
			context.TODO(),
			fedReq,
		)
		if err != nil {
			log.Error(err, errorCreatingFederationMsg)
			return false, err
		}

		statusCode := res.StatusCode()

		switch {
		case statusCode >= 200 && statusCode < 300:
			log.Info("Created", "response", res.JSON200)
			federResponse := res.JSON200
			zones := []v1beta1.ZoneDetails{}
			if federResponse.OfferedAvailabilityZones != nil {
				for _, z := range *federResponse.OfferedAvailabilityZones {
					zones = append(zones, v1beta1.ZoneDetails{
						GeographyDetails: z.GeographyDetails,
						Geolocation:      z.Geolocation,
						ZoneId:           z.ZoneId,
					})
				}
			}

			if compareSameAZs(f.Status.OfferedAvailabilityZones, zones) && f.Status.State == v1beta1.FederationStateAvailable {
				return false, nil
			}
			f.Status.OfferedAvailabilityZones = zones
			f.Status.State = v1beta1.FederationStateAvailable
			f.Status.FederationContextId = *federResponse.FederationContextId

			upErr := r.Status().Update(ctx, f.DeepCopy())
			if upErr != nil {
				log.Error(upErr, "Error Updating resource", "federation", f.Name)
				return false, upErr
			}

		case statusCode == 400:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		case statusCode == 401:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		case statusCode == 404:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		case statusCode == 409:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		case statusCode == 422:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		case statusCode == 500:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		case statusCode == 503:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		case statusCode == 520:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		default:
			log.Info(unexpectedStatusCodeMsg, "status", statusCode, "body", string(res.Body))
		}
	} else {
		log.Info(">>> [Federation] Using Kubernetes API to create federation")
		log.Info(">>> [Federation] Retrieving kubeconfig from secret")
		kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, f.Labels[v1beta1.FederationSecretNameLabel], f.Namespace)
		if err != nil {
			log.Error(err, ">>> [Federation] Error getting kubeconfig from secret")
			return false, err
		}
		log.Info(">>> [Federation] Building dynamic client with kubeconfig")
		hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, f.Labels[v1beta1.FederationHostOPLabel])
		if err != nil {
			log.Error(err, ">>> [Federation] Error building client with kubeconfig")
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
			log.Error(err, ">>> [Federation] Error marshaling federation patch")
			return false, err
		}
		log.Info(">>> [Federation] Patching Federation resource in host cluster")
		err = PatchResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "federations", f.Labels[v1beta1.FederationNamespaceLabel], f.Labels[v1beta1.FederationHostIdLabel], k8stypes.PatchType("application/merge-patch+json"), fedPatch)
		if err != nil {
			log.Error(err, ">>> [Federation] Failed to apply patch to target cluster resource")
			return false, err
		}
		log.Info(">>> [Federation] Successfully patched Federation resource in host cluster")
		watcher.StartRemoteWatcherFederation(ctx, hostClient, f.Labels[v1beta1.FederationNamespaceLabel], f.Name, f.Namespace)

		log.Info(">>> [Federation] Retrieve current state from host federation")
		hostFed, err := GetResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "federations", f.Labels[v1beta1.FederationNamespaceLabel], f.Labels[v1beta1.FederationHostIdLabel])
		if err != nil {
			log.Error(err, ">>> [Federation] Failed to get federation resource from target cluster")
			return false, err // Errore di rete, riproviamo
		}
		offeredAZs, azFound, _ := unstructured.NestedSlice(hostFed.Object, "spec", "offeredAvailabilityZones")
		log.Info(">>> [Federation] Zones:", "offeredAZs", offeredAZs)
		federationContextId, ctxFound, _ := unstructured.NestedString(hostFed.Object, "status", "federationContextId")
		if !azFound || !ctxFound {
			log.Info(">>> [Federation] Remote data not yet ready. Waiting for watch events...")
			return false, nil
		}

		// 3. Costruiamo le Zone
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

		// 4. Controlliamo se c'è un effettivo cambiamento per evitare loop infiniti di update
		if compareSameAZs(f.Status.OfferedAvailabilityZones, zones) &&
			f.Status.State == v1beta1.FederationStateAvailable {
			return false, nil // Tutto è già allineato
		}

		// 5. Aggiorniamo lo status locale
		f.Status.OfferedAvailabilityZones = zones
		f.Status.State = v1beta1.FederationStateAvailable
		f.Status.FederationContextId = federationContextId

		upErr := r.Status().Update(ctx, f.DeepCopy())
		if upErr != nil {
			log.Error(upErr, ">>> [Federation] Error Updating resource status", "federation", f.Name)
			return false, upErr
		}
	}
	return true, nil
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

func (r *FederationReconciler) handleExternalFederationDeletion(
	ctx context.Context, f *v1beta1.Federation, isRest bool) error {
	log := log.FromContext(ctx)
	if isRest {
		log.Info(">>> [Federation] Deleting external federation")
		res, err := r.GetOPGClient(
			f.Labels[v1beta1.ExternalIdLabel],
			f.Spec.GuestPartnerCredentials.TokenUrl,
			f.Spec.GuestPartnerCredentials.ClientId,
		).DeleteFederationDetailsWithResponse(
			context.TODO(),
			f.Status.FederationContextId,
		)
		if err != nil {
			log.Error(err, errorCreatingFederationMsg)
			return err
		}

		statusCode := res.StatusCode()

		switch {
		case statusCode >= 200 && statusCode < 300:
			log.Info("Deleted")
			// federResponse.OfferedAvailabilityZones
		case statusCode == 400:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		case statusCode == 401:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		case statusCode == 404:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		case statusCode == 409:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		case statusCode == 422:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		case statusCode == 500:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		case statusCode == 503:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		case statusCode == 520:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		default:
			log.Info(unexpectedStatusCodeMsg, "status", statusCode, "body", string(res.Body))
		}
	} else {
		log.Info(">>> [Federation] Deleting external federation via Kubernetes API")
		kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, f.Labels[v1beta1.FederationSecretNameLabel], f.Namespace)
		if err != nil {
			log.Error(err, ">>> [Federation] Error getting kubeconfig from secret")
			return err
		}
		log.Info(">>> [Federation] Building dynamic client with kubeconfig")
		hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, f.Labels[v1beta1.FederationHostOPLabel])
		if err != nil {
			log.Error(err, ">>> [Federation] Error building client with kubeconfig")
			return err
		}
		err = DeleteResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "federations", f.Labels[v1beta1.FederationNamespaceLabel], f.Labels[v1beta1.FederationHostIdLabel])
		if err != nil && !errors.IsNotFound(err) {
			log.Error(err, ">>> [Federation] Failed to delete federation resource from target cluster")
			return err
		}
		log.Info(">>> [Federation] Stopping background watcher for remote host")
		watcher.StopRemoteWatcherFederation(f.Labels[v1beta1.FederationHostIdLabel], f.Name)
	}
	return nil
}

func (r *FederationReconciler) handleAcceptExternalAZ(ctx context.Context, f *v1beta1.Federation, isRest bool) error {
	log := log.FromContext(ctx)
	if isRest {
		if len(f.Status.OfferedAvailabilityZones) == 0 {
			log.Info(">>> [Federation] No AZ was offered, no AZ available to be accepted")
			return nil
		}
		az := f.Status.OfferedAvailabilityZones[0].ZoneId

		fedReq := opgmodels.ZoneRegistrationRequestData{
			AcceptedAvailabilityZones: []opgmodels.ZoneIdentifier{az},
		}

		res, err := r.GetOPGClient(
			f.Labels[v1beta1.ExternalIdLabel],
			f.Spec.GuestPartnerCredentials.TokenUrl,
			f.Spec.GuestPartnerCredentials.ClientId,
		).ZoneSubscribeWithResponse(
			context.TODO(),
			f.Status.FederationContextId,
			fedReq,
		)
		if err != nil {
			log.Error(err, ">>> [Federation] Error accepting AZ")
			return err
		}

		statusCode := res.StatusCode()

		switch {
		case statusCode >= 200 && statusCode < 300:
			log.Info(">>> [Federation] Created", "response", res.JSON200)
			f.Spec.AcceptedAvailabilityZones = []string{az}

			upErr := r.Update(ctx, f.DeepCopy())
			if upErr != nil {
				log.Error(upErr, ">>> [Federation] Error Updating resource", "federation", f.Name)
				return upErr
			}
		case statusCode == 400:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		case statusCode == 401:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		case statusCode == 404:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		case statusCode == 409:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		case statusCode == 422:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		case statusCode == 500:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		case statusCode == 503:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		case statusCode == 520:
			handleProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		default:
			log.Info(unexpectedStatusCodeMsg, "status", statusCode, "body", string(res.Body))
		}
	} else {
		if len(f.Status.OfferedAvailabilityZones) == 0 {
			log.Info(">>> [Federation] No offered AZs discovered from host yet, skipping acceptance for now")
			return nil
		}
		az := f.Status.OfferedAvailabilityZones[0].ZoneId
		if len(f.Spec.AcceptedAvailabilityZones) > 0 && f.Spec.AcceptedAvailabilityZones[0] == az {
			log.Info(">>> [Federation] AZ already accepted locally, skipping patch")
			return nil
		}
		log.Info(">>> [Federation] Accepting AZ in Kubernetes federation via cross-cluster patch", "az", az)
		kubeconfigBytes, err := GetKubeconfigFromSecret(ctx, r.Client, f.Labels[v1beta1.FederationSecretNameLabel], f.Namespace)
		if err != nil {
			log.Error(err, ">>> [Federation] Error getting kubeconfig from secret")
			return err
		}
		log.Info(">>> [Federation] Building dynamic client with kubeconfig")
		hostClient, err := BuildClientWithKubeconfig(kubeconfigBytes, f.Labels[v1beta1.FederationHostOPLabel])
		if err != nil {
			log.Error(err, ">>> [Federation] Error building client with kubeconfig")
			return err
		}
		fedPatch, err := json.Marshal(map[string]interface{}{
			"spec": map[string]interface{}{
				"acceptedAvailabilityZones": []string{az},
			},
		})
		if err != nil {
			log.Error(err, ">>> [Federation] Error marshaling patch")
			return err
		}
		err = PatchResource(ctx, hostClient, "opg.ewbi.nby.one", "v1beta1", "federations", f.Labels[v1beta1.FederationNamespaceLabel], f.Labels[v1beta1.FederationHostIdLabel], k8stypes.PatchType("application/merge-patch+json"), fedPatch)
		if err != nil {
			log.Error(err, ">>> [Federation] Failed to apply patch to target cluster resource")
			return err
		}
		log.Info(">>> [Federation] Successfully patched Federation resource in host cluster")
		f.Spec.AcceptedAvailabilityZones = []string{az}
		if err := r.Update(ctx, f.DeepCopy()); err != nil {
			log.Error(err, ">>> [Federation] Failed to update local spec with accepted AZ")
			return err
		}
	}
	return nil
}
