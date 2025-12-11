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
		latest := &v1beta1.ApplicationInstance{}
		r.Get(ctx, client.ObjectKeyFromObject(&a), latest)
		if isGuest {
			if err := r.handleExternalAppInstDeletion(ctx, latest, feder); err != nil {
				log.Error(err, "error deleting appInst")
				latest.Status.Phase = v1beta1.ApplicationInstancePhaseError
				upErr := r.Status().Update(ctx, latest)
				if upErr != nil {
					log.Error(upErr, errorUpdatingResourceStatusMsg)
				}
				return ctrl.Result{}, err
			}
		}
		// if external appInst is correctly deleted, we can remove the finalizer
		if controllerutil.RemoveFinalizer(&a, v1beta1.ApplicationInstanceFinalizer) {
			log.Info("Removed basic finalizer for appInst")
			if err := r.Update(ctx, latest); err != nil {
				//log.Error(err, "update failed while removing finalizers")
				return ctrl.Result{}, err
			}
			log.Info("removed all finalizers, exiting...")
			return ctrl.Result{}, nil
		}
	}

	// if federation is guest, send OPG API request
	if isGuest {
		latest := &v1beta1.ApplicationInstance{}
		r.Get(ctx, client.ObjectKeyFromObject(&a), latest)
		if err := r.handleExternalAppInstCreation(ctx, latest, feder); err != nil {
			log.Info("error creating appInst")
			latest.Status.Phase = v1beta1.ApplicationInstancePhaseError
			upErr := r.Status().Update(ctx, latest)
			if upErr != nil {
				log.Error(upErr, errorUpdatingResourceStatusMsg)
			}
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
	} else {
		if a.Status.Phase == "" {
			a.Status.Phase = v1beta1.ApplicationInstancePhaseReady
			a.Status.State = "Pending"
			log.Info("Initialized new CR state", "phase", a.Status.Phase, "state", a.Status.State)
		} else {
			log.Info("Existing CR state", "phase", a.Status.Phase, "state", a.Status.State)
		}
		if a.GetDeletionTimestamp().IsZero() {
			upErr := r.Status().Update(ctx, a.DeepCopy())
			if upErr != nil {
				log.Error(upErr, errorUpdatingResourceStatusMsg)
			}
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
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
) error {
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
		return err
	}

	statusCode := res.StatusCode()
	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info("APP INSTANCES - Status code 2xx received from OPG API", "status", statusCode)
		latest := &v1beta1.ApplicationInstance{}
		r.Get(ctx, client.ObjectKeyFromObject(a), latest)
		if !latest.GetDeletionTimestamp().IsZero() {
			// if external application instance is correctly deleted, we can remove the finalizer
			if controllerutil.RemoveFinalizer(latest, v1beta1.ApplicationInstanceFinalizer) {
				log.Info("Removed basic finalizer for ApplicationInstance")
				r.Update(ctx, latest)
				return nil
			}
		}
		latest.Status.Phase = v1beta1.ApplicationInstancePhaseReady
		switch statusCode {
		case 202:
			latest.Status.State = "Pending"
		case 200:
			latest.Status.State = "Ready"
			latest.Status.AppInstanceId = res.JSON200.AppInstanceId
		default:
			latest.Status.State = "Pending"
		}
		log.Info("Created/Updated external application instances", "phase", latest.Status.Phase, "state", latest.Status.State)
		r.Status().Update(ctx, latest)
		time.Sleep(3 * time.Second)
		r.handleExternalAppInstGet(ctx, latest, feder)
		return nil

	case statusCode == 400:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		log.Info("Couldn't be created", "Detail", res.ApplicationproblemJSON400.Detail)
		return errors.New(*res.ApplicationproblemJSON400.Detail)
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
			return errors.New(*res.ApplicationproblemJSON500.Detail)
		}
	case statusCode == 503:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
	case statusCode == 520:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
	default:
		latest := &v1beta1.ApplicationInstance{}
		r.Get(ctx, client.ObjectKeyFromObject(a), latest)
		latest.Status.Phase = v1beta1.ApplicationInstancePhaseError
		latest.Status.State = "Error"
		r.Status().Update(ctx, latest)
	}
	return nil
}

func (r *ApplicationInstanceReconciler) handleExternalAppInstGet(
	ctx context.Context, a *v1beta1.ApplicationInstance, feder *v1beta1.Federation,
) error {
	log := log.FromContext(ctx)
	log.Info("Getting external appInst details")
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
		return err
	}
	statusCode := res.StatusCode()

	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info("APP INSTANCES - Status code 2xx received from OPG API", "status", statusCode)
		latest := &v1beta1.ApplicationInstance{}
		r.Get(ctx, client.ObjectKeyFromObject(a), latest)
		if !latest.GetDeletionTimestamp().IsZero() {
			// if external application instance is correctly deleted, we can remove the finalizer
			if controllerutil.RemoveFinalizer(latest, v1beta1.ApplicationInstanceFinalizer) {
				log.Info("Removed basic finalizer for ApplicationInstance")
				r.Update(ctx, latest)
				return nil
			}
		}
		latest.Status.State = v1beta1.ApplicationInstanceState(*res.JSON200.AppInstanceState)
		if len(res.JSON200.AccessPointInfo) > 0 {
			log.Info("APP INSTANCES - Updating Access Point Info in status")
			latest.Status.AccessPointInfo = []v1beta1.AccessPointInfo{}
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
						Ipv6Addresses: ap.Ipv6Addresses.([]string),
					})
				}
				latest.Status.AccessPointInfo = append(latest.Status.AccessPointInfo, apInfo)
			}
			log.Info("APP INSTANCES - Access Point Info updated in status", "info", latest.Status.AccessPointInfo)
			if err := r.Status().Update(ctx, latest); err != nil {
				log.Error(err, " error updating appInst status")
				return err
			}
		}
		time.Sleep(5 * time.Second)
		r.handleExternalAppInstGet(ctx, latest, feder)

	case statusCode == 400:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		log.Info("Couldn't be created", "Detail", res.ApplicationproblemJSON400.Detail)
		return errors.New(*res.ApplicationproblemJSON400.Detail)
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
			return errors.New(*res.ApplicationproblemJSON500.Detail)
		}
	case statusCode == 503:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
	case statusCode == 520:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
	default:
		latest := &v1beta1.ApplicationInstance{}
		r.Get(ctx, client.ObjectKeyFromObject(a), latest)
		latest.Status.Phase = v1beta1.ApplicationInstancePhaseError
		latest.Status.State = "Error"
		r.Status().Update(ctx, latest)
	}
	return nil
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
