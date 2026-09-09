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

// ApplicationDeploymentReconciler reconciles a ApplicationDeployment object
type ApplicationDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
	RestClient *rest.ApplicationDeploymentReconciler
}
type ExternalAppDeployClient interface {
	CreateApplicationDeployment(ctx context.Context, f *v1beta1.ApplicationDeployment, fed *v1beta1.Federation) error
	DeleteApplicationDeployment(ctx context.Context, f *v1beta1.ApplicationDeployment, fed *v1beta1.Federation) error
	UpdateApplicationDeploymentStatus(ctx context.Context, f *v1beta1.ApplicationDeployment, fed *v1beta1.Federation) error //Callback for REST and GET for K8s
}

func (r *ApplicationDeploymentReconciler) getExternalClient(isRest bool) ExternalAppDeployClient {
	if isRest {
		return r.RestClient
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.ApplicationDeployment{}).
		Named("applicationdeployment").
		Complete(r)
}

// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=applicationdeployments,verbs=*,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=applicationdeployments/status,verbs=get;update;patch,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=applicationdeployments/finalizers,verbs=update,namespace=foo

func (r *ApplicationDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := ctrl.Log
	log.Info(">>> [AppDeploy] Starting RECONCILE FUNCTION.", "name", req.Name, "namespace", req.Namespace)
	defer log.Info(">>> [AppDeploy] End RECONCILE FUNCTION.", "name", req.Name, "namespace", req.Namespace)

	// Getting main ApplicationDeployment or requeue
	var appDeploy v1beta1.ApplicationDeployment
	if err := r.Get(ctx, req.NamespacedName, &appDeploy); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, ">>> [AppDeploy] Error getting object.", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, err
	}
	//Helper function to set the status to NotAvailable and update the resource
	originalAppDeploy := appDeploy.DeepCopy()
	defer func() {
		isDeleting := !appDeploy.GetDeletionTimestamp().IsZero()
		if err != nil && !isDeleting {
			log.Error(err, ">>> [AppDeploy] UNEXPECTED ERROR detected in Reconcile, setting state to Failed before patching", "name", appDeploy.Name, "namespace", appDeploy.Namespace)
			appDeploy.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStateFailed
		}

		// Metadata Patch (Annotations, Labels, Finalizers)
		metaChanged := !reflect.DeepEqual(appDeploy.Annotations, originalAppDeploy.Annotations) ||
			!reflect.DeepEqual(appDeploy.Labels, originalAppDeploy.Labels) ||
			!reflect.DeepEqual(appDeploy.Finalizers, originalAppDeploy.Finalizers)

		if metaChanged {
			currentStatus := appDeploy.Status.DeepCopy()
			// Using Patch instead of Update to avoid overwriting changes made by other controllers
			if patchErr := r.Patch(ctx, &appDeploy, client.MergeFrom(originalAppDeploy)); patchErr != nil {
				if !apierrors.IsNotFound(patchErr) {
					log.Error(patchErr, ">>> [AppDeploy] UNEXPECTED ERROR during ApplicationDeployment Metadata UPDATE.", "name", appDeploy.Name, "namespace", appDeploy.Namespace)
				}
				if err == nil {
					err = patchErr
				}
				return // If there's an error patching metadata, we return early to avoid patching status with potentially inconsistent data
			}
			if currentStatus != nil {
				appDeploy.Status = *currentStatus
			}
			// Alignment of the resource version after patching metadata
			originalAppDeploy.SetResourceVersion(appDeploy.GetResourceVersion())
		}

		if isDeleting {
			return
		}
		// Status Update
		if patchErr := r.Status().Patch(ctx, &appDeploy, client.MergeFrom(originalAppDeploy)); patchErr != nil {
			if !apierrors.IsNotFound(patchErr) {
				log.Error(patchErr, ">>> [AppDeploy] UNEXPECTED ERROR during ApplicationDeployment Status UPDATE.", "name", appDeploy.Name, "namespace", appDeploy.Namespace)
			}
			if err == nil {
				err = patchErr
			}
		} else {
			log.Info(">>> [AppDeploy] SUCCESSFULLY Reconciled.", "name", appDeploy.Name, "namespace", appDeploy.Namespace)
		}
	}()

	isGuest := IsGuestResource(appDeploy.Spec.RelationType)
	fed, isRest, err := GetFederation(ctx, isGuest, r.Client, appDeploy.Spec.FederationContextId, appDeploy.Namespace)
	extClient := r.getExternalClient(isRest)
	if err != nil {
		log.Error(err, ">>> [AppDeploy] Should always have a parent federation.", "name", appDeploy.Name, "namespace", appDeploy.Namespace)
		return ctrl.Result{}, err
	}

	// Check if the federation is locked and stop the watcher if it is (K8s only) or stop the callbacks if it is (REST only)
	if !CheckFederationState(fed, isRest, "AppDeploy", appDeploy.Name, appDeploy.Namespace) {
		return ctrl.Result{}, nil
	}
	// Get the appropriate external client based on the federation technology

	// Handle deletion of the ApplicationDeployment resource
	if !appDeploy.GetDeletionTimestamp().IsZero() {
		if isGuest {
			if err := extClient.DeleteApplicationDeployment(ctx, &appDeploy, fed); err != nil {
				log.Error(err, ">>> [AppDeploy] Error deleting ApplicationDeployment.", "name", appDeploy.Name, "namespace", appDeploy.Namespace)
				return ctrl.Result{}, err
			}
		}
		if controllerutil.RemoveFinalizer(&appDeploy, v1beta1.ApplicationDeploymentFinalizer) {
			log.Info(">>> [AppDeploy] Removed basic finalizer for ApplicationDeployment, exiting...", "name", appDeploy.Name, "namespace", appDeploy.Namespace)
		}
		return ctrl.Result{}, nil
	}

	// Handle creation/finalizer
	if controllerutil.AddFinalizer(&appDeploy, v1beta1.ApplicationDeploymentFinalizer) {
		log.Info(">>> [AppDeploy] Added finalizer to ApplicationDeployment.", "name", appDeploy.Name, "namespace", appDeploy.Namespace)
		return ctrl.Result{}, nil
	}

	if appDeploy.Labels == nil {
		appDeploy.Labels = make(map[string]string)
	}
	isNewAppDeploy := appDeploy.Status.AppInstanceInfo.AppInstanceState == ""
	if !isGuest {
		// Host ApplicationDeployment handling
		if isNewAppDeploy {
			// Deve rispettare il pattern: [A-Za-z0-9][A-Za-z0-9_]{6,62}[A-Za-z0-9]$`
			appDeploy.Status.AppInstanceInfo.AppInstIdentifier = uuid.Base62(appDeploy.Spec.FederationContextId, appDeploy.Spec.AppId, appDeploy.Spec.AppInstanceId)
			appDeploy.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStatePending
			appDeploy.Labels[v1beta1.ResourceIdLabel] = "appdeploy-" + uuid.V5(appDeploy.Spec.FederationContextId+appDeploy.Spec.AppId+appDeploy.Spec.AppInstanceId)
		} else {
			if isRest {
				if err := extClient.UpdateApplicationDeploymentStatus(ctx, &appDeploy, fed); err != nil {
					log.Error(err, ">>> [AppDeploy] Error during CALLBACK OPERATION via OPG EWBI API.", "name", appDeploy.Name, "namespace", appDeploy.Namespace)
					return ctrl.Result{}, err
				}
			} else {
				log.Info(">>> [AppDeploy] Resource updated (GUEST via watcher update through the resource)", "name", appDeploy.Name, "namespace", appDeploy.Namespace)
			}
		}
	} else {
		// Guest ApplicationDeployment handling
		if isNewAppDeploy {
			appDeploy.Status.AppInstanceInfo.AppInstanceState = v1beta1.ApplicationDeploymentStatePending

			// Check if the ZONE si AVAILABLE
			if !isRest {
				zoneObj := &v1beta1.AvailabilityZone{}
				zoneList := &v1beta1.AvailabilityZoneList{}
				if err := r.List(
					ctx,
					zoneList,
					client.InNamespace(appDeploy.Namespace),
					client.MatchingLabels{
						v1beta1.ResourceIdLabel: "zone-" + uuid.V5(appDeploy.Spec.ZoneId+appDeploy.Spec.FederationContextId),
					}); err != nil {
					return ctrl.Result{}, err
				}
				if len(zoneList.Items) == 0 {
					log.Info(">>> [AppOnboard] No Zone found for AppDeploy ", "name", appDeploy.Name, "naemspace", appDeploy.Namespace, "appId", appDeploy.Spec.AppId)
					return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
				}
				zoneObj = &zoneList.Items[0]
				if zoneObj.Status.State != v1beta1.ZoneStateAvailable {
					log.Info(">>> [AppOnboard] Zone is not AVAILABLE for ApplicationDeployment.", "name", appDeploy.Name, "namespace", appDeploy.Namespace, "appId", appDeploy.Spec.ZoneId, "state", zoneObj.Status.State)
					return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
				}
			}
			// checking if Application is Onboarded
			appOnboardObj := &v1beta1.ApplicationOnboarding{}
			appOnboardList := &v1beta1.ApplicationOnboardingList{}
			if err := r.List(
				ctx,
				appOnboardList,
				client.InNamespace(appDeploy.Namespace),
				client.MatchingLabels{
					v1beta1.ResourceIdLabel: "apponboard-" + uuid.V5(appDeploy.Spec.AppId+appDeploy.Spec.FederationContextId),
				}); err != nil {
				return ctrl.Result{}, err
			}
			if len(appOnboardList.Items) == 0 {
				log.Info(">>> [AppOnboard] No AppOnboard found for AppDeploy ", "name", appDeploy.Name, "naemspace", appDeploy.Namespace, "appId", appDeploy.Spec.AppId)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
			appOnboardObj = &appOnboardList.Items[0]
			if appOnboardObj.Status.State != v1beta1.ApplicationOnboardingStateOnboarded {
				log.Info(">>> [AppOnboard] ApplicationOnboarding is not ONBOARDED for ApplicationDeployment.", "name", appDeploy.Name, "namespace", appDeploy.Namespace, "appId", appDeploy.Spec.AppId, "state", appOnboardObj.Status.State)
				return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
			}
			if err := extClient.CreateApplicationDeployment(ctx, &appDeploy, fed); err != nil {
				log.Error(err, ">>> [AppDeploy] Error APPLYING/UPDATING SPEC ApplicationDeployment.", "name", appDeploy.Name, "namespace", appDeploy.Namespace)
				return ctrl.Result{}, err
			}
			if appDeploy.Labels == nil {
				appDeploy.Labels = make(map[string]string)
			}
			appDeploy.Labels[v1beta1.ResourceIdLabel] = "appdeploy-" + uuid.V5(appDeploy.Spec.FederationContextId+appDeploy.Spec.AppId+appDeploy.Spec.AppInstanceId)
		} else {
			if isRest {
				log.Info(">>> [AppDeploy][REST] Received UPDATEs via CALLBACK OPERATION with OPG EWBI API.", "name", appDeploy.Name, "namespace", appDeploy.Namespace)
			} else {
				if err := extClient.UpdateApplicationDeploymentStatus(ctx, &appDeploy, fed); err != nil {
					log.Error(err, ">>> [AppDeploy] Error updating ApplicationDeployment.", "name", appDeploy.Name, "namespace", appDeploy.Namespace)
					return ctrl.Result{}, err
				}
			}
		}
	}
	return ctrl.Result{}, nil
}
