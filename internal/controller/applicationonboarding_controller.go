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

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	rest "github.com/neonephos-katalis/opg-ewbi-operator/internal/rest"
	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/uuid"
)

// ApplicationOnboardingReconciler reconciles a ApplicationOnboarding object
type ApplicationOnboardingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
	RestClient *rest.ApplicationOnboardingReconciler
}

type ExternalAppOnboardingClient interface {
	CreateApplicationOnboarding(ctx context.Context, f *v1beta1.ApplicationOnboarding, fed *v1beta1.Federation) error
	DeleteApplicationOnboarding(ctx context.Context, f *v1beta1.ApplicationOnboarding, fed *v1beta1.Federation) error
	UpdateApplicationOnboardingStatus(ctx context.Context, f *v1beta1.ApplicationOnboarding, fed *v1beta1.Federation) error //Callback for REST and GET for K8s
}

func (r *ApplicationOnboardingReconciler) getExternalClient(isRest bool) ExternalAppOnboardingClient {
	if isRest {
		return r.RestClient
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationOnboardingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.ApplicationOnboarding{}).
		Named("applicationonboarding").
		Complete(r)
}

// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=applicationonboardings,verbs=*,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=applicationonboardings/status,verbs=get;update;patch,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=applicationonboardings/finalizers,verbs=update,namespace=foo

func (r *ApplicationOnboardingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := ctrl.Log
	log.Info(">>> [AppOnboard] Starting RECONCILE FUNCTION.", "name", req.Name, "namespace", req.Namespace)
	defer log.Info(">>> [AppOnboard] End RECONCILE FUNCTION.", "name", req.Name, "namespace", req.Namespace)

	// Getting main ApplicationOnboarding or requeue
	var appOnboard v1beta1.ApplicationOnboarding
	if err := r.Get(ctx, req.NamespacedName, &appOnboard); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, ">>> [AppOnboard] Error getting object.", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	//Helper function to set the status to NotAvailable and update the resource
	originalAppOnboard := appOnboard.DeepCopy()
	defer func() {
		isDeleting := !appOnboard.GetDeletionTimestamp().IsZero()
		if err != nil && !isDeleting {
			log.Error(err, ">>> [AppOnboard] UNEXPECTED ERROR detected in Reconcile, setting state to Failed before patching", "name", appOnboard.Name, "namespace", appOnboard.Namespace)
			appOnboard.Status.State = v1beta1.ApplicationOnboardingStateFailed
		}

		// Metadata Patch (Annotations, Labels, Finalizers)
		metaChanged := !reflect.DeepEqual(appOnboard.Annotations, originalAppOnboard.Annotations) ||
			!reflect.DeepEqual(appOnboard.Labels, originalAppOnboard.Labels) ||
			!reflect.DeepEqual(appOnboard.Finalizers, originalAppOnboard.Finalizers)

		if metaChanged {
			currentStatus := appOnboard.Status.DeepCopy()
			// Using Patch instead of Update to avoid overwriting changes made by other controllers
			if patchErr := r.Patch(ctx, &appOnboard, client.MergeFrom(originalAppOnboard)); patchErr != nil {
				if !apierrors.IsNotFound(patchErr) {
					log.Error(patchErr, ">>> [AppOnboard] UNEXPECTED ERROR during ApplicationOnboarding Metadata UPDATE.", "name", appOnboard.Name, "namespace", appOnboard.Namespace)
				}
				if err == nil {
					err = patchErr
				}
				return // If there's an error patching metadata, we return early to avoid patching status with potentially inconsistent data
			}
			if currentStatus != nil {
				appOnboard.Status = *currentStatus
			}
			// Alignment of the resource version after patching metadata
			originalAppOnboard.SetResourceVersion(appOnboard.GetResourceVersion())
		}
		if isDeleting {
			return
		}
		// Status Update
		if patchErr := r.Status().Patch(ctx, &appOnboard, client.MergeFrom(originalAppOnboard)); patchErr != nil {
			if !apierrors.IsNotFound(patchErr) {
				log.Error(patchErr, ">>> [AppOnboard] UNEXPECTED ERROR during ApplicationOnboarding Status UPDATE.", "name", appOnboard.Name, "namespace", appOnboard.Namespace)
			}
			if err == nil {
				err = patchErr
			}
		} else {
			log.Info(">>> [AppOnboard] SUCCESSFULLY Reconciled.", "name", appOnboard.Name, "namespace", appOnboard.Namespace)
		}
	}()

	isGuest := IsGuestResource(appOnboard.Spec.RelationType)
	fed, isRest, err := GetFederation(ctx, isGuest, r.Client, appOnboard.Spec.FederationContextId, appOnboard.Namespace)
	extClient := r.getExternalClient(isRest)
	if err != nil {
		log.Error(err, ">>> [AppOnboard] Should always have a parent federation.", "name", appOnboard.Name, "namespace", appOnboard.Namespace)
		return ctrl.Result{}, err
	}
	// Check if the federation is locked and stop the watcher if it is (K8s only) or stop the callbacks if it is (REST only)
	if !CheckFederationState(fed, isRest, "AppOnboard", appOnboard.Name, appOnboard.Namespace) {
		return ctrl.Result{}, nil
	}

	// Handle deletion of the ApplicationOnboarding resource
	if !appOnboard.GetDeletionTimestamp().IsZero() {
		if isGuest {
			if err := extClient.DeleteApplicationOnboarding(ctx, &appOnboard, fed); err != nil {
				log.Error(err, ">>> [AppOnboard] Error deleting ApplicationOnboarding.", "name", appOnboard.Name, "namespace", appOnboard.Namespace)
				appOnboard.Status.State = v1beta1.ApplicationOnboardingStateFailed
				return ctrl.Result{}, err
			}
		}
		if controllerutil.RemoveFinalizer(&appOnboard, v1beta1.ApplicationOnboardingFinalizer) {
			log.Info(">>> [AppOnboard] Removed basic finalizer for ApplicationOnboarding, exiting...", "name", appOnboard.Name, "namespace", appOnboard.Namespace)
		}
		return ctrl.Result{}, nil
	}

	// Handle creation/finalizer
	if controllerutil.AddFinalizer(&appOnboard, v1beta1.ApplicationOnboardingFinalizer) {
		log.Info(">>> [AppOnboard] Added finalizer to ApplicationOnboarding.", "name", appOnboard.Name, "namespace", appOnboard.Namespace)
		return ctrl.Result{}, nil
	}
	if appOnboard.Labels == nil {
		appOnboard.Labels = make(map[string]string)
	}

	isNewAppOnboard := appOnboard.Status.State == ""
	if !isGuest {
		// Host ApplicationOnboarding handling
		if isNewAppOnboard {
			appOnboard.Status.State = v1beta1.ApplicationOnboardingStatePending
			appOnboard.Labels[v1beta1.ResourceIdLabel] = "apponboard-" + uuid.V5(appOnboard.Spec.AppInfo.AppId+appOnboard.Spec.FederationContextId)
		} else {
			// Callback for REST and GET for K8s
			if isRest {
				if err := extClient.UpdateApplicationOnboardingStatus(ctx, &appOnboard, fed); err != nil {
					log.Error(err, ">>> [AppOnboard][REST] Error during CALLBACK OPERATION via OPG EWBI API.", "name", appOnboard.Name, "namespace", appOnboard.Namespace)
					return ctrl.Result{}, err
				}
			} else {
				log.Info(">>> [AppOnboard][K8s] Resource updated (GUEST via watcher update through the resource)", "name", appOnboard.Name, "namespace", appOnboard.Namespace)
			}
		}
		return ctrl.Result{}, nil
	} else {
		// Guest ApplicationOnboarding handling
		if isNewAppOnboard {
			appOnboard.Status.State = v1beta1.ApplicationOnboardingStatePending
			appComponentSpec := appOnboard.Spec.AppInfo.AppComponentSpecs

			for _, appComponent := range appComponentSpec {
				artObj := &v1beta1.Artefact{}
				artList := &v1beta1.ArtefactList{}
				if err := r.List(
					ctx,
					artList,
					client.InNamespace(appOnboard.Namespace),
					client.MatchingLabels{
						v1beta1.ResourceIdLabel: "art-" + uuid.V5(appComponent.ArtefactId+appOnboard.Spec.FederationContextId),
					}); err != nil {
					return ctrl.Result{}, err
				}
				if len(artList.Items) == 0 {
					log.Info(">>> [Artefact] No Artefact foud for AppOnboard ", "name", appOnboard.Name, "naemspace", appOnboard.Namespace, "artefactId", appComponent.ArtefactId)
				}
				artObj = &artList.Items[0]
				if artObj.Status.State != v1beta1.ArtefactStateReady {
					log.Info(">>> [AppOnboard] Artefact is not READY for ApplicationOnboarding.", "name", appOnboard.Name, "naemspace", appOnboard.Namespace, "artefactId", appComponent.ArtefactId, "state", artObj.Status.State)
					return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
				}
			}
			if err := extClient.CreateApplicationOnboarding(ctx, &appOnboard, fed); err != nil {
				log.Error(err, ">>> [AppOnboard] Error APPLYING/UPDATING SPEC ApplicationOnboarding.", "name", appOnboard.Name, "namespace", appOnboard.Namespace)
				return ctrl.Result{}, err
			}
			if appOnboard.Labels == nil {
				appOnboard.Labels = make(map[string]string)
			}
			appOnboard.Labels[v1beta1.ResourceIdLabel] = "apponboard-" + uuid.V5(appOnboard.Spec.AppInfo.AppId+appOnboard.Spec.FederationContextId)
			log.Info(">>> [AppOnboard] SUCCESSFULLY APPLIED SPEC AND SET INITIAL STATUS.", "name", appOnboard.Name, "namespace", appOnboard.Namespace)
		} else {
			if isRest {
				log.Info(">>> [AppOnboard][REST] Received UPDATEs via CALLBACK OPERATION with OPG EWBI API.", "name", appOnboard.Name, "namespace", appOnboard.Namespace)
			} else {
				if err := extClient.UpdateApplicationOnboardingStatus(ctx, &appOnboard, fed); err != nil {
					log.Error(err, ">>> [AppOnboard][K8s] Error updating ApplicationOnboarding.", "name", appOnboard.Name, "namespace", appOnboard.Namespace)
					return ctrl.Result{}, err
				}
			}
		}
	}
	return ctrl.Result{}, nil
}
