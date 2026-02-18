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
	"reflect"
	"time"

	opgmodels "github.com/neonephos-katalis/opg-ewbi-api/api/federation/models"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/v1beta1"
	opgewbiv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
)

// ApplicationInstanceReconciler reconciles a ApplicationInstance object
type ApplicationInstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=applicationinstances,verbs=*,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=applicationinstances/status,verbs=get;update;patch,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=applicationinstances/finalizers,verbs=update,namespace=foo

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Modify the Reconcile function to compare the state specified by
// the ApplicationInstance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *ApplicationInstanceReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("name", req.Name, "namespace", req.Namespace)
	log.Info("starting reconcile function for appInst")
	defer log.Info("end reconcile for appInst")

	// Getting main appInst or requeue
	var a v1beta1.ApplicationInstance
	if err := r.Get(ctx, req.NamespacedName, &a); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("appInst object not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "error getting appInst object")
		return ctrl.Result{}, err
	}

	// Getting appInst's federation or requeue by using federation-context-id label
	// extraLabels := map[string]string{v1beta1.FederationRelationLabel: a.Labels[v1beta1.FederationRelationLabel]}
	isGuest := IsGuestResource(a.Labels)
	extraLabels := map[string]string{}
	if isGuest {
		extraLabels[v1beta1.FederationRelationLabel] = string(v1beta1.FederationRelationGuest)
	} else {
		extraLabels[v1beta1.FederationRelationLabel] = string(v1beta1.FederationRelationHost)
	}
	feder, err := GetFederationByContextId(ctx, r.Client, a.Labels[v1beta1.FederationContextIdLabel], extraLabels)
	if err != nil {
		log.Error(err, "An ApplicattionInstance should always have a parent federation")
		a.Status.Phase = v1beta1.ApplicationInstancePhaseError
		upErr := r.Status().Update(ctx, a.DeepCopy())
		if upErr != nil {
			log.Error(upErr, errorUpdatingResourceStatusMsg)
		}
		return ctrl.Result{}, err
	}

	log.Info("Federation object obtained", "name", feder.Name)

	if a.GetDeletionTimestamp().IsZero() {
		if controllerutil.AddFinalizer(&a, v1beta1.ApplicationInstanceFinalizer) {
			log.Info("Added finalizer to appInst")
			if err := r.Update(ctx, a.DeepCopy()); err != nil {
				log.Info("unable to Update appInst with finalizer")
				return ctrl.Result{}, err
			}
			log.Info("Successfully added finalizer to appInst")
			return ctrl.Result{}, nil
		}
	} else {
		if isGuest {
			if err := r.handleExternalAppInstDeletion(ctx, &a, feder); err != nil {
				log.Error(err, "error deleting appInst")
				a.Status.Phase = v1beta1.ApplicationInstancePhaseError
				upErr := r.Status().Update(ctx, a.DeepCopy())
				if upErr != nil {
					log.Error(upErr, errorUpdatingResourceStatusMsg)
				}
				return ctrl.Result{}, err
			}
		}
		// if external appInst is correctly deleted, we can remove the finalizer
		if controllerutil.RemoveFinalizer(&a, v1beta1.ApplicationInstanceFinalizer) {
			log.Info("Removed basic finalizer for appInst")
			if err := r.Update(ctx, a.DeepCopy()); err != nil {
				//log.Error(err, "update failed while removing finalizers")
				return ctrl.Result{}, err
			}
			log.Info("removed all finalizers, exiting...")
			return ctrl.Result{}, nil
		}
	}

	// if federation is guest, send OPG API request to create the resource on Host
	// NOTE: No polling - callbacks from Host will update status via Federation API server handler
	if isGuest {
		// Only handle creation - callbacks will handle status updates
		if a.Status.AppInstanceId == "" {
			if result, err := r.handleExternalAppInstCreation(ctx, &a, feder); err != nil {
				log.Info("error creating appInst")
				a.Status.Phase = v1beta1.ApplicationInstancePhaseError
				upErr := r.Status().Update(ctx, a.DeepCopy())
				if upErr != nil {
					log.Error(upErr, errorUpdatingResourceStatusMsg)
				}
				return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
			} else {
				return result, nil
			}
		}
		// Resource already created - no polling needed
		// Host will send callbacks via Federation API to update status
		log.Info("AppInst already created, waiting for callbacks from Host", "appInstanceId", a.Status.AppInstanceId)
		return ctrl.Result{}, nil
	} else {
		// HOST side: manage local CR and send callbacks to Guest on status changes
		if a.Status.Phase == "" {
			a.Status.Phase = v1beta1.ApplicationInstancePhaseReady
			a.Status.State = "Pending"
			a.Status.AppInstanceId = a.Labels[v1beta1.ExternalIdLabel]
			log.Info("-<0>- Initialized new CR state", "phase", a.Status.Phase, "state", a.Status.State, "appInstanceId", a.Status.AppInstanceId)
		} else {
			log.Info("-<1>- Existing CR state", "phase", a.Status.Phase, "state", a.Status.State)
		}
		
		log.Info("-<2>- Checking delation timestamp")
		if a.GetDeletionTimestamp().IsZero() {
			log.Info("-<3>- Updatting resource")
			upErr := r.Status().Update(ctx, a.DeepCopy())
			if upErr != nil {
				log.Error(upErr, errorUpdatingResourceStatusMsg)
			}

			log.Info("-<4>- Sending callback to", "federation", feder)
			// Send callback to Guest (event-driven, triggered on every reconciliation)
			// For ApplicationInstance: continue callbacks until resource is deleted
			if err := r.sendAppInstCallback(ctx, &a, feder); err != nil {
				log.Error(err, "failed to send callback to Guest")
				// Don't fail reconciliation - callback is best-effort
			}
		}
		return ctrl.Result{}, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&opgewbiv1beta1.ApplicationInstance{}).
		Named("applicationinstance").
		Complete(r)
}

func (r *ApplicationInstanceReconciler) handleExternalAppInstCreation(
	ctx context.Context, a *v1beta1.ApplicationInstance, feder *v1beta1.Federation,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	zone := struct {
		FlavourId           string                                                   `json:"flavourId"`
		ResPool             *string                                                  `json:"resPool,omitempty"`
		ResourceConsumption *opgmodels.InstallAppJSONBodyZoneInfoResourceConsumption `json:"resourceConsumption,omitempty"`
		ZoneId              string                                                   `json:"zoneId"`
	}{
		FlavourId:           a.Spec.ZoneInfo.FlavourId,
		ResPool:             &a.Spec.ZoneInfo.ResPool,
		ResourceConsumption: (*opgmodels.InstallAppJSONBodyZoneInfoResourceConsumption)(&a.Spec.ZoneInfo.ResourceConsumption),
		ZoneId:              a.Spec.ZoneInfo.ZoneId,
	}

	reqBody := opgmodels.InstallAppJSONRequestBody{
		AppId:               a.Spec.AppId,
		AppInstCallbackLink: a.Spec.CallbBackLink,
		AppInstanceId:       a.Labels[v1beta1.ExternalIdLabel],
		AppProviderId:       a.Spec.AppProviderId,
		AppVersion:          a.Spec.AppVersion,
		ZoneInfo:            zone,
	}

	res, err := r.GetOPGClient(
		feder.Labels[v1beta1.ExternalIdLabel],
		feder.Spec.GuestPartnerCredentials.TokenUrl,
		feder.Spec.GuestPartnerCredentials.ClientId,
	).InstallAppWithResponse(
		context.TODO(),
		feder.Status.FederationContextId,
		reqBody)

	if err != nil {
		log.Error(err, "error creating appInst")
		return ctrl.Result{}, err
	}

	statusCode := res.StatusCode()
	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info("APP INSTANCES - Status code 2xx received from OPG API", "status", statusCode)
		a.Status.Phase = v1beta1.ApplicationInstancePhaseReady
		a.Status.State = "Pending"
		log.Info("******************************", "JSON200", res.JSON200)
		a.Status.AppInstanceId = a.Name //"app-inst-2dae064c-28cc-456e-8b0a-dd67bab7d8f7"
		log.Info("Created/Updated external application instances", "phase", a.Status.Phase, "state", a.Status.State, "appInstanceId", a.Status.AppInstanceId)
		r.Status().Update(ctx, a)
		// No RequeueAfter - callbacks from Host will update status via Federation API server handler
		return ctrl.Result{}, nil
	case statusCode == 400:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		log.Info("Couldn't be created", "Detail", res.ApplicationproblemJSON400.Detail)
		return ctrl.Result{}, errors.New(*res.ApplicationproblemJSON400.Detail)
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
		// this should be deleted when API returns a 400 for this case
		if *res.ApplicationproblemJSON500.Detail == "application not found" {
			return ctrl.Result{}, errors.New(*res.ApplicationproblemJSON500.Detail)
		}
	case statusCode == 503:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
	case statusCode == 520:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
	default:
		a.Status.Phase = v1beta1.ApplicationInstancePhaseError
		a.Status.State = "Error"
		r.Status().Update(ctx, a)
	}
	return ctrl.Result{}, nil
}

// handleExternalAppInstGet retrieves external AppInst details on-demand.
// NOTE: This function is no longer used for polling - status updates come via callbacks.
// Can still be called for on-demand queries when needed.
func (r *ApplicationInstanceReconciler) handleExternalAppInstGet(
	ctx context.Context, a *v1beta1.ApplicationInstance, feder *v1beta1.Federation,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("DEPRECATED: Getting external appInst details - use callbacks instead")
	res, err := r.GetOPGClient(
		feder.Labels[v1beta1.ExternalIdLabel],
		feder.Spec.GuestPartnerCredentials.TokenUrl,
		feder.Spec.GuestPartnerCredentials.ClientId,
	).GetAppInstanceDetailsWithResponse(
		context.TODO(),
		feder.Status.FederationContextId,
		a.Spec.AppId,
		a.Status.AppInstanceId,
		a.Spec.ZoneInfo.ZoneId,
	)
	if err != nil {
		log.Error(err, "error getting appInst info")
		return ctrl.Result{}, err
	}
	statusCode := res.StatusCode()

	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info("********APP INSTANCES - ACCESS POINT INFO - Status code 2xx received from OPG API", "status", statusCode)
		if res.JSON200.AppInstanceState != nil {
			oldStatus := a.Status.DeepCopy()
			newState := v1beta1.ApplicationInstanceState(*res.JSON200.AppInstanceState)
			a.Status.State = newState
			log.Info("°°°°°°°°°°°°°°°°°°°°APP INSTANCES", "JSON200", res.JSON200)
			if res.JSON200.AppInstanceState != nil {
				log.Info("APP INSTANCES - Updating Access Point Info in status")
				newAccessPoints := []v1beta1.AccessPointInfo{}
				for _, info := range res.JSON200.AccessPointInfo {
					apInfo := v1beta1.AccessPointInfo{
						InterfaceId:  info.InterfaceId,
						AccessPoints: []v1beta1.AccessPoints{},
					}
					for _, ap := range info.AccessPoints {
						apInfo.AccessPoints = append(apInfo.AccessPoints, v1beta1.AccessPoints{
							Port:          int(ap.Port),
							Fqdn:          ap.Fqdn,
							Ipv4Addresses: ap.Ipv4Addresses,
							Ipv6Addresses: ap.Ipv6Addresses,
						})
					}
					newAccessPoints = append(newAccessPoints, apInfo)
				}
				a.Status.AccessPointInfo = newAccessPoints
				if !reflect.DeepEqual(oldStatus.AccessPointInfo, a.Status.AccessPointInfo) || oldStatus.State != a.Status.State {
					log.Info("APP INSTANCES - Status changed, updating resource")
					if err := r.Status().Update(ctx, a); err != nil {
						log.Error(err, "error updating appInst status")
						return ctrl.Result{}, err
					}
				} else {
					log.Info("APP INSTANCES - Status unchanged, skipping update")
				}
				return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
			} else {
				return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
			}
		} else {
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
	case statusCode == 400:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		log.Info("Couldn't be created", "Detail", res.ApplicationproblemJSON400.Detail)
		return ctrl.Result{}, errors.New(*res.ApplicationproblemJSON400.Detail)
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
		// this should be deleted when API returns a 400 for this case
		if *res.ApplicationproblemJSON500.Detail == "application not found" {
			return ctrl.Result{}, errors.New(*res.ApplicationproblemJSON500.Detail)
		}
	case statusCode == 503:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
	case statusCode == 520:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
	default:
		a.Status.Phase = v1beta1.ApplicationInstancePhaseError
		a.Status.State = "Error"
		r.Status().Update(ctx, a)
	}
	return ctrl.Result{}, nil
}

func (r *ApplicationInstanceReconciler) handleExternalAppInstDeletion(
	ctx context.Context, appInst *v1beta1.ApplicationInstance, feder *v1beta1.Federation,
) error {
	log := log.FromContext(ctx)
	log.Info("Deleting external appInst")
	// we should delete the appInst
	res, err := r.GetOPGClient(
		feder.Labels[v1beta1.ExternalIdLabel],
		feder.Spec.GuestPartnerCredentials.TokenUrl,
		feder.Spec.GuestPartnerCredentials.ClientId,
	).RemoveAppWithResponse(
		context.TODO(),
		feder.Status.FederationContextId,
		appInst.Spec.AppId,
		appInst.Labels[v1beta1.ExternalIdLabel],
		appInst.Spec.ZoneInfo.ZoneId,
	)
	if err != nil {
		log.Error(err, "error deleting federation")
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
	return nil
}

// sendAppInstCallback sends a callback to the Guest with the current ApplicationInstance status.
// This is called by the Host reconciler when status changes (event-driven, not polling).
// For ApplicationInstance: callbacks continue until the resource is deleted.
func (r *ApplicationInstanceReconciler) sendAppInstCallback(
	ctx context.Context,
	a *v1beta1.ApplicationInstance,
	feder *v1beta1.Federation,
) error {
	log := log.FromContext(ctx)

	// Check if callback is configured
	if feder.Spec.Partner.StatusLink == "" {
		log.Info("No callback StatusLink configured in Federation, skipping callback")
		return nil
	}

	log.Info("Sending AppInst callback to Guest",
		"appInstanceId", a.Status.AppInstanceId,
		"state", a.Status.State,
		"statusLink", feder.Spec.Partner.StatusLink)

	// Build callback body with current status
	// AppInstCallbackLinkJSONRequestBody requires: AppId, AppInstanceId, AppInstanceInfo, ZoneId
	state := opgmodels.InstanceState(a.Status.State)
	accessPointInfo := r.convertAccessPointInfoToOPGSingle(a.Status.AccessPointInfo)

	labels := a.GetLabels()

	callbackBody := opgmodels.AppInstCallbackLinkJSONRequestBody{
		AppId:               a.Spec.AppId,
		AppInstanceId:       labels["opg.ewbi.nby.one/id"]
		FederationContextId: &feder.Status.FederationContextId,
		ZoneId:              a.Spec.ZoneInfo.ZoneId,
	}
	callbackBody.AppInstanceInfo.AppInstanceState = &state
	callbackBody.AppInstanceInfo.AccesspointInfo = accessPointInfo

	log.Info("*********************************************************************************************", "AppInstanceState", state)
	// Get callback client (pointing to Guest's callback URL)
	// Using a different cache key to separate callback client from regular client
	callbackClient := r.GetOPGClient(
		feder.Labels[v1beta1.ExternalIdLabel]+"-callback",
		feder.Spec.Partner.StatusLink,
		feder.Spec.Partner.CallbackCredentials.ClientId,
	)

	// Send callback to Guest
	log.Info("#############################################################################################", "callbackbody", callbackBody)
	res, err := callbackClient.AppInstCallbackLinkWithResponse(
		ctx,
		feder.Spec.Partner.CallbackCredentials.ClientId,
		callbackBody,
	)
	log.Info("#############################################################################################", "res", res.StatusCode())
	if err != nil {
		return err
	}

	statusCode := res.StatusCode()
	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info("Successfully sent AppInst callback to Guest", "status", statusCode)
	case statusCode == 400:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
	case statusCode == 401:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
	case statusCode == 404:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
	default:
		log.Info("Callback returned unexpected status", "status", statusCode, "body", string(res.Body))
	}

	return nil
}

// convertAccessPointInfoToOPGSingle converts v1beta1.AccessPointInfo slice to a single *opgmodels.AccessPointInfo
// For callbacks, we take the first AccessPointInfo if available.
func (r *ApplicationInstanceReconciler) convertAccessPointInfoToOPGSingle(
	apiInfoList []v1beta1.AccessPointInfo,
) *opgmodels.AccessPointInfo {
	if len(apiInfoList) == 0 {
		return nil
	}

	// Take the first AccessPointInfo for the callback
	api := apiInfoList[0]
	accessPoints := []opgmodels.AccessPoints{}
	for _, ap := range api.AccessPoints {
		accessPoints = append(accessPoints, opgmodels.AccessPoints{
			Port:          ap.Port,
			Fqdn:          ap.Fqdn,
			Ipv4Addresses: ap.Ipv4Addresses,
			Ipv6Addresses: ap.Ipv6Addresses,
		})
	}

	return &opgmodels.AccessPointInfo{
		InterfaceId:  api.InterfaceId,
		AccessPoints: accessPoints,
	}
}

// convertAccessPointInfoToOPG converts v1beta1.AccessPointInfo slice to opgmodels.AccessPointInfo slice
// Kept for backward compatibility if needed for other use cases.
func (r *ApplicationInstanceReconciler) convertAccessPointInfoToOPG(
	apiInfoList []v1beta1.AccessPointInfo,
) []opgmodels.AccessPointInfo {
	result := []opgmodels.AccessPointInfo{}
	for _, api := range apiInfoList {
		accessPoints := []opgmodels.AccessPoints{}
		for _, ap := range api.AccessPoints {
			accessPoints = append(accessPoints, opgmodels.AccessPoints{
				Port:          ap.Port,
				Fqdn:          ap.Fqdn,
				Ipv4Addresses: ap.Ipv4Addresses,
				Ipv6Addresses: ap.Ipv6Addresses,
			})
		}
		result = append(result, opgmodels.AccessPointInfo{
			InterfaceId:  api.InterfaceId,
			AccessPoints: accessPoints,
		})
	}
	return result
}
