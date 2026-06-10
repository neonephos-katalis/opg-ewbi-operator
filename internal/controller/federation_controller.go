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
	k8s "github.com/neonephos-katalis/opg-ewbi-operator/internal/k8s"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	rest "github.com/neonephos-katalis/opg-ewbi-operator/internal/rest"
	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/uuid"
)

const (
	errorCreatingFederationMsg = "error creating federation"
)

// FederationReconciler reconciles a Federation object
type FederationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
	K8sClient  *k8s.FederationReconciler
	RestClient *rest.FederationReconciler
}

const (
	errorUpdatingFederationStatusMsg = ">>> [Federation] Error Updating resource status"
)

// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=federations,verbs=*,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=federations/status,verbs=get;update;patch,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=federations/finalizers,verbs=update,namespace=foo

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Modify the Reconcile function to compare the state specified by
// the Federation object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *FederationReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)

	log.Info(">>> [Federation] Starting reconcile function for federation")
	defer log.Info(">>> [Federation] End reconcile for federation")

	// Getting main federation or requeue
	var f v1beta1.Federation
	if err := r.Get(ctx, req.NamespacedName, &f); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info(">>> [Federation] Object not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "XXX [Federation] Error getting federation object")
		return ctrl.Result{}, err
	}
	log.Info(">>> [Federation] Object obtained", "name", f.Name, "originOP", f.Spec.OriginOP)
	isGuest := IsGuestResource(f.Labels)
	isRest := IsRestTechnology(f.Labels)
	log.Info(">>> [Federation] Resource type evaluation", "isGuest", isGuest, "isRest", isRest)
	if f.GetDeletionTimestamp().IsZero() {
		if controllerutil.AddFinalizer(&f, v1beta1.FederationFinalizer) {
			log.Info(">>> [Federation] Added finalizer to Federation")
			if err := r.Update(ctx, f.DeepCopy()); err != nil {
				log.Info(">>> [Federation] Unable to Update Federation with finalizer")
				return ctrl.Result{}, err
			}
			log.Info(">>> [Federation] Successfully added finalizer to federation")
			return ctrl.Result{}, nil
		}
	} else {
		if isGuest {
			if err := r.handleExternalFederationDeletion(ctx, &f, isRest); err != nil {
				log.Error(err, ">>> [Federation] Error deleting federation")
				f.Status.State = v1beta1.FederationStateNotAvailable
				upErr := r.Status().Update(ctx, f.DeepCopy())
				if upErr != nil {
					log.Error(upErr, errorUpdatingFederationStatusMsg)
				}
				return ctrl.Result{}, err
			}
		}
		// if external federation is correctly deleted, we can remove the finalizer
		if controllerutil.RemoveFinalizer(&f, v1beta1.FederationFinalizer) {
			log.Info(">>> [Federation] Removed basic finalizer for Federation")
			if err := r.Update(ctx, f.DeepCopy()); err != nil {
				log.Error(err, ">>> [Federation] Update failed while removing finalizers")
				return ctrl.Result{}, err
			}
			log.Info(">>> [Federation] Removed all finalizers, exiting...")
			return ctrl.Result{}, nil
		}
	}

	// if federation is guest, send OPG API request
	if isGuest {
		updated, err := r.handleExternalFederationCreation(ctx, &f, isRest)
		if err != nil {
			log.Error(err, errorCreatingFederationMsg)
			f.Status.State = v1beta1.FederationStateNotAvailable
			upErr := r.Status().Update(ctx, f.DeepCopy())
			if upErr != nil {
				log.Error(upErr, errorUpdatingFederationStatusMsg)
			}
			return ctrl.Result{}, err
		}
		if updated && isRest {
			// return, we will accept the AZ at the next reconcile
			return ctrl.Result{}, nil
		}
		if updated && !isRest {
			if err := r.K8sClient.SyncStatusWithHost(ctx, &f); err != nil {
				log.Error(err, ">>> [Federation] Error syncing status with host cluster")
				f.Status.State = v1beta1.FederationStateNotAvailable
				upErr := r.Status().Update(ctx, f.DeepCopy())
				if upErr != nil {
					log.Error(upErr, errorUpdatingFederationStatusMsg)
				}
			}
		}
		if err := r.handleAcceptExternalAZ(ctx, &f, isRest); err != nil {
			log.Error(err, ">>> [Federation] Error accepting AZ federation")
			f.Status.State = v1beta1.FederationStateNotAvailable
			upErr := r.Status().Update(ctx, f.DeepCopy())
			if upErr != nil {
				log.Error(upErr, errorUpdatingFederationStatusMsg)
			}
			return ctrl.Result{}, err
		}
	} else {
		if f.Status.State == "" {
			f.Status.State = v1beta1.FederationStateNotAvailable
			if !isRest {
				f.Status.FederationContextId = uuid.V5(f.Spec.GuestPartnerCredentials.ClientId)
			}
		}
		if f.Spec.AcceptedAvailabilityZones != nil {
			f.Status.State = v1beta1.FederationStateAvailable
		}
		upErr := r.Status().Update(ctx, f.DeepCopy())
		if upErr != nil {
			log.Error(upErr, errorUpdatingFederationStatusMsg)
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FederationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Federation{}).
		Named("federation").
		// Add watch on the channel receiving events from remote clusters, to trigger reconciliation when an event is received
		WatchesRawSource(
			source.Channel(
				k8s.FederationRemoteEvents,
				&handler.EnqueueRequestForObject{},
			),
		).
		Complete(r)
}

func (r *FederationReconciler) handleExternalFederationCreation(
	ctx context.Context, f *v1beta1.Federation, isRest bool) (statusChanged bool, err error) {
	if isRest {
		return r.RestClient.CreateFederation(ctx, f)
	} else {
		return r.K8sClient.CreateFederation(ctx, f)
	}
}

func (r *FederationReconciler) handleExternalFederationDeletion(
	ctx context.Context, f *v1beta1.Federation, isRest bool) error {
	if isRest {
		return r.RestClient.DeleteFederation(ctx, f)
	} else {
		return r.K8sClient.DeleteFederation(ctx, f)
	}
}

func (r *FederationReconciler) handleAcceptExternalAZ(ctx context.Context, f *v1beta1.Federation, isRest bool) error {
	if isRest {
		return r.RestClient.AcceptExternalAZ(ctx, f)
	} else {
		return r.K8sClient.AcceptExternalAZ(ctx, f)
	}
}
