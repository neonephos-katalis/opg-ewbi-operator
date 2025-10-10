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

	opgmodels "github.com/nbycomp/neonephos-opg-ewbi-api/api/federation/models"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/nbycomp/neonephos-opg-ewbi-operator/api/v1beta1"
	"github.com/nbycomp/neonephos-opg-ewbi-operator/internal/opg"
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

	log.Info(
		"starting reconcile function for federation",
	)
	defer log.Info("end reconcile for federation")

	// Getting main federation or requeue
	var f v1beta1.Federation
	if err := r.Get(ctx, req.NamespacedName, &f); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("federation object not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "error getting federation object")
		return ctrl.Result{}, err
	}
	log.Info("Federation object obtained", "name", f.Name, "originOP", f.Spec.OriginOP)
	isGuest := IsGuestResource(f.Labels)
	if f.GetDeletionTimestamp().IsZero() {
		if controllerutil.AddFinalizer(&f, v1beta1.FederationFinalizer) {
			log.Info("Added finalizer to Federation")
			if err := r.Update(ctx, f.DeepCopy()); err != nil {
				log.Info("unable to Update Federation with finalizer")
				return ctrl.Result{}, err
			}
			log.Info("Successfully added finalizer to federation")
			return ctrl.Result{}, nil
		}
	} else {
		if isGuest {
			if err := r.handleExternalFederationDeletion(ctx, &f); err != nil {
				log.Error(err, "error deleting federation")
				f.Status.Phase = v1beta1.FederationPhaseError
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
		updated, err := r.handleExternalFederationCreation(ctx, &f)
		if err != nil {
			log.Error(err, errorCreatingFederationMsg)
			f.Status.Phase = v1beta1.FederationPhaseError
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

		if err := r.handleAcceptExternalAZ(ctx, &f); err != nil {
			log.Error(err, "error accepting az federation")
			f.Status.Phase = v1beta1.FederationPhaseError
			upErr := r.Status().Update(ctx, f.DeepCopy())
			if upErr != nil {
				log.Error(upErr, errorUpdatingResourceStatusMsg)
			}
			return ctrl.Result{}, err
		}
	} else {
		f.Status.Phase = v1beta1.FederationPhaseReady
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
		Complete(r)
}

func (r *FederationReconciler) handleExternalFederationCreation(
	ctx context.Context, f *v1beta1.Federation) (statusChanged bool, err error) {
	log := log.FromContext(ctx)
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
	ctx context.Context, f *v1beta1.Federation,
) error {
	log := log.FromContext(ctx)
	log.Info("Deleting external federation")
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
	return nil
}

func (r *FederationReconciler) handleAcceptExternalAZ(ctx context.Context, f *v1beta1.Federation) error {
	log := log.FromContext(ctx)

	if len(f.Status.OfferedAvailabilityZones) != 1 {
		log.Info("No AZ was offered, no AZ available to be accepted")
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
		log.Error(err, "error accepting AZ")
		return err
	}

	statusCode := res.StatusCode()

	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info("Created", "response", res.JSON200)
		f.Spec.AcceptedAvailabilityZones = []string{az}

		upErr := r.Update(ctx, f.DeepCopy())
		if upErr != nil {
			log.Error(upErr, "Error Updating resource", "federation", f.Name)
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
	return nil
}
