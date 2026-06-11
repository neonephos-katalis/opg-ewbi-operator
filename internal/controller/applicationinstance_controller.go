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

// ApplicationInstanceReconciler reconciles a ApplicationInstance object
type ApplicationInstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
	K8sClient  *k8s.ApplicationInstanceReconciler
	RestClient *rest.ApplicationInstanceReconciler
}

const (
	errorUpdatingApplicationInstanceStatusMsg = ">>> [AppInst] Error Updating resource status"
)

// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=applicationinstances,verbs=*,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=applicationinstances/status,verbs=get;update;patch,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=applicationinstances/finalizers,verbs=update,namespace=foo

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Modify the Reconcile function to compare the state specified by
// the ApplicationInstance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *ApplicationInstanceReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("name", req.Name, "namespace", req.Namespace)
	log.Info(">>> [AppInst] starting reconcile function for appInst")
	defer log.Info(">>> [AppInst] end reconcile for appInst")

	// Getting main appInst or requeue
	var a v1beta1.ApplicationInstance
	if err := r.Get(ctx, req.NamespacedName, &a); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info(">>> [AppInst] Object not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, ">>> [AppInst] Error getting appInst object")
		return ctrl.Result{}, err
	}

	// Getting appInst's federation or requeue by using federation-context-id label
	// extraLabels := map[string]string{v1beta1.FederationRelationLabel: a.Labels[v1beta1.FederationRelationLabel]}
	isGuest := IsGuestResource(a.Labels)
	extraLabels := map[string]string{}
	if isGuest {
		extraLabels[v1beta1.FederationRelationLabel] = string(v1beta1.FederationRelationGuest)
	} else {
		extraLabels[v1beta1.FederationRelationLabel] = string(v1beta1.FederationRelationHost)
	}
	feder, err := GetFederationByContextId(ctx, r.Client, a.Labels[v1beta1.FederationContextIdLabel], extraLabels)
	if err != nil {
		log.Error(err, ">>> [AppInst] An ApplicattionInstance should always have a parent federation")
		a.Status.State = v1beta1.ApplicationInstanceStateFailed
		upErr := r.Status().Update(ctx, a.DeepCopy())
		if upErr != nil {
			log.Error(upErr, errorUpdatingApplicationInstanceStatusMsg)
		}
		return ctrl.Result{}, err
	}

	isRest := IsRestTechnology(feder.Labels)
	log.Info(">>> [AppInst] Federation object obtained", "name", feder.Name)

	if a.GetDeletionTimestamp().IsZero() {
		if controllerutil.AddFinalizer(&a, v1beta1.ApplicationInstanceFinalizer) {
			log.Info(">>> [AppInst] Added finalizer to appInst")
			if err := r.Update(ctx, a.DeepCopy()); err != nil {
				log.Info(">>> [AppInst] unable to Update appInst with finalizer")
				return ctrl.Result{}, err
			}
			log.Info(">>> [AppInst] Successfully added finalizer to appInst")
			return ctrl.Result{}, nil
		}
	} else {
		if isGuest {
			if err := r.handleExternalAppInstDeletion(ctx, &a, feder, isRest); err != nil {
				log.Error(err, ">>> [AppInst] Error deleting appInst")
				a.Status.State = v1beta1.ApplicationInstanceStateFailed
				upErr := r.Status().Update(ctx, a.DeepCopy())
				if upErr != nil {
					log.Error(upErr, errorUpdatingApplicationInstanceStatusMsg)
				}
				return ctrl.Result{}, err
			}
		}
		// if external appInst is correctly deleted, we can remove the finalizer
		if controllerutil.RemoveFinalizer(&a, v1beta1.ApplicationInstanceFinalizer) {
			log.Info(">>> [AppInst] Removed basic finalizer for appInst")
			if err := r.Update(ctx, a.DeepCopy()); err != nil {
				log.Error(err, ">>> [AppInst] Update failed while removing finalizers")
				return ctrl.Result{}, err
			}
			log.Info(">>> [AppInst] Removed all finalizers, exiting...")
			return ctrl.Result{}, nil
		}
	}

	// if federation is guest, send OPG API request
	if isGuest {
		if a.Status.State == "" {
			log.Info(">>> [AppInst] AppInst is in Pending state, getting access point info")
			if err := r.handleExternalAppInstCreation(ctx, &a, feder, isRest); err != nil {
				log.Error(err, ">>> [AppInst] Error creating appInst info before deletion")
				a.Status.State = v1beta1.ApplicationInstanceStateFailed
				upErr := r.Status().Update(ctx, a.DeepCopy())
				if upErr != nil {
					log.Error(upErr, errorUpdatingApplicationInstanceStatusMsg)
				}
				return ctrl.Result{}, err
			}
		}
		if err := r.handelExternalAppInstUpdate(ctx, &a, feder, isRest); err != nil {
			log.Info(">>> [AppInst] Error updating AppInst")
			a.Status.State = v1beta1.ApplicationInstanceStateFailed
			upErr := r.Status().Update(ctx, a.DeepCopy())
			if upErr != nil {
				log.Error(upErr, errorUpdatingApplicationInstanceStatusMsg)
			}
			return ctrl.Result{}, nil
		}
	} else {
		if a.Status.State == "" {
			a.Status.State = v1beta1.ApplicationInstanceStatePending
			a.Status.AppInstanceId = a.Labels[v1beta1.ExternalIdLabel]
			log.Info(">>> [AppInst] Initialized new CR state", "state", a.Status.State, "appInstanceId", a.Status.AppInstanceId)
			upErr := r.Status().Update(ctx, a.DeepCopy())
			if upErr != nil {
				log.Error(upErr, errorUpdatingApplicationInstanceStatusMsg)
				return ctrl.Result{}, upErr
			}
		} else {
			if isRest {
				log.Info(">>> [AppInst] New CR state", "state", a.Status.State)
				if err := r.handleExternalAppInstCallback(ctx, &a, feder); err != nil {
					log.Error(err, ">>> [AppInst] error handling appInst callback")
					a.Status.State = v1beta1.ApplicationInstanceStateFailed
					upErr := r.Status().Update(ctx, a.DeepCopy())
					if upErr != nil {
						log.Error(upErr, errorUpdatingApplicationInstanceStatusMsg)
					}
				}
			}
		}

	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&opgewbiv1beta1.ApplicationInstance{}).
		Named("applicationinstance").
		WatchesRawSource(
			source.Channel(
				k8s.ApplicationInstanceRemoteEvents,
				&handler.EnqueueRequestForObject{},
			),
		).
		Complete(r)
}

func (r *ApplicationInstanceReconciler) handelExternalAppInstUpdate(ctx context.Context, a *v1beta1.ApplicationInstance, feder *v1beta1.Federation, isRest bool) error {
	if isRest {
		log := log.FromContext(ctx)
		log.Info(">>> [AppInst][REST] Current state is ", "state", a.Status.State)
		return nil
	} else {
		return r.K8sClient.SyncStatusWithHost(ctx, a, feder)
	}
}

func (r *ApplicationInstanceReconciler) handleExternalAppInstCreation(
	ctx context.Context, a *v1beta1.ApplicationInstance, feder *v1beta1.Federation, isRest bool,
) error {
	if isRest {
		return r.RestClient.CreateApplicationInstance(ctx, a, feder)
	} else {
		return r.K8sClient.CreateApplicationInstance(ctx, a, feder)
	}

}

func (r *ApplicationInstanceReconciler) handleExternalAppInstCallback(
	ctx context.Context,
	a *v1beta1.ApplicationInstance,
	feder *v1beta1.Federation,
) error {
	return r.RestClient.CallbackApplicationInstance(ctx, a, feder)
}

func (r *ApplicationInstanceReconciler) handleExternalAppInstDeletion(
	ctx context.Context, appInst *v1beta1.ApplicationInstance, feder *v1beta1.Federation, isRest bool,
) error {
	if isRest {
		return r.RestClient.DeleteApplicationInstance(ctx, appInst, feder)
	} else {
		return r.K8sClient.DeleteApplicationInstance(ctx, appInst, feder)
	}
}
