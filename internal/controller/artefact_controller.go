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
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/multipart"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
)

// ArtefactReconciler reconciles a Artefact object
type ArtefactReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=artefacts,verbs=*,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=artefacts/status,verbs=get;update;patch,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=artefacts/finalizers,verbs=update,namespace=foo

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Modify the Reconcile function to compare the state specified by
// the Artefact object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile
func (r *ArtefactReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("name", req.Name, "namespace", req.Namespace)
	log.Info("starting reconcile function for artefact")
	defer log.Info("end reconcile for artefact")

	// Getting main artefact or requeue
	var a v1beta1.Artefact
	if err := r.Get(ctx, req.NamespacedName, &a); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("artefact object not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "error getting artefact object")
		return ctrl.Result{}, err
	}

	// Getting artefact's federation or requeue by using federation-context-id label
	isGuest := IsGuestResource(a.Labels)
	extraLabels := map[string]string{}
	if isGuest {
		extraLabels[v1beta1.FederationRelationLabel] = string(v1beta1.FederationRelationGuest)
	} else {
		extraLabels[v1beta1.FederationRelationLabel] = string(v1beta1.FederationRelationHost)
	}
	feder, err := GetFederationByContextId(ctx, r.Client, a.Labels[v1beta1.FederationContextIdLabel], extraLabels)
	if err != nil {
		log.Error(err, "An Artefact should always have a parent federation")
		a.Status.Phase = v1beta1.ArtefactPhaseError
		upErr := r.Status().Update(ctx, a.DeepCopy())
		if upErr != nil {
			log.Error(upErr, errorUpdatingResourceStatusMsg)
		}
		return ctrl.Result{}, err
	}

	log.Info("Federation object obtained", "name", feder.Name)

	if a.GetDeletionTimestamp().IsZero() {
		if controllerutil.AddFinalizer(&a, v1beta1.ArtefactFinalizer) {
			log.Info("Added finalizer to Artefact")
			if err := r.Update(ctx, a.DeepCopy()); err != nil {
				log.Info("unable to Update Artefact with finalizer")
				return ctrl.Result{}, err
			}
			log.Info("Successfully added finalizer to Artefact")
			return ctrl.Result{}, nil
		}
	} else {
		latest := &v1beta1.Artefact{}
		r.Get(ctx, client.ObjectKeyFromObject(&a), latest)
		if isGuest {
			if err := r.handleExternalArtefactDeletion(ctx, latest, feder); err != nil {
				log.Error(err, "error deleting Artefact")
				latest.Status.Phase = v1beta1.ArtefactPhaseError
				upErr := r.Status().Update(ctx, latest)
				if upErr != nil {
					log.Error(upErr, errorUpdatingResourceStatusMsg)
				}
				return ctrl.Result{}, err
			}
		}
		// if external Artefact is correctly deleted, we can remove the finalizer
		if controllerutil.RemoveFinalizer(&a, v1beta1.ArtefactFinalizer) {
			log.Info("Removed basic finalizer for Artefact")
			if err := r.Update(ctx, a.DeepCopy()); err != nil {
				log.Error(err, "update failed while removing finalizers")
				return ctrl.Result{}, err
			}
			log.Info("removed all finalizers, exiting...")
			return ctrl.Result{}, nil
		}
	}

	// if federation is guest, send OPG API request
	if isGuest {
		latest := &v1beta1.Artefact{}
		r.Get(ctx, client.ObjectKeyFromObject(&a), latest)
		if err := r.handleExternalArtefactCreation(ctx, latest, feder); err != nil {
			log.Info("error creating Artefact")
			latest.Status.Phase = v1beta1.ArtefactPhaseError
			upErr := r.Status().Update(ctx, latest)
			if upErr != nil {
				log.Error(upErr, errorUpdatingResourceStatusMsg)
			}
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
	} else {
		if a.Status.Phase == "" {
			a.Status.Phase = v1beta1.ArtefactPhaseReady
			a.Status.State = v1beta1.ArtefactStateReconciling
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
func (r *ArtefactReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&opgewbiv1beta1.Artefact{}).
		Named("artefact").
		Complete(r)
}

func (r *ArtefactReconciler) handleExternalArtefactCreation(
	ctx context.Context, a *v1beta1.Artefact, feder *v1beta1.Federation,
) error {
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
		log.Error(err, "error serializing multipart body")
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
		log.Error(err, "error creating Artefact")
		return err
	}

	statusCode := res.StatusCode()

	switch {
	case statusCode >= 200 && statusCode < 300:
		log.Info("ARTEFACTS - Status code 2xx received from OPG API", "status", statusCode)
		latest := &v1beta1.Artefact{}
		r.Get(ctx, client.ObjectKeyFromObject(a), latest)
		if !latest.GetDeletionTimestamp().IsZero() {
			// if external file is correctly deleted, we can remove the finalizer
			if controllerutil.RemoveFinalizer(latest, v1beta1.ArtefactFinalizer) {
				log.Info("Removed basic finalizer for Artefact")
				r.Update(ctx, latest)
				return nil
			}
		} else {
			latest.Status.Phase = v1beta1.ArtefactPhaseReady
			switch statusCode {
			case 202:
				latest.Status.State = v1beta1.ArtefactStateReconciling
			case 200:
				latest.Status.State = v1beta1.ArtefactStateReady
			default:
				latest.Status.State = v1beta1.ArtefactStateReconciling
			}
		}
		log.Info("Created/Updated external artefact", "phase", latest.Status.Phase, "state", latest.Status.State)
		r.Status().Update(ctx, latest)
		time.Sleep(3 * time.Second)
		r.handleExternalArtefactCreation(ctx, latest, feder)

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
		if *res.ApplicationproblemJSON500.Detail == "file not found" {
			return errors.New(*res.ApplicationproblemJSON500.Detail)
		}
	case statusCode == 503:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
	case statusCode == 520:
		handleProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
	default:
		latest := &v1beta1.Artefact{}
		r.Get(ctx, client.ObjectKeyFromObject(a), latest)
		latest.Status.Phase = v1beta1.ArtefactPhaseReady
		latest.Status.State = v1beta1.ArtefactStateError
		r.Status().Update(ctx, latest)
		//log.Info(unexpectedStatusCodeMsg, "status", statusCode, "body", string(res.Body))
	}
	return nil
}

func (r *ArtefactReconciler) handleExternalArtefactDeletion(
	ctx context.Context, f *v1beta1.Artefact, feder *v1beta1.Federation,
) error {
	log := log.FromContext(ctx)
	log.Info("Deleting external Artefact")
	// we should delete the Artefact
	res, err := r.GetOPGClient(
		feder.Labels[v1beta1.ExternalIdLabel],
		feder.Spec.GuestPartnerCredentials.TokenUrl,
		feder.Spec.GuestPartnerCredentials.ClientId,
	).RemoveArtefactWithResponse(
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
