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

// FileReconciler reconciles a File object
type FileReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

func handleFileProblemDetails(log logr.Logger, code int, p *opgmodels.ProblemDetails) {
	log.Info(">>> [File] Response with error", "error", code, "details", p)
}

const (
	errorUpdatingFileStatusMsg = ">>> [File][REST] Error Updating resource status"
	unexpectedStatusFileMsg    = ">>> [File][REST] Unexpected Status Code"
)

func (r *FileReconciler) CreateFile(ctx context.Context, f *v1beta1.File, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [File][REST] Creating external file via REST API")
	fileReqBody := opgmodels.UploadFileMultipartBody{
		AppProviderId: f.Spec.AppProviderId,
		FileId:        f.Labels[v1beta1.ExternalIdLabel],
		FileName:      f.Spec.FileName,
		FileRepoLocation: &opgmodels.ObjectRepoLocation{
			Password: &f.Spec.Repo.Password,
			RepoURL:  &f.Spec.Repo.URL,
			Token:    &f.Spec.Repo.Token,
			UserName: &f.Spec.Repo.UserName,
		},
		FileType:        opgmodels.VirtImageType(f.Spec.FileType),
		FileVersionInfo: f.Spec.FileVersion,
		ImgInsSetArch:   opgmodels.CPUArchType(f.Spec.Image.InstructionSetArchitecture),
		ImgOSType: opgmodels.OSType{
			Architecture: opgmodels.OSTypeArchitecture(f.Spec.Image.OS.Architecture),
			Distribution: opgmodels.OSTypeDistribution(f.Spec.Image.OS.Distribution),
			License:      opgmodels.OSTypeLicense(f.Spec.Image.OS.License),
			Version:      opgmodels.OSTypeVersion(f.Spec.Image.OS.Version),
		},
		RepoType: (*opgmodels.UploadFileMultipartBodyRepoType)(&f.Spec.Repo.Type),
	}

	body, contentType, err := multipart.SerializeUploadFileMultipartBody(fileReqBody)
	if err != nil {
		log.Error(err, ">>> [File][REST] Error serializing multipart body")
		return err
	}
	res, err := r.GetOPGClient(
		feder.Labels[v1beta1.ExternalIdLabel],
		feder.Spec.GuestPartnerCredentials.TokenUrl,
		feder.Spec.GuestPartnerCredentials.ClientId,
	).UploadFileWithBodyWithResponse(
		context.TODO(),
		f.Labels[v1beta1.FederationContextIdLabel],
		contentType,
		body)
	if err != nil {
		log.Error(err, ">>> [File][REST] Error creating file")
		f.Status.State = v1beta1.FileStateError
		return err
	}
	statusCode := res.StatusCode()
	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info(">>> [File][REST] Status code 2xx received from OPG API", "status", statusCode)
		f.Status.State = v1beta1.FileStatePending
		log.Info(">>> [File][REST] Created external file", "state", f.Status.State)
	case statusCode == 400:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		f.Status.State = v1beta1.FileStateError
		log.Info(">>> [File][REST] Couldn't be created", "Detail", res.ApplicationproblemJSON400.Detail)
		return errors.New(*res.ApplicationproblemJSON400.Detail)
	case statusCode == 401:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		f.Status.State = v1beta1.FileStateError
	case statusCode == 404:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		f.Status.State = v1beta1.FileStateError
	case statusCode == 409:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		f.Status.State = v1beta1.FileStateError
	case statusCode == 422:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		f.Status.State = v1beta1.FileStateError
	case statusCode == 500:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		f.Status.State = v1beta1.FileStateError
		// this should be deleted when API returns a 400 for this case
		if *res.ApplicationproblemJSON500.Detail == "file not found" {
			return errors.New(*res.ApplicationproblemJSON500.Detail)
		}
	case statusCode == 503:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		f.Status.State = v1beta1.FileStateError
	case statusCode == 520:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		f.Status.State = v1beta1.FileStateError
	default:
		f.Status.State = v1beta1.FileStatePending
	}
	upErr := r.Status().Update(ctx, f)
	if upErr != nil {
		log.Error(upErr, ">>> [File][REST] Error updating file status")
	}
	return nil
}

func (r *FileReconciler) CallbackFile(ctx context.Context, f *v1beta1.File, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	// Check if callback is configured
	if feder.Spec.Partner.StatusLink == "" {
		log.Info(">>> [File][REST] No callback StatusLink configured in Federation, skipping App callback")
		return nil
	}
	log.Info(">>> [File][REST] Sending App callback to Guest",
		"appId", f.Labels[v1beta1.ExternalIdLabel],
		"state", f.Status.State,
		"statusLink", feder.Spec.Partner.StatusLink)
	callbackBody := opgmodels.FileStatusCallbackLinkJSONRequestBody{
		FileId:       f.Labels[v1beta1.ExternalIdLabel],
		UpdateStatus: opgmodels.FileStatusCallbackLinkJSONBodyUpdateStatus(f.Status.State),
	}
	// Get callback client (pointing to Guest's callback URL via Federation.spec.partner.statusLink)
	res, err := r.GetOPGClient(
		feder.Labels[v1beta1.ExternalIdLabel],
		feder.Spec.Partner.StatusLink,
		feder.Spec.Partner.CallbackCredentials.ClientId,
	).FileStatusCallbackLinkWithResponse(
		context.TODO(),
		feder.Spec.Partner.CallbackCredentials.ClientId,
		callbackBody)

	if err != nil {
		log.Error(err, ">>> [File][REST] Error sending App callback")
		return err
	}
	statusCode := res.StatusCode()
	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info(">>> [File][REST] Successfully sent File callback to Guest", "status", statusCode)
	case statusCode == 400:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		f.Status.State = v1beta1.FileStateError
	case statusCode == 401:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		f.Status.State = v1beta1.FileStateError
	case statusCode == 404:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		f.Status.State = v1beta1.FileStateError
	default:
		log.Info(">>> [File][REST] Callback returned unexpected status", "status", statusCode, "body", string(res.Body))
		f.Status.State = v1beta1.FileStatePending
	}
	upErr := r.Status().Update(ctx, f.DeepCopy())
	if upErr != nil {
		log.Error(upErr, errorUpdatingFileStatusMsg)
		return upErr
	}
	return nil
}
func (r *FileReconciler) DeleteFile(ctx context.Context, f *v1beta1.File, feder *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [File][REST] Deleting external file")
	// we should delete the file
	res, err := r.GetOPGClient(
		feder.Labels[v1beta1.ExternalIdLabel],
		feder.Spec.GuestPartnerCredentials.TokenUrl,
		feder.Spec.GuestPartnerCredentials.ClientId,
	).RemoveFileWithResponse(
		context.TODO(),
		feder.Status.FederationContextId,
		f.Labels[v1beta1.ExternalIdLabel],
	)
	if err != nil {
		log.Error(err, ">>> [File][REST] Error deleting external file")
		return err
	}

	statusCode := res.StatusCode()

	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info(">>> [File][REST] Deleted external file successfully")
		// federResponse.OfferedAvailabilityZones
	case statusCode == 400:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		f.Status.State = v1beta1.FileStateError
	case statusCode == 401:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		f.Status.State = v1beta1.FileStateError
	case statusCode == 404:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		f.Status.State = v1beta1.FileStateError
	case statusCode == 409:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		f.Status.State = v1beta1.FileStateError
	case statusCode == 422:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		f.Status.State = v1beta1.FileStateError
	case statusCode == 500:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		f.Status.State = v1beta1.FileStateError
	case statusCode == 503:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		f.Status.State = v1beta1.FileStateError
	case statusCode == 520:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		f.Status.State = v1beta1.FileStateError
	default:
		log.Info(unexpectedStatusFileMsg, "status", statusCode, "body", string(res.Body))
		f.Status.State = v1beta1.FileStatePending
	}
	upErr := r.Status().Update(ctx, f.DeepCopy())
	if upErr != nil {
		log.Error(upErr, errorUpdatingFileStatusMsg)
		return upErr
	}
	return nil
}
