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
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/multipart"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ArtefactReconciler reconciles an Artefact object
type ArtefactReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

func handleArtefactProblemDetails(log logr.Logger, code int, p *opgmodels.ProblemDetails) {
	log.Info(">>> [Artefact][REST] Response with error", "error", code, "details", p)
}

const (
	errorUpdatingArtefactStatusMsg = ">>> [Artefact][REST] Error Updating resource status"
	unexpectedStatusArtefactMsg    = ">>> [Artefact][REST] Unexpected Status Code"
)

func (r *ArtefactReconciler) CreateArtefact(ctx context.Context, a *v1beta1.Artefact, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	components := []opgmodels.ComponentSpec{}
	for _, c := range a.Spec.ComponentSpec {
		components = append(components, opgmodels.ComponentSpec{
			CommandLineParams: &opgmodels.CommandLineParams{
				Command:     c.CommandLineParams.Command,
				CommandArgs: &c.CommandLineParams.Args,
			},
			// CompEnvParams:          &[]opgmodels.CompEnvParams{},
			ComponentName: c.Name,
			ComputeResourceProfile: opgmodels.ComputeResourceInfo{
				CpuArchType:    opgmodels.ComputeResourceInfoCpuArchType(c.ComputeResourceProfile.CPUArchType),
				CpuExclusivity: &c.ComputeResourceProfile.CPUExclusivity,
				// DiskStorage:    new(int32),
				// Fpga:           new(int),
				// Gpu:            &[]opgmodels.GpuInfo{},
				// Hugepages:      &[]opgmodels.HugePage{},
				Memory: c.ComputeResourceProfile.Memory,
				NumCPU: c.ComputeResourceProfile.NumCPU,
				// Vpu:    new(int),
			},
			// DeploymentConfig:  &opgmodels.DeploymentConfig{},
			ExposedInterfaces: &[]opgmodels.InterfaceDetails{},
			Images:            c.Images,
			NumOfInstances:    int32(c.NumOfInstances),
			// PersistentVolumes: &[]opgmodels.PersistentVolumeDetails{},
			RestartPolicy: opgmodels.ComponentSpecRestartPolicy(c.RestartPolicy),
		})
	}

	reqBody := opgmodels.UploadArtefactMultipartBody{
		AppProviderId:          a.Spec.AppProviderId,
		ArtefactDescriptorType: opgmodels.UploadArtefactMultipartBodyArtefactDescriptorType(a.Spec.DescriptorType),
		ArtefactId:             a.Labels[v1beta1.ExternalIdLabel],
		ArtefactName:           a.Spec.ArtefactName,
		// ArtefactRepoLocation:   &opgmodels.ObjectRepoLocation{}
		// RepoType:            &"",
		ArtefactVersionInfo: a.Spec.ArtefactVersion,
		ArtefactVirtType:    opgmodels.UploadArtefactMultipartBodyArtefactVirtType(a.Spec.VirtType),
		ComponentSpec:       components,
	}

	body, contentType, err := multipart.SerializeUploadArtefactMultipartBody(reqBody)
	if err != nil {
		log.Error(err, ">>> [Artefact][REST] Error serializing multipart body")
		return err
	}

	res, err := r.GetOPGClient(
		feder.Labels[v1beta1.ExternalIdLabel],
		feder.Spec.GuestPartnerCredentials.TokenUrl,
		feder.Spec.GuestPartnerCredentials.ClientId,
	).UploadArtefactWithBodyWithResponse(
		context.TODO(),
		feder.Status.FederationContextId,
		contentType,
		body)

	if err != nil {
		log.Error(err, ">>> [Artefact][REST] Error creating Artefact")
		return err
	}

	statusCode := res.StatusCode()

	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info(">>> [Artefact][REST] Status code 2xx received from OPG API", "status", statusCode)
		a.Status.State = v1beta1.ArtefactStateReconciling
		log.Info(">>> [Artefact][REST] Created/Updated external artefact", "state", a.Status.State)
	case statusCode == 400:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		log.Info(">>> [Artefact][REST] Couldn't be created", "Detail", res.ApplicationproblemJSON400.Detail)
		return errors.New(*res.ApplicationproblemJSON400.Detail)
	case statusCode == 401:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		a.Status.State = v1beta1.ArtefactStateError
	case statusCode == 404:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		a.Status.State = v1beta1.ArtefactStateError
	case statusCode == 409:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		a.Status.State = v1beta1.ArtefactStateError
	case statusCode == 422:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		a.Status.State = v1beta1.ArtefactStateError
	case statusCode == 500:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		// this should be deleted when API returns a 400 for this case
		if *res.ApplicationproblemJSON500.Detail == "file not found" {
			return errors.New(*res.ApplicationproblemJSON500.Detail)
		}
	case statusCode == 503:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
	case statusCode == 520:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
	default:
		a.Status.State = v1beta1.ArtefactStateReconciling
	}
	upErr := r.Status().Update(ctx, a)
	if upErr != nil {
		log.Error(upErr, errorUpdatingArtefactStatusMsg)
		return upErr
	}
	return nil
}

func (r *ArtefactReconciler) DeleteArtefact(ctx context.Context, a *v1beta1.Artefact, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [Artefact][REST] Deleting external Artefact")
	// we should delete the Artefact
	res, err := r.GetOPGClient(
		feder.Labels[v1beta1.ExternalIdLabel],
		feder.Spec.GuestPartnerCredentials.TokenUrl,
		feder.Spec.GuestPartnerCredentials.ClientId,
	).RemoveArtefactWithResponse(
		context.TODO(),
		feder.Status.FederationContextId,
		a.Labels[v1beta1.ExternalIdLabel],
	)
	if err != nil {
		log.Error(err, ">>> [Artefact][REST] Error deleting artefact")
		a.Status.State = v1beta1.ArtefactStateError
		return err
	}

	statusCode := res.StatusCode()

	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info(">>> [Artefact][REST] Deleted")
		// federResponse.OfferedAvailabilityZones
		a.Status.State = v1beta1.ArtefactStateReady
	case statusCode == 400:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		a.Status.State = v1beta1.ArtefactStateError
	case statusCode == 401:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		a.Status.State = v1beta1.ArtefactStateError
	case statusCode == 404:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		a.Status.State = v1beta1.ArtefactStateError
	case statusCode == 409:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		a.Status.State = v1beta1.ArtefactStateError
	case statusCode == 422:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		a.Status.State = v1beta1.ArtefactStateError
	case statusCode == 500:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		a.Status.State = v1beta1.ArtefactStateError
	case statusCode == 503:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		a.Status.State = v1beta1.ArtefactStateError
	case statusCode == 520:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		a.Status.State = v1beta1.ArtefactStateError
	default:
		log.Info(unexpectedStatusArtefactMsg, "status", statusCode, "body", string(res.Body))
		a.Status.State = v1beta1.ArtefactStateReconciling
	}
	upErr := r.Status().Update(ctx, a)
	if upErr != nil {
		log.Error(upErr, errorUpdatingArtefactStatusMsg)
		return upErr
	}
	return nil
}

func (r *ArtefactReconciler) CallbackArtefact(ctx context.Context, a *v1beta1.Artefact, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	// Check if callback is configured
	if feder.Spec.Partner.StatusLink == "" {
		log.Info(">>> [Artefact][REST] No callback StatusLink configured in Federation, skipping App callback")
		return nil
	}
	log.Info(">>> [Artefact][REST] Sending App callback to Guest",
		"appId", a.Labels[v1beta1.ExternalIdLabel],
		"state", a.Status.State,
		"statusLink", feder.Spec.Partner.StatusLink)
	callbackBody := opgmodels.ArtefactStatusCallbackLinkJSONRequestBody{
		ArtefactId:   a.Labels[v1beta1.ExternalIdLabel],
		UpdateStatus: opgmodels.ArtefactStatusCallbackLinkJSONBodyUpdateStatus(a.Status.State),
	}
	// Get callback client (pointing to Guest's callback URL via Federation.spec.partner.statusLink)
	res, err := r.GetOPGClient(
		feder.Labels[v1beta1.ExternalIdLabel],
		feder.Spec.Partner.StatusLink,
		feder.Spec.Partner.CallbackCredentials.ClientId,
	).ArtefactStatusCallbackLinkWithResponse(
		context.TODO(),
		feder.Spec.Partner.CallbackCredentials.ClientId,
		callbackBody)

	if err != nil {
		log.Error(err, ">>> [Artefact][REST] Error sending App callback")
		return err
	}
	statusCode := res.StatusCode()
	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info(">>> [Artefact][REST] Successfully sent Artefact callback to Guest", "status", statusCode)
	case statusCode == 400:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		a.Status.State = v1beta1.ArtefactStateError
	case statusCode == 401:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		a.Status.State = v1beta1.ArtefactStateError
	case statusCode == 404:
		handleArtefactProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		a.Status.State = v1beta1.ArtefactStateError
	default:
		log.Info(">>> [Artefact][REST] Callback returned unexpected status", "status", statusCode, "body", string(res.Body))
		a.Status.State = v1beta1.ArtefactStateReconciling
	}
	upErr := r.Status().Update(ctx, a)
	if upErr != nil {
		log.Error(upErr, errorUpdatingArtefactStatusMsg)
		return upErr
	}
	return nil
}
