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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	opgewbiv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	k8s "github.com/neonephos-katalis/opg-ewbi-operator/internal/k8s"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	rest "github.com/neonephos-katalis/opg-ewbi-operator/internal/rest"
)

const (
	errorUpdatingApplicationStatusMsg = ">>> [App] Error Updating application status"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
	K8sClient  *k8s.ApplicationReconciler
	RestClient *rest.ApplicationReconciler
}

// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=applications,verbs=*,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=applications/status,verbs=get;update;patch,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=applications/finalizers,verbs=update,namespace=foo

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Modify the Reconcile function to compare the state specified by
// the Application object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile
func (r *ApplicationReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("name", req.Name, "namespace", req.Namespace)
	log.Info(">>> [App] Starting reconcile function for app")
	defer log.Info("<<< [App] End reconcile for app")

	// Getting main app or requeue
	var a v1beta1.Application
	if err := r.Get(ctx, req.NamespacedName, &a); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("app object not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, ">>> [App] Error getting app object")
		return ctrl.Result{}, err
	}

	// Getting app's federation or requeue by using federation-context-id label
	isGuest := IsGuestResource(a.Labels)
	extraLabels := map[string]string{}
	if isGuest {
		extraLabels[v1beta1.FederationRelationLabel] = string(v1beta1.FederationRelationGuest)
	} else {
		extraLabels[v1beta1.FederationRelationLabel] = string(v1beta1.FederationRelationHost)
	}
	feder, err := GetFederationByContextId(ctx, r.Client, a.Labels[v1beta1.FederationContextIdLabel], extraLabels)
	if err != nil {
		log.Error(err, ">>> [App] An Applicattion should always have a parent federation")
		a.Status.State = v1beta1.ApplicationStateFailed
		upErr := r.Status().Update(ctx, a.DeepCopy())
		if upErr != nil {
			log.Error(upErr, errorUpdatingApplicationStatusMsg)
		}
		return ctrl.Result{}, err
	}

	isRest := IsRestTechnology(feder.Labels)
	log.Info(">>> [App] Federation object obtained", "name", feder.Name)

	if a.GetDeletionTimestamp().IsZero() {
		if controllerutil.AddFinalizer(&a, v1beta1.AppFinalizer) {
			log.Info(">>> [App] Added finalizer to app")
			if err := r.Update(ctx, a.DeepCopy()); err != nil {
				log.Info(">>> [App] Unable to Update app with finalizer")
				return ctrl.Result{}, err
			}
			log.Info(">>> [App] Successfully added finalizer to app")
			return ctrl.Result{}, nil
		}
	} else {
		if isGuest {
			if err := r.handleExternalAppDeletion(ctx, &a, feder, isRest); err != nil {
				log.Error(err, ">>> [App] Error deleting app")
				a.Status.State = v1beta1.ApplicationStateFailed
				upErr := r.Status().Update(ctx, a.DeepCopy())
				if upErr != nil {
					log.Error(upErr, errorUpdatingApplicationStatusMsg)
				}
				return ctrl.Result{}, err
			}
		}
		// if external app is correctly deleted, we can remove the finalizer
		if controllerutil.RemoveFinalizer(&a, v1beta1.AppFinalizer) {
			log.Info(">>> [App] Removed basic finalizer for app")
			if err := r.Update(ctx, a.DeepCopy()); err != nil {
				log.Error(err, ">>> [App] Update failed while removing finalizers")
				return ctrl.Result{}, err
			}
			log.Info(">>> [App] Removed all finalizers, exiting...")
			return ctrl.Result{}, nil
		}
	}

	// if federation is guest, send OPG API request
	if isGuest {
		if a.Status.State == "" {
			if err := r.handleExternalAppCreation(ctx, &a, feder, isRest); err != nil {
				log.Info(">>> [App] Error creating app")
				a.Status.State = v1beta1.ApplicationStateFailed
				upErr := r.Status().Update(ctx, a.DeepCopy())
				if upErr != nil {
					log.Error(upErr, errorUpdatingApplicationStatusMsg)
				}
				return ctrl.Result{}, nil
			}
		}
		if err := r.handleExternalAppUpdate(ctx, &a, feder, isRest); err != nil {
			log.Info(">>> [App] Error updating app")
			a.Status.State = v1beta1.ApplicationStateFailed
			upErr := r.Status().Update(ctx, a.DeepCopy())
			if upErr != nil {
				log.Error(upErr, errorUpdatingApplicationStatusMsg)
			}
			return ctrl.Result{}, nil
		}

	} else {
		if a.Status.State == "" {
			a.Status.State = v1beta1.ApplicationStatePending
			log.Info(">>> [App] Initialized new CR state", "state", a.Status.State)
			upErr := r.Status().Update(ctx, a.DeepCopy())
			if upErr != nil {
				log.Error(upErr, errorUpdatingApplicationStatusMsg)
				return ctrl.Result{}, upErr
			}
		} else {
			if isRest {
				log.Info(">>> [App] New CR state", "state", a.Status.State)
				if err := r.handleExternalAppCallback(ctx, &a, feder); err != nil {
					log.Error(err, ">>> [App] Error handling app callback")
					a.Status.State = v1beta1.ApplicationStateFailed
					upErr := r.Status().Update(ctx, a.DeepCopy())
					if upErr != nil {
						log.Error(upErr, errorUpdatingApplicationStatusMsg)
					}
				}
			}
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&opgewbiv1beta1.Application{}).
		Named("application").
		WatchesRawSource(
			source.Channel(
				k8s.ApplicationRemoteEvents,
				&handler.EnqueueRequestForObject{},
			),
		).
		Complete(r)
}

func (r *ApplicationReconciler) handleExternalAppUpdate(ctx context.Context, a *v1beta1.Application, feder *v1beta1.Federation, isRest bool) error {
	if isRest {
		log := log.FromContext(ctx)
		log.Info(">>> [App][REST] Current state is ", "state", a.Status.State)
		return nil
	} else {
		return r.K8sClient.SyncStatusWithHost(ctx, a, feder)
	}
}

func (r *ApplicationReconciler) handleExternalAppCreation(
	ctx context.Context, a *v1beta1.Application, feder *v1beta1.Federation, isRest bool,
) error {
	if isRest {
		return r.RestClient.CreateApplication(ctx, a, feder)
	} else {
		return r.K8sClient.CreateApplication(ctx, a, feder)
	}

}

func (r *ApplicationReconciler) handleExternalAppCallback(
	ctx context.Context, a *v1beta1.Application, feder *v1beta1.Federation,
) error {
	return r.RestClient.CallbackApplication(ctx, a, feder)
}

func (r *ApplicationReconciler) handleExternalAppDeletion(
	ctx context.Context, a *v1beta1.Application, feder *v1beta1.Federation, isRest bool,
) error {
	if isRest {
		return r.RestClient.DeleteApplication(ctx, a, feder)
	} else {
		return r.K8sClient.DeleteApplication(ctx, a, feder)
	}
}
