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

package rest

import (
	"context"

	"github.com/go-logr/logr"
	opgmodels "github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// FederationReconciler reconciles a Federation object
type FederationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

func handleFederationProblemDetails(log logr.Logger, code int, p *opgmodels.ProblemDetails) {
	log.Info(">>> [Federation] Response with error", "error", code, "details", p)
}

const (
	errorCreatingFederationMsg    = ">>> [Federation][REST] Error Creating resource status"
	unexpectedStatusFederationMsg = ">>> [Federation][REST] Unexpected Status Code"
)

func (r *FederationReconciler) CreateFederation(ctx context.Context, f *v1beta1.Federation) (statusChanged bool, err error) {
	log := log.FromContext(ctx)
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
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
	case statusCode == 401:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
	case statusCode == 404:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
	case statusCode == 409:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
	case statusCode == 422:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
	case statusCode == 500:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
	case statusCode == 503:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
	case statusCode == 520:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
	default:
		log.Info(unexpectedStatusFederationMsg, "status", statusCode, "body", string(res.Body))
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
func (r *FederationReconciler) AcceptExternalAZ(ctx context.Context, f *v1beta1.Federation) error {
	log := log.FromContext(ctx)
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
		log.Info(">>> [Federation][REST] Created", "response", res.JSON200)
		f.Spec.AcceptedAvailabilityZones = []string{az}

		upErr := r.Update(ctx, f.DeepCopy())
		if upErr != nil {
			log.Error(upErr, ">>> [Federation][REST] Error Updating resource", "federation", f.Name)
			return upErr
		}
	case statusCode == 400:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
	case statusCode == 401:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
	case statusCode == 404:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
	case statusCode == 409:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
	case statusCode == 422:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
	case statusCode == 500:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
	case statusCode == 503:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
	case statusCode == 520:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
	default:
		log.Info(unexpectedStatusFederationMsg, "status", statusCode, "body", string(res.Body))
	}
	return nil
}

func (r *FederationReconciler) DeleteFederation(ctx context.Context, f *v1beta1.Federation) error {
	log := log.FromContext(ctx)
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
		log.Error(err, ">>> [Federation][REST] Error deleting external federation")
		return err
	}

	statusCode := res.StatusCode()

	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info(">>> [Federation][REST] Deleted federation successfully")
		// federResponse.OfferedAvailabilityZones
	case statusCode == 400:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
	case statusCode == 401:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
	case statusCode == 404:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
	case statusCode == 409:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
	case statusCode == 422:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
	case statusCode == 500:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
	case statusCode == 503:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
	case statusCode == 520:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
	default:
		log.Info(unexpectedStatusFederationMsg, "status", statusCode, "body", string(res.Body))
	}
	return nil
}
