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

// ApplicationDeploymentReconciler reconciles an ApplicationDeployment object
type ApplicationDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

func handleApplicationDeploymentProblemDetails(log logr.Logger, code int, p *opgmodels.ProblemDetails) {
	log.Info(">>> [AppDep][REST] Response with error", "error", code, "details", p)
}

func (r *ApplicationDeploymentReconciler) CreateApplicationDeployment(ctx context.Context, a *v1beta1.ApplicationDeployment, fed *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	zone := struct {
		FlavourId           string                                                   `json:"flavourId"`
		ResPool             *string                                                  `json:"resPool,omitempty"`
		ResourceConsumption *opgmodels.InstallAppJSONBodyZoneInfoResourceConsumption `json:"resourceConsumption,omitempty"`
		ZoneId              string                                                   `json:"zoneId"`
	}{
		FlavourId:           a.Spec.AppDetails.ZoneInfo.FlavourId,
		ResPool:             &a.Spec.AppDetails.ZoneInfo.ResPool,
		ResourceConsumption: (*opgmodels.InstallAppJSONBodyZoneInfoResourceConsumption)(&a.Spec.AppDetails.ZoneInfo.ResourceConsumption),
		ZoneId:              a.Spec.ZoneId,
	}

	reqBody := opgmodels.InstallAppJSONRequestBody{
		AppId:               a.Spec.AppId,
		AppInstCallbackLink: a.Spec.AppDetails.AppInstCallbackLink,
		AppInstanceId:       &a.Spec.AppInstanceId,
		AppProviderId:       a.Spec.AppProviderId,
		AppVersion:          a.Spec.AppDetails.AppVersion,
		ZoneInfo:            zone,
	}
	params := opgmodels.InstallAppParams{
		IdempotencyKey: "1",
	}
	res, err := r.GetOPGClient(
		fed.Status.FederationContextId,
		fed.Spec.FederationData.RestOptions.TokenUrl,
		fed.Spec.FederationData.ClientId,
	).InstallAppWithResponse(
		context.TODO(),
		fed.Status.FederationContextId,
		&params,
		reqBody)

	if err != nil {
		log.Error(err, ">>> [AppInst][REST] Error creating appInst")
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
		return err
	}
	statusCode := res.StatusCode()
	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info(">>> [AppDep][REST] Status code 2xx received from OPG API", "status", statusCode)

		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStatePending
		log.Info(">>> [AppDep][REST] Created external application instances", "state", a.Status.AppInstanceInfo.AppInstanceState, "appInstanceId", a.Spec.AppInstanceId)

	case statusCode == 400:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		log.Info(">>> [AppDep][REST] Couldn't be created", "Detail", res.ApplicationproblemJSON400.Detail)
	case statusCode == 401:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
	case statusCode == 404:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
	case statusCode == 409:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
	case statusCode == 422:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
	case statusCode == 500:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
		// this should be deleted when API returns a 400 for this case
		if *res.ApplicationproblemJSON500.Detail == "application not found" {
			return errors.New(*res.ApplicationproblemJSON500.Detail)
		}
	case statusCode == 503:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
	case statusCode == 520:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
	default:
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStatePending
	}
	upErr := r.Status().Update(ctx, a)
	if upErr != nil {
		log.Error(upErr, ">>> [AppDep][REST] UNEXPECTED ERROR updating application deployment status", "name", a.Name, "namespace", a.Namespace, "state", a.Status.AppInstanceInfo.AppInstanceState)
		return upErr
	}
	return nil
}

func (r *ApplicationDeploymentReconciler) DeleteApplicationDeployment(ctx context.Context, a *v1beta1.ApplicationDeployment, fed *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [AppDep][REST] Deleting external appDep")
	// we should delete the appDep
	res, err := r.GetOPGClient(
		fed.Status.FederationContextId,
		fed.Spec.FederationData.RestOptions.TokenUrl,
		fed.Spec.FederationData.ClientId,
	).RemoveAppWithResponse(
		context.TODO(),
		fed.Status.FederationContextId,
		a.Spec.AppId,
		a.Spec.AppInstanceId,
		a.Spec.ZoneId,
	)
	if err != nil {
		log.Error(err, ">>> [AppInst][REST] Error deleting external appInst")
		return err
	}

	statusCode := res.StatusCode()

	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info(">>> [AppDep][REST] Deleted external appDep")
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateTerminating
		// federResponse.OfferedAvailabilityZones
	case statusCode == 400:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
	case statusCode == 401:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
	case statusCode == 404:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
	case statusCode == 409:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
	case statusCode == 422:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
	case statusCode == 500:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
	case statusCode == 503:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
	case statusCode == 520:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
	default:
		log.Info(">>> [AppDep][REST] Unexpected status code", "status", statusCode, "body", string(res.Body))
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStatePending
	}
	upErr := r.Status().Update(ctx, a)
	if upErr != nil {
		log.Error(upErr, ">>> [AppDep][REST] UNEXPECTED ERROR updating application deployment status", "name", a.Name, "namespace", a.Namespace, "state", a.Status.AppInstanceInfo.AppInstanceState)
		return upErr
	}
	return nil
}

func (r *ApplicationDeploymentReconciler) UpdateApplicationDeploymentStatus(ctx context.Context, a *v1beta1.ApplicationDeployment, fed *v1beta1.Federation) error {
	log := log.FromContext(ctx)

	// Check if callback is configured
	if fed.Spec.FederationData.RestOptions.PartnerStatusLink == "" {
		log.Info(">>> [AppDep][REST] No callback StatusLink configured in Federation, skipping callback")
		return nil
	}

	log.Info(">>> [AppDep][REST] Sending AppDep callback to Guest",
		"appInstanceId", a.Spec.AppInstanceId,
		"state", a.Status.AppInstanceInfo.AppInstanceState,
		"statusLink", fed.Spec.FederationData.RestOptions.PartnerStatusLink)
	// Build callback body with current status
	// AppInstCallbackLinkJSONRequestBody requires: AppId, AppInstanceId, AppInstanceInfo, ZoneId
	state := opgmodels.InstanceState(a.Status.AppInstanceInfo.AppInstanceState)
	callbackBody := opgmodels.AppInstCallbackLinkJSONRequestBody{
		AppId:         a.Spec.AppId,
		AppInstanceId: a.Spec.AppInstanceId,
		ZoneId:        a.Spec.ZoneId,
	}
	callbackBody.AppInstanceInfo.AppInstanceState = &state
	accessPointInfo := opgmodels.AccessPointInfo{}
	if len(a.Status.AccessPointInfo) > 0 {
		for _, ap := range a.Status.AccessPointInfo {
			endpoint := opgmodels.ServiceEndpoint{
				Port: ap.AccessPoints.Port,
			}
			if ap.AccessPoints.Fqdn != "" {
				fqdn := opgmodels.EdgeAppFQDN(ap.AccessPoints.Fqdn)
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
		fed.Status.FederationContextId,
		fed.Spec.FederationData.RestOptions.PartnerStatusLink,
		"host",
	).AppInstCallbackLinkWithResponse(
		context.TODO(),
		fed.Status.FederationContextId,
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
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
	case statusCode == 401:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
	case statusCode == 404:
		handleApplicationDeploymentProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
	default:
		log.Info(">>> [AppInst][REST] ApplicationInstance callback returned unexpected status", "status", statusCode, "body", string(res.Body))
		a.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStatePending
	}
	upErr := r.Status().Update(ctx, a)
	if upErr != nil {
		log.Error(upErr, ">>> [AppInst][REST] UNEXPECTED ERROR updating application deployment status", "name", a.Name, "namespace", a.Namespace, "state", a.Status.AppInstanceInfo.AppInstanceState)
		return upErr
	}
	return nil
}
