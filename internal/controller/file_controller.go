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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	opgmodels "github.com/neonephos-katalis/opg-ewbi-api/api/federation/models"
	"github.com/neonephos-katalis/opg-ewbi-operator/api/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/indexer"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/multipart"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
)

// FileReconciler reconciles a File object
type FileReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=files,verbs=*,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=files/status,verbs=get;update;patch,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=files/finalizers,verbs=update,namespace=foo

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Modify the Reconcile function to compare the state specified by
// the File object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile
func (r *FileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("name", req.Name, "namespace", req.Namespace)
	log.Info("starting reconcile function for file")
	defer log.Info("end reconcile for file")

	// Getting main file or requeue
	var f v1beta1.File
	if err := r.Get(ctx, req.NamespacedName, &f); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("file object not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "error getting file object")
		return ctrl.Result{}, err
	}
	log.Info("File object obtained", "name", f.Spec.FileName, "version", f.Spec.FileVersion)

	// Getting file's federation or requeue by using federation-context-id label
	isGuest := IsGuestResource(f.Labels)
	extraLabels := map[string]string{}
	if isGuest {
		extraLabels[v1beta1.FederationRelationLabel] = string(v1beta1.FederationRelationGuest)
	} else {
		extraLabels[v1beta1.FederationRelationLabel] = string(v1beta1.FederationRelationHost)
	}

	feder, err := GetFederationByContextId(ctx, r.Client, f.Labels[v1beta1.FederationContextIdLabel], extraLabels)
	if err != nil {
		log.Error(err, "A File should always have a parent federation")
		f.Status.Phase = v1beta1.FilePhaseError
		upErr := r.Status().Update(ctx, f.DeepCopy())
		if upErr != nil {
			log.Error(upErr, errorUpdatingResourceStatusMsg)
		}
		return ctrl.Result{}, err
	}

	log.Info("Federation object obtained", "name", feder.Name)

	if f.GetDeletionTimestamp().IsZero() {
		if controllerutil.AddFinalizer(&f, v1beta1.FileFinalizer) {
			log.Info("Added finalizer to File")
			if err := r.Update(ctx, f.DeepCopy()); err != nil {
				log.Info("unable to Update File with finalizer")
				return ctrl.Result{}, err
			}
			log.Info("Successfully added finalizer to file")
			return ctrl.Result{}, nil
		}
	} else {
		if isGuest {
			if err := r.handleExternalFileDeletion(ctx, &f, feder); err != nil {
				log.Error(err, "error deleting file")
				f.Status.Phase = v1beta1.FilePhaseError
				upErr := r.Status().Update(ctx, f.DeepCopy())
				if upErr != nil {
					log.Error(upErr, errorUpdatingResourceStatusMsg)
				}
				return ctrl.Result{}, err
			}
		}
		// if external file is correctly deleted, we can remove the finalizer
		if controllerutil.RemoveFinalizer(&f, v1beta1.FileFinalizer) {
			log.Info("Removed basic finalizer for File")
			if err := r.Update(ctx, f.DeepCopy()); err != nil {
				//log.Error(err, "update failed while removing finalizers") //Commented to reduce log noise
				return ctrl.Result{}, nil
			}
			log.Info("removed all finalizers, exiting...")
			return ctrl.Result{}, nil
		}
	}

	// if federation is guest, send OPG API request
	if isGuest {
		if result, err := r.handleExternalFileCreation(ctx, &f, feder); err != nil {
			log.Info("error creating file")
			f.Status.Phase = v1beta1.FilePhaseError
			upErr := r.Status().Update(ctx, f.DeepCopy())
			if upErr != nil {
				log.Error(upErr, errorUpdatingResourceStatusMsg)
			}
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		} else {
			return result, nil
		}
	} else {
		if f.Status.Phase == "" {
			f.Status.Phase = v1beta1.FilePhaseReady
			f.Status.State = "Pending"
			log.Info("Initialized new CR state", "phase", f.Status.Phase, "state", f.Status.State)
		} else {
			log.Info("Existing CR state", "phase", f.Status.Phase, "state", f.Status.State)
		}
		if f.GetDeletionTimestamp().IsZero() {
			upErr := r.Status().Update(ctx, f.DeepCopy())
			if upErr != nil {
				log.Error(upErr, errorUpdatingResourceStatusMsg)
			}
		}
		return ctrl.Result{}, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *FileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := indexer.GetFederationIndexers(context.Background(), mgr); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.File{}).
		Named("file").
		Complete(r)
}

func (r *FileReconciler) handleExternalFileCreation(
	ctx context.Context, f *v1beta1.File, feder *v1beta1.Federation,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)
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
		log.Error(err, "error serializing multipart body")
		return ctrl.Result{}, err
	}
	res, err := r.GetOPGClient(
		feder.Labels[v1beta1.ExternalIdLabel],
		feder.Spec.GuestPartnerCredentials.TokenUrl,
		feder.Spec.GuestPartnerCredentials.ClientId,
	).UploadFileWithBodyWithResponse(
		context.TODO(),
		feder.Status.FederationContextId,
		contentType,
		body)
	if err != nil {
		log.Error(err, "error creating file")
		return ctrl.Result{}, err
	}
	statusCode := res.StatusCode()
	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info("FILE - Status code 2xx received from OPG API", "status", statusCode)
		f.Status.Phase = v1beta1.FilePhaseReady
		switch statusCode {
		case 202:
			f.Status.State = "Pending"
		case 200:
			f.Status.State = "Ready"
		default:
			f.Status.State = "Pending"
		}
		log.Info("Created/Updated external file", "phase", f.Status.Phase, "state", f.Status.State)
		r.Status().Update(ctx, f)
		return ctrl.Result{}, nil
		//return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
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
		if *res.ApplicationproblemJSON500.Detail == "file not found" {
			return ctrl.Result{}, errors.New(*res.ApplicationproblemJSON500.Detail)
		}
	case statusCode == 503:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
	case statusCode == 520:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
	default:
		f.Status.Phase = v1beta1.FilePhaseReady
		f.Status.State = "Error"
		r.Status().Update(ctx, f)
		//log.Info(unexpectedStatusCodeMsg, "status", statusCode, "body", string(res.Body))
	}
	return ctrl.Result{}, nil
}

func (r *FileReconciler) handleExternalFileDeletion(
	ctx context.Context, f *v1beta1.File, feder *v1beta1.Federation,
) error {
	log := log.FromContext(ctx)
	log.Info("Deleting external file")
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
