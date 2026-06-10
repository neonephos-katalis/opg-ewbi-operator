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
	"errors"

	"github.com/go-logr/logr"
	opgmodels "github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ApplicationInstanceReconciler reconciles an Application object
type ApplicationInstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

func handleApplicationInstanceProblemDetails(log logr.Logger, code int, p *opgmodels.ProblemDetails) {
	log.Info(">>> [AppInst][REST] Response with error", "error", code, "details", p)
}

const (
	errorUpdatingApplicationInstanceStatusMsg = ">>> [AppInst][REST] Error Updating resource status"
	unexpectedStatusApplicationInstanceMsg    = ">>> [AppInst][REST] Unexpected Status Code"
)

func (r *ApplicationInstanceReconciler) CreateApplicationInstance(ctx context.Context, a *v1beta1.ApplicationInstance, feder *v1beta1.Federation) error {
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
		log.Error(err, ">>> [AppInst][REST] Error creating appInst")
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
		return err
	}
	statusCode := res.StatusCode()
	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info(">>> [AppInst][REST] Status code 2xx received from OPG API", "status", statusCode)

		a.Status.State = v1beta1.ApplicationInstanceStatePending
		a.Status.AppInstanceId = a.Name //"app-inst-2dae064c-28cc-456e-8b0a-dd67bab7d8f7"
		log.Info(">>> [AppInst][REST] Created external application instances", "state", a.Status.State, "appInstanceId", a.Status.AppInstanceId)

	case statusCode == 400:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		log.Info(">>> [AppInst][REST] Couldn't be created", "Detail", res.ApplicationproblemJSON400.Detail)
	case statusCode == 401:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
	case statusCode == 404:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
	case statusCode == 409:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
	case statusCode == 422:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
	case statusCode == 500:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
		// this should be deleted when API returns a 400 for this case
		if *res.ApplicationproblemJSON500.Detail == "application not found" {
			return errors.New(*res.ApplicationproblemJSON500.Detail)
		}
	case statusCode == 503:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
	case statusCode == 520:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
	default:
		a.Status.State = v1beta1.ApplicationInstanceStatePending
	}
	upErr := r.Status().Update(ctx, a)
	if upErr != nil {
		log.Error(upErr, errorUpdatingApplicationInstanceStatusMsg)
		return upErr
	}
	return nil
}

func (r *ApplicationInstanceReconciler) DeleteApplicationInstance(ctx context.Context, a *v1beta1.ApplicationInstance, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [AppInst][REST] Deleting external appInst")
	// we should delete the appInst
	res, err := r.GetOPGClient(
		feder.Labels[v1beta1.ExternalIdLabel],
		feder.Spec.GuestPartnerCredentials.TokenUrl,
		feder.Spec.GuestPartnerCredentials.ClientId,
	).RemoveAppWithResponse(
		context.TODO(),
		feder.Status.FederationContextId,
		a.Spec.AppId,
		a.Labels[v1beta1.ExternalIdLabel],
		a.Spec.ZoneInfo.ZoneId,
	)
	if err != nil {
		log.Error(err, ">>> [AppInst][REST] Error deleting external appInst")
		return err
	}

	statusCode := res.StatusCode()

	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info(">>> [AppInst][REST] Deleted external appInst")
		a.Status.State = v1beta1.ApplicationInstanceStateTerminating
		// federResponse.OfferedAvailabilityZones
	case statusCode == 400:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
	case statusCode == 401:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
	case statusCode == 404:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
	case statusCode == 409:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
	case statusCode == 422:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
	case statusCode == 500:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
	case statusCode == 503:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
	case statusCode == 520:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
	default:
		log.Info(unexpectedStatusApplicationInstanceMsg, "status", statusCode, "body", string(res.Body))
		a.Status.State = v1beta1.ApplicationInstanceStatePending
	}
	upErr := r.Status().Update(ctx, a)
	if upErr != nil {
		log.Error(upErr, errorUpdatingApplicationInstanceStatusMsg)
		return upErr
	}
	return nil
}

func (r *ApplicationInstanceReconciler) CallbackApplicationInstance(ctx context.Context, a *v1beta1.ApplicationInstance, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)

	// Check if callback is configured
	if feder.Spec.Partner.StatusLink == "" {
		log.Info(">>> [AppInst][REST] No callback StatusLink configured in Federation, skipping callback")
		return nil
	}

	log.Info(">>> [AppInst][REST] Sending AppInst callback to Guest",
		"appInstanceId", a.Status.AppInstanceId,
		"state", a.Status.State,
		"statusLink", feder.Spec.Partner.StatusLink)
	// Build callback body with current status
	// AppInstCallbackLinkJSONRequestBody requires: AppId, AppInstanceId, AppInstanceInfo, ZoneId
	state := opgmodels.InstanceState(a.Status.State)
	callbackBody := opgmodels.AppInstCallbackLinkJSONRequestBody{
		AppId:         a.Spec.AppId,
		AppInstanceId: a.Labels[v1beta1.ExternalIdLabel],
		ZoneId:        a.Spec.ZoneInfo.ZoneId,
	}
	callbackBody.AppInstanceInfo.AppInstanceState = &state
	accessPointInfo := opgmodels.AccessPointInfo{}
	if len(a.Status.AccessPointInfo) > 0 {
		for _, ap := range a.Status.AccessPointInfo {
			endpoint := opgmodels.ServiceEndpoint{
				Port: ap.AccessPoints.Port,
			}
			if ap.AccessPoints.Fqdn != "" {
				fqdn := opgmodels.Fqdn(ap.AccessPoints.Fqdn)
				endpoint.Fqdn = &fqdn
			}
			if len(ap.AccessPoints.Ipv4Addresses) > 0 {
				ipv4List := make([]opgmodels.Ipv4Addr, len(ap.AccessPoints.Ipv4Addresses))
				for i, addr := range ap.AccessPoints.Ipv4Addresses {
					ipv4List[i] = opgmodels.Ipv4Addr(addr)
				}
				endpoint.Ipv4Addresses = &ipv4List
			}
			if len(ap.AccessPoints.Ipv6Addresses) > 0 {
				ipv6List := make([]opgmodels.Ipv6Addr, len(ap.AccessPoints.Ipv6Addresses))
				for i, addr := range ap.AccessPoints.Ipv6Addresses {
					ipv6List[i] = opgmodels.Ipv6Addr(addr)
				}
				endpoint.Ipv6Addresses = &ipv6List
			}
			accessPointInfo = append(accessPointInfo, struct {
				AccessPoints opgmodels.ServiceEndpoint `json:"accessPoints"`
				InterfaceId  opgmodels.InterfaceId     `json:"interfaceId"`
			}{
				AccessPoints: endpoint,
				InterfaceId:  opgmodels.InterfaceId(ap.InterfaceId),
			})
		}
		callbackBody.AppInstanceInfo.AccesspointInfo = &accessPointInfo
	}
	// Get callback client (pointing to Guest's callback URL)
	// Using a different cache key to separate callback client from regular client
	res, err := r.GetOPGClient(
		feder.Labels[v1beta1.ExternalIdLabel],
		feder.Spec.Partner.StatusLink,
		feder.Spec.Partner.CallbackCredentials.ClientId,
	).AppInstCallbackLinkWithResponse(
		context.TODO(),
		feder.Spec.Partner.CallbackCredentials.ClientId,
		callbackBody,
	)
	if err != nil {
		log.Error(err, ">>> [AppInst][REST] Error while sending applicationinstance callback")
		return err
	}

	statusCode := res.StatusCode()
	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info(">>> [AppInst][REST] Successfully sent ApplicationInstance callback to Guest", "status", statusCode)
	case statusCode == 400:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
	case statusCode == 401:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
	case statusCode == 404:
		handleApplicationInstanceProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
	default:
		log.Info(">>> [AppInst][REST] ApplicationInstance callback returned unexpected status", "status", statusCode, "body", string(res.Body))
		a.Status.State = v1beta1.ApplicationInstanceStatePending
	}
	upErr := r.Status().Update(ctx, a)
	if upErr != nil {
		log.Error(upErr, errorUpdatingApplicationInstanceStatusMsg)
		return upErr
	}
	return nil
}
