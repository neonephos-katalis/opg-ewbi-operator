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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ImageReconciler reconciles an Image object
type ImageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

func handleFileProblemDetails(log logr.Logger, code int, p *opgmodels.ProblemDetails) {
	log.Info(">>> [Image][REST] Response with error", "error", code, "details", p)
}

const (
	errorUpdatingImageStatusMsg = ">>> [Image][REST] Error Updating resource status"
	unexpectedStatusImageMsg    = ">>> [Image][REST] Unexpected Status Code"
)

func (r *ImageReconciler) CreateImage(ctx context.Context, image *v1beta1.Image, fed *v1beta1.Federation) error {
	log := ctrl.Log
	log.Info(">>> [Image][REST] Creating external image via REST API")
	if image.Spec.ImageBody == nil {
		return errors.New("image body is nil")
	}
	repoLocation := &opgmodels.ObjectRepoLocation{}
	if image.Spec.ImageBody.ImageRepoLocation != nil {
		repoLocation = &opgmodels.ObjectRepoLocation{
			Password: &image.Spec.ImageBody.ImageRepoLocation.Password,
			RepoURL:  &image.Spec.ImageBody.ImageRepoLocation.RepoURL,
			Token:    &image.Spec.ImageBody.ImageRepoLocation.Token,
			UserName: &image.Spec.ImageBody.ImageRepoLocation.UserName,
		}
	}
	osType := opgmodels.OSType{}
	if image.Spec.ImageBody.ImgOSType != nil {
		osType = opgmodels.OSType{
			Architecture: opgmodels.OSTypeArchitecture(image.Spec.ImageBody.ImgOSType.Architecture),
			Distribution: opgmodels.OSTypeDistribution(image.Spec.ImageBody.ImgOSType.Distribution),
			License:      opgmodels.OSTypeLicense(image.Spec.ImageBody.ImgOSType.License),
			Version:      opgmodels.OSTypeVersion(image.Spec.ImageBody.ImgOSType.Version),
		}
	}
	imageInsSetArch := opgmodels.CPUArchType("")
	if image.Spec.ImageBody.ImgInsSetArch != "" {
		imageInsSetArch = opgmodels.CPUArchType(image.Spec.ImageBody.ImgInsSetArch)
	}
	uri := opgmodels.Uri("")
	if image.Spec.ImageNotifLink != "" {
		uri = opgmodels.Uri(image.Spec.ImageNotifLink)
		log.Info(">>> [Image][REST] Callback StatusLink configured", "name", image.Name, "namespace", image.Namespace, "callback", uri)
	}
	fileReqBody := opgmodels.UploadFileMultipartBody{
		FileNotifLink:    &uri,
		AppProviderId:    image.Spec.ImageBody.AppProviderId,
		FileId:           image.Spec.ImageId,
		FileName:         image.Spec.ImageBody.ImageName,
		FileRepoLocation: repoLocation,
		FileType:         opgmodels.VirtImageType(image.Spec.ImageBody.ImageType),
		FileVersionInfo:  image.Spec.ImageBody.ImageVersionInfo,
		ImgInsSetArch:    imageInsSetArch,
		ImgOSType:        osType,
		RepoType:         (*opgmodels.RepoType)(&image.Spec.ImageBody.RepoType),
	}

	body, contentType, err := multipart.SerializeUploadFileMultipartBody(fileReqBody)
	if err != nil {
		log.Error(err, ">>> [Image][REST] Error serializing multipart body.", "name", image.Name, "namespace", image.Namespace)
		return err
	}
	res, err := r.GetOPGClient(
		fed.Status.FederationContextId,
		fed.Spec.FederationData.RestOptions.TokenUrl,
		fed.Spec.FederationData.ClientId,
	).UploadFileWithBodyWithResponse(
		context.TODO(),
		image.Spec.FederationContextId,
		contentType,
		body)
	if err != nil {
		log.Error(err, ">>> [Image][REST] Error creating image via REST API.", "name", image.Name, "namespace", image.Namespace)
		image.Status.State = v1beta1.ImageStateError
		return err
	}
	statusCode := res.StatusCode()
	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info(">>> [Image][REST] Status code 2xx received from OPG API.", " name", image.Name, "namespace", image.Namespace)
		image.Status.State = v1beta1.ImageStatePending
		log.Info(">>> [Image][REST] Created external image", "state", image.Status.State)
	case statusCode == 400:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		image.Status.State = v1beta1.ImageStateError
		log.Info(">>> [Image][REST] 400 - Couldn't be created", "Detail", res.ApplicationproblemJSON400.Detail)
		return errors.New(*res.ApplicationproblemJSON400.Detail)
	case statusCode == 401:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 404:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 409:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 422:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 500:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		image.Status.State = v1beta1.ImageStateError
		// this should be deleted when API returns a 400 for this case
		if *res.ApplicationproblemJSON500.Detail == "file not found" {
			return errors.New(*res.ApplicationproblemJSON500.Detail)
		}
	case statusCode == 503:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 520:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		image.Status.State = v1beta1.ImageStateError
	default:
		image.Status.State = v1beta1.ImageStatePending
	}
	return nil
}

func (r *ImageReconciler) UpdateImageStatus(ctx context.Context, image *v1beta1.Image, fed *v1beta1.Federation) error {
	log := ctrl.Log
	// Check if callback is configured
	if image.Spec.ImageNotifLink == "" {
		log.Info(">>> [Image][REST] No callback StatusLink configured in Image, skipping.", "name", image.Name, "namespace", image.Namespace)
		return nil
	}
	log.Info(">>> [Image][REST] Sending callback to Guest")
	callbackBody := opgmodels.FileStatusCallbackLinkJSONRequestBody{
		FileId:       opgmodels.FileId(image.Spec.ImageId),
		UpdateStatus: opgmodels.FileStatusCallbackLinkJSONBodyUpdateStatus(image.Status.State),
	}
	// Get callback client (pointing to Guest's callback URL via Federation.spec.partner.statusLink)
	res, err := r.GetOPGClient(
		fed.Status.FederationContextId,
		image.Spec.ImageNotifLink,
		"host",
	).FileStatusCallbackLinkWithResponse(
		context.TODO(),
		image.Spec.FederationContextId,
		callbackBody)

	if err != nil {
		log.Error(err, ">>> [Image][REST] Error sending Image callback to Guest.", "name", image.Name, "namespace", image.Namespace)
		return err
	}
	statusCode := res.StatusCode()
	switch {
	case statusCode == 200:
	case statusCode == 204:
		log.Info(">>> [Image][REST] Successfully sent Image callback to Guest.", "name", image.Name, "namespace", image.Namespace)
	case statusCode == 400:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 401:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 404:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 409:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 422:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 500:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		image.Status.State = v1beta1.ImageStateError
		// this should be deleted when API returns a 400 for this case
		if *res.ApplicationproblemJSON500.Detail == "file not found" {
			return errors.New(*res.ApplicationproblemJSON500.Detail)
		}
	case statusCode == 503:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 520:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		image.Status.State = v1beta1.ImageStateError
	default:
		log.Info(">>> [Image][REST] Callback returned unexpected status", "status", statusCode, "body", string(res.Body))
		image.Status.State = v1beta1.ImageStatePending
	}
	return nil
}

func (r *ImageReconciler) DeleteImage(ctx context.Context, image *v1beta1.Image, fed *v1beta1.Federation) error {
	log := log.FromContext(ctx)
	log.Info(">>> [Image][REST] Deleting external image", "name", image.Name, "namespace", image.Namespace)
	res, err := r.GetOPGClient(
		fed.Status.FederationContextId,
		fed.Spec.FederationData.RestOptions.TokenUrl,
		fed.Spec.FederationData.ClientId,
	).RemoveFileWithResponse(
		context.TODO(),
		image.Spec.FederationContextId,
		image.Spec.ImageId,
	)
	if err != nil {
		log.Error(err, ">>> [Image][REST] Error deleting external image", "name", image.Name, "namespace", image.Namespace)
		return err
	}

	statusCode := res.StatusCode()

	switch {
	case statusCode == 200:
		log.Info(">>> [Image][REST] 200 - Deleted external image successfully", "name", image.Name, "namespace", image.Namespace)
	case statusCode == 400:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 401:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 404:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 409:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 422:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 500:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 503:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		image.Status.State = v1beta1.ImageStateError
	case statusCode == 520:
		handleFileProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		image.Status.State = v1beta1.ImageStateError
	default:
		log.Info(unexpectedStatusImageMsg, "status", statusCode, "body", string(res.Body))
		image.Status.State = v1beta1.ImageStatePending
	}
	return nil
}
