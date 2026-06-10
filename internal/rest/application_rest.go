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

// ApplicationReconciler reconciles an Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

func handleApplicationProblemDetails(log logr.Logger, code int, p *opgmodels.ProblemDetails) {
	log.Info(">>> [App][REST] Response with error", "error", code, "details", p)
}

const (
	errorUpdatingApplicationStatusMsg = ">>> [App][REST] Error Updating resource status"
	unexpectedStatusApplicationMsg    = ">>> [App][REST] Unexpected Status Code"
)

func (r *ApplicationReconciler) CreateApplication(ctx context.Context, a *v1beta1.Application, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	numUsers := int(a.Spec.QoSProfile.UsersPerAppInst)
	multiUserClients := opgmodels.AppQoSProfileMultiUserClients(a.Spec.QoSProfile.MultiUserClients)
	components := opgmodels.AppComponentSpecs{}

	// opgmodels.AppComponentSpecs{} is a "[]struct"
	for _, c := range a.Spec.ComponentSpecs {

		newComponent := struct {
			ArtefactId    opgmodels.ArtefactId `json:"artefactId"`
			ComponentName *string              `json:"componentName,omitempty"`
			ServiceNameEW *string              `json:"serviceNameEW,omitempty"`
			ServiceNameNB *string              `json:"serviceNameNB,omitempty"`
		}{
			ArtefactId: c.ArtefactId,
		}
		components = append(components, newComponent)
	}

	appReqBody := opgmodels.OnboardApplicationJSONRequestBody{
		// AppDeploymentZones:    &[]opgmodels.ZoneIdentifier{},
		AppId: a.Labels[v1beta1.ExternalIdLabel],
		AppMetaData: opgmodels.AppMetaData{
			AccessToken: a.Spec.MetaData.AccessToken,
			// AppDescription:  new(string),
			AppName: a.Spec.MetaData.Name,
			// Category:        &"",
			MobilitySupport: &a.Spec.MetaData.MobilitySupport,
			Version:         a.Spec.MetaData.Version,
		},
		AppProviderId: a.Spec.AppProviderId,
		AppQoSProfile: opgmodels.AppQoSProfile{
			AppProvisioning: &a.Spec.QoSProfile.Provisioning,
			// BandwidthRequired:   new(int32),
			LatencyConstraints:  opgmodels.AppQoSProfileLatencyConstraints(a.Spec.QoSProfile.LatencyConstraints),
			MultiUserClients:    &multiUserClients,
			NoOfUsersPerAppInst: &numUsers,
		},
		AppStatusCallbackLink: a.Spec.StatusLink,
		AppComponentSpecs:     components,
	}

	res, err := r.GetOPGClient(
		feder.Labels[v1beta1.ExternalIdLabel],
		feder.Spec.GuestPartnerCredentials.TokenUrl,
		feder.Spec.GuestPartnerCredentials.ClientId,
	).OnboardApplicationWithResponse(
		context.TODO(),
		feder.Status.FederationContextId,
		appReqBody)

	if err != nil {
		log.Error(err, ">>> [App][REST] Error creating app")
		return err
	}

	statusCode := res.StatusCode()
	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info(">>> [App][REST] Status code 2xx received from OPG API", "status", statusCode)

		a.Status.State = v1beta1.ApplicationStatePending
		log.Info(">>> [App][REST] Created external application", "state", a.Status.State)
	case statusCode == 400:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		log.Info(">>> [App][REST] Couldn't be created", "Detail", res.ApplicationproblemJSON400.Detail)
		return errors.New(*res.ApplicationproblemJSON400.Detail)
	case statusCode == 401:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		a.Status.State = v1beta1.ApplicationStateFailed
	case statusCode == 404:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		a.Status.State = v1beta1.ApplicationStateFailed
	case statusCode == 409:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		a.Status.State = v1beta1.ApplicationStateFailed
	case statusCode == 422:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		a.Status.State = v1beta1.ApplicationStateFailed
	case statusCode == 500:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		a.Status.State = v1beta1.ApplicationStateFailed
		// this should be deleted when API returns a 400 for this case
		if *res.ApplicationproblemJSON500.Detail == "artefact not found" {
			return errors.New(*res.ApplicationproblemJSON500.Detail)
		}
	case statusCode == 503:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		a.Status.State = v1beta1.ApplicationStateFailed
	case statusCode == 520:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		a.Status.State = v1beta1.ApplicationStateFailed
	default:
		a.Status.State = v1beta1.ApplicationStatePending
	}
	upErr := r.Status().Update(ctx, a)
	if upErr != nil {
		log.Error(upErr, errorUpdatingApplicationStatusMsg)
		return upErr
	}
	return nil
}

func (r *ApplicationReconciler) DeleteApplication(ctx context.Context, a *v1beta1.Application, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info("Deleting external app")
	// we should delete the app
	res, err := r.GetOPGClient(
		feder.Labels[v1beta1.ExternalIdLabel],
		feder.Spec.GuestPartnerCredentials.TokenUrl,
		feder.Spec.GuestPartnerCredentials.ClientId,
	).DeleteAppWithResponse(
		context.TODO(),
		feder.Status.FederationContextId,
		a.Labels[v1beta1.ExternalIdLabel],
	)
	if err != nil {
		log.Error(err, ">>> [App][REST] Error deleting application")
		a.Status.State = v1beta1.ApplicationStateFailed
		return err
	}

	statusCode := res.StatusCode()

	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info(">>> [App][REST] Deleted external application")
		a.Status.State = v1beta1.ApplicationStateRemoved
	case statusCode == 400:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		a.Status.State = v1beta1.ApplicationStateFailed
	case statusCode == 401:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		a.Status.State = v1beta1.ApplicationStateFailed
	case statusCode == 404:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		a.Status.State = v1beta1.ApplicationStateFailed
	case statusCode == 409:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		a.Status.State = v1beta1.ApplicationStateFailed
	case statusCode == 422:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		a.Status.State = v1beta1.ApplicationStateFailed
	case statusCode == 500:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		a.Status.State = v1beta1.ApplicationStateFailed
	case statusCode == 503:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		a.Status.State = v1beta1.ApplicationStateFailed
	case statusCode == 520:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		a.Status.State = v1beta1.ApplicationStateFailed
	default:
		log.Info(unexpectedStatusApplicationMsg, "status", statusCode, "body", string(res.Body))
		a.Status.State = v1beta1.ApplicationStatePending
	}

	upErr := r.Status().Update(ctx, a)
	if upErr != nil {
		log.Error(upErr, errorUpdatingApplicationStatusMsg)
		return upErr
	}
	return nil
}

func (r *ApplicationReconciler) CallbackApplication(ctx context.Context, a *v1beta1.Application, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	// Check if callback is configured
	if feder.Spec.Partner.StatusLink == "" {
		log.Info(">>> [App][REST] No callback StatusLink configured in Federation, skipping App callback")
		return nil
	}
	log.Info(">>> [App][REST] Sending App callback to Guest",
		"appId", a.Labels[v1beta1.ExternalIdLabel],
		"state", a.Status.State,
		"statusLink", feder.Spec.Partner.StatusLink)
	callbackBody := opgmodels.AppStatusCallbackLinkJSONRequestBody{
		AppId: a.Labels[v1beta1.ExternalIdLabel],
		StatusInfo: []struct {
			OnboardStatusInfo opgmodels.AppStatusCallbackLinkJSONBodyStatusInfoOnboardStatusInfo `json:"onboardStatusInfo"`
			ZoneId            opgmodels.ZoneIdentifier                                           `json:"zoneId"`
		}{
			{
				OnboardStatusInfo: opgmodels.AppStatusCallbackLinkJSONBodyStatusInfoOnboardStatusInfo(a.Status.State),
				ZoneId:            "zone-es-madrid-001",
			},
		},
	}
	// Get callback client (pointing to Guest's callback URL via Federation.spec.partner.statusLink)
	res, err := r.GetOPGClient(
		feder.Labels[v1beta1.ExternalIdLabel],
		feder.Spec.Partner.StatusLink,
		feder.Spec.Partner.CallbackCredentials.ClientId,
	).AppStatusCallbackLinkWithResponse(
		context.TODO(),
		feder.Spec.Partner.CallbackCredentials.ClientId,
		callbackBody)
	if err != nil {
		log.Error(err, ">>> [App][REST] Error sending App callback")
		a.Status.State = v1beta1.ApplicationStateFailed
		return err
	}
	statusCode := res.StatusCode()
	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info(">>> [App][REST] Successfully sent App callback to Guest", "status", statusCode)
	case statusCode == 400:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		a.Status.State = v1beta1.ApplicationStateFailed
	case statusCode == 401:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		a.Status.State = v1beta1.ApplicationStateFailed
	case statusCode == 404:
		handleApplicationProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		a.Status.State = v1beta1.ApplicationStateFailed
	default:
		log.Info(">>> [App][REST] Callback returned unexpected status", "status", statusCode, "body", string(res.Body))
		a.Status.State = v1beta1.ApplicationStatePending
	}
	upErr := r.Status().Update(ctx, a)
	if upErr != nil {
		log.Error(upErr, errorUpdatingApplicationStatusMsg)
		return upErr
	}
	return nil
}
