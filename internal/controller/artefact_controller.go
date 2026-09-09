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
	"reflect"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	k8s "github.com/neonephos-katalis/opg-ewbi-operator/internal/k8s"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	rest "github.com/neonephos-katalis/opg-ewbi-operator/internal/rest"
	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/uuid"
)

// ArtefactReconciler reconciles a Artefact object
type ArtefactReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
	K8sClient  *k8s.ArtefactReconciler
	RestClient *rest.ArtefactReconciler
}

type ExternalArtefactClient interface {
	CreateArtefact(ctx context.Context, f *v1beta1.Artefact, fed *v1beta1.Federation) error
	DeleteArtefact(ctx context.Context, f *v1beta1.Artefact, fed *v1beta1.Federation) error
	UpdateArtefactStatus(ctx context.Context, f *v1beta1.Artefact, fed *v1beta1.Federation) error //Callback for REST and GET for K8s
}

func (r *ArtefactReconciler) getExternalClient(isRest bool) ExternalArtefactClient {
	if isRest {
		return r.RestClient
	}
	return r.K8sClient
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArtefactReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Artefact{}).
		Named("artefact").
		WatchesRawSource(
			source.Channel(
				k8s.ArtefactRemoteEvents,
				&handler.EnqueueRequestForObject{},
			),
		).
		Complete(r)
}

// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=artefacts,verbs=*,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=artefacts/status,verbs=get;update;patch,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=artefacts/finalizers,verbs=update,namespace=foo

func (r *ArtefactReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := ctrl.Log
	log.Info(">>> [Artefact] Starting RECONCILE FUNCTION.", "name", req.Name, "namespace", req.Namespace)
	defer log.Info(">>> [Artefact] End RECONCILE FUNCTION.", "name", req.Name, "namespace", req.Namespace)

	// Getting main artefact or requeue
	var art v1beta1.Artefact
	if err := r.Get(ctx, req.NamespacedName, &art); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, ">>> [Artefact] Error getting artefact object.", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	//Helper function to set the status to NotAvailable and update the resource
	originalArtefact := art.DeepCopy()
	defer func() {
		isDeleting := !art.GetDeletionTimestamp().IsZero()
		if err != nil && !isDeleting {
			log.Error(err, ">>> [Artefact] UNEXPECTED ERROR detected in Reconcile, setting state to Failed before patching", "name", art.Name, "namespace", art.Namespace)
			art.Status.State = v1beta1.ArtefactStateError
		}

		// Metadata Patch (Annotations, Labels, Finalizers)
		metaChanged := !reflect.DeepEqual(art.Annotations, originalArtefact.Annotations) ||
			!reflect.DeepEqual(art.Labels, originalArtefact.Labels) ||
			!reflect.DeepEqual(art.Finalizers, originalArtefact.Finalizers)

		if metaChanged {
			currentStatus := art.Status.DeepCopy()
			// Using Patch instead of Update to avoid overwriting changes made by other controllers
			if patchErr := r.Patch(ctx, &art, client.MergeFrom(originalArtefact)); patchErr != nil {
				if !apierrors.IsNotFound(patchErr) {
					log.Error(patchErr, ">>> [Artefact] UNEXPECTED ERROR during Artefact Metadata UPDATE.", "name", art.Name, "namespace", art.Namespace)
				}
				if err == nil {
					err = patchErr
				}
				return // If there's an error patching metadata, we return early to avoid patching status with potentially inconsistent data
			}
			if currentStatus != nil {
				art.Status = *currentStatus
			}
			// Alignment of the resource version after patching metadata
			originalArtefact.SetResourceVersion(art.GetResourceVersion())
		}
		if isDeleting {
			return
		}
		// Status Update
		if patchErr := r.Status().Patch(ctx, &art, client.MergeFrom(originalArtefact)); patchErr != nil {
			if !apierrors.IsNotFound(patchErr) {
				log.Error(patchErr, ">>> [Artefact] UNEXPECTED ERROR during Artefact Status UPDATE.", "name", art.Name, "namespace", art.Namespace)
			}
			if err == nil {
				err = patchErr
			}
		} else {
			log.Info(">>> [Artefact] SUCCESSFULLY Reconciled.", "name", art.Name, "namespace", art.Namespace)
		}
	}()

	isGuest := IsGuestResource(art.Spec.RelationType)
	fed, isRest, err := GetFederation(ctx, isGuest, r.Client, art.Spec.FederationContextId, art.Namespace)
	extClient := r.getExternalClient(isRest)
	if err != nil {
		log.Error(err, ">>> [Artefact] Should always have a parent federation.", "name", art.Name, "namespace", art.Namespace)
		art.Status.State = v1beta1.ArtefactStateError
		return ctrl.Result{}, err
	}

	// Check if the federation is locked and stop the watcher if it is (K8s only) or stop the callbacks if it is (REST only)
	if !CheckFederationState(fed, isRest, "Artefact", art.Name, art.Namespace) {
		return ctrl.Result{}, nil
	}

	// Handle deletion of the Artefact resource
	if !art.GetDeletionTimestamp().IsZero() {
		if isGuest {
			if err := extClient.DeleteArtefact(ctx, &art, fed); err != nil {
				log.Error(err, ">>> [Artefact] Error deleting Artefact.", "name", art.Name, "namespace", art.Namespace)
				art.Status.State = v1beta1.ArtefactStateError
				return ctrl.Result{}, err
			}
		}
		if controllerutil.RemoveFinalizer(&art, v1beta1.ArtefactFinalizer) {
			log.Info(">>> [Artefact] Removed basic finalizer for Artefact, exiting...", "name", art.Name, "namespace", art.Namespace)
		}
		return ctrl.Result{}, nil
	}

	// Handle creation/finalizer
	if controllerutil.AddFinalizer(&art, v1beta1.ArtefactFinalizer) {
		log.Info(">>> [Artefact] Added finalizer to Artefact.", "name", art.Name, "namespace", art.Namespace)
		return ctrl.Result{}, nil
	}

	if art.Labels == nil {
		art.Labels = make(map[string]string)
	}

	isNewArtefact := art.Status.State == ""
	if !isGuest {
		// Host Artefact handling
		if isNewArtefact {
			art.Status.State = v1beta1.ArtefactStatePending
			art.Labels[v1beta1.ResourceIdLabel] = "art-" + uuid.V5(art.Spec.ArtefactId+art.Spec.FederationContextId)
		} else {
			if isRest {
				if err := extClient.UpdateArtefactStatus(ctx, &art, fed); err != nil {
					log.Error(err, ">>> [Artefact][REST] Error during CALLBACK OPERATION via OPG EWBI API.", "name", art.Name, "namespace", art.Namespace)
					return ctrl.Result{}, err
				}
			} else {
				log.Info(">>> [Artefact][K8s] Resource updated (GUEST via watcher update through the resource)", "name", art.Name, "namespace", art.Namespace)
			}
		}
		return ctrl.Result{}, nil
	} else {
		// Guest Artefact handling
		if isNewArtefact {
			art.Status.State = v1beta1.ArtefactStatePending
			art.Labels[v1beta1.ResourceIdLabel] = "art-" + uuid.V5(art.Spec.ArtefactId+art.Spec.FederationContextId)
			componentSpec := art.Spec.ArtefactBody.ComponentSpec
			for _, component := range componentSpec {
				image := component.Images
				for _, imageId := range image {
					imageObj := &v1beta1.Image{}
					imageList := &v1beta1.ImageList{}
					if err := r.List(
						ctx,
						imageList,
						client.InNamespace(art.Namespace),
						client.MatchingLabels{
							v1beta1.ResourceIdLabel: "image-" + uuid.V5(imageId+art.Spec.FederationContextId),
						}); err != nil {
						return ctrl.Result{}, err
					}
					if len(imageList.Items) == 0 {
						log.Info(">>> [Artefact] No Image foud for Artefact ", "name", art.Name, "naemspace", art.Namespace, "imageId", imageId)
					}
					imageObj = &imageList.Items[0]
					if imageObj.Status.State != v1beta1.ImageStateReady {
						log.Info(">>> [Artefact] Image is not READY for Artefact.", "name", art.Name, "namespace", art.Namespace, "imageId", imageId, "imageState", imageObj.Status.State)
						return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
					}
				}
			}
			if err := extClient.CreateArtefact(ctx, &art, fed); err != nil {
				log.Error(err, ">>> [Artefact] Error APPLYING/UPDATING SPEC Artefact.", "name", art.Name, "namespace", art.Namespace)
				return ctrl.Result{}, err
			}
			log.Info(">>> [Artefact] SUCCESSFULLY APPLIED SPEC AND SET INITIAL STATUS.", "name", art.Name, "namespace", art.Namespace)
			return ctrl.Result{}, nil
		} else {
			if isRest {
				log.Info(">>> [Artefact][REST] Received UPDATEs via CALLBACK OPERATION with OPG EWBI API.", "name", art.Name, "namespace", art.Namespace)
			} else {
				if err := extClient.UpdateArtefactStatus(ctx, &art, fed); err != nil {
					log.Error(err, ">>> [Artefact][K8s] Error updating Artefact.", "name", art.Name, "namespace", art.Namespace)
					return ctrl.Result{}, err
				}
			}
		}

	}
	return ctrl.Result{}, nil
}
