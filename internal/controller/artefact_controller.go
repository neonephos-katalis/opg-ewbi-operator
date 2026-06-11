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

// ArtefactReconciler reconciles a Artefact object
type ArtefactReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
	K8sClient  *k8s.ArtefactReconciler
	RestClient *rest.ArtefactReconciler
}

const (
	errorUpdatingArtefactStatusMsg = ">>> [Artefact] Error Updating resource status"
)

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
	log.Info(">>> [Artefact] Starting reconcile function for artefact")
	defer log.Info(">>> [Artefact] End reconcile for artefact")

	// Getting main artefact or requeue
	var a v1beta1.Artefact
	if err := r.Get(ctx, req.NamespacedName, &a); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info(">>> [Artefact] Object not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, ">>> [Artefact] Error getting artefact object")
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
		log.Error(err, ">>> [Artefact] An Artefact should always have a parent federation")
		a.Status.State = v1beta1.ArtefactStateError
		upErr := r.Status().Update(ctx, a.DeepCopy())
		if upErr != nil {
			log.Error(upErr, errorUpdatingArtefactStatusMsg)
		}
		return ctrl.Result{}, err
	}

	isRest := IsRestTechnology(feder.Labels)

	log.Info(">>> [Artefact] Object obtained", "name", feder.Name)

	if a.GetDeletionTimestamp().IsZero() {
		if controllerutil.AddFinalizer(&a, v1beta1.ArtefactFinalizer) {
			log.Info(">>> [Artefact] Added finalizer to Artefact")
			if err := r.Update(ctx, a.DeepCopy()); err != nil {
				log.Info(">>> [Artefact] Unable to update Artefact with finalizer")
				return ctrl.Result{}, err
			}
			log.Info(">>> [Artefact] Successfully added finalizer to Artefact")
			return ctrl.Result{}, nil
		}
	} else {
		if isGuest {
			if err := r.handleExternalArtefactDeletion(ctx, &a, feder, isRest); err != nil {
				log.Error(err, ">>> [Artefact] Error deleting Artefact")
				a.Status.State = v1beta1.ArtefactStateError
				upErr := r.Status().Update(ctx, a.DeepCopy())
				if upErr != nil {
					log.Error(upErr, errorUpdatingArtefactStatusMsg)
				}
				return ctrl.Result{}, err
			}
		}
		// if external Artefact is correctly deleted, we can remove the finalizer
		if controllerutil.RemoveFinalizer(&a, v1beta1.ArtefactFinalizer) {
			log.Info(">>>[Artefact] Removed basic finalizer for Artefact")
			if err := r.Update(ctx, a.DeepCopy()); err != nil {
				log.Error(err, ">>> [Artefact] Update failed while removing finalizers")
				return ctrl.Result{}, err
			}
			log.Info(">>> [Artefact] Removed all finalizers, exiting...")
			return ctrl.Result{}, nil
		}
	}

	// if federation is guest, send OPG API request
	if isGuest {
		if a.Status.State == "" {
			if err := r.handleExternalArtefactCreation(ctx, &a, feder, isRest); err != nil {
				log.Info(">>> [Artefact] Error creating Artefact")
				a.Status.State = v1beta1.ArtefactStateError
				upErr := r.Status().Update(ctx, a.DeepCopy())
				if upErr != nil {
					log.Error(upErr, errorUpdatingArtefactStatusMsg)
				}
				return ctrl.Result{}, nil
			}
		}
		if err := r.handelExternalArtefactUpdate(ctx, &a, feder, isRest); err != nil {
			log.Info(">>> [Artefact] Error updating Artefact")
			a.Status.State = v1beta1.ArtefactStateError
			upErr := r.Status().Update(ctx, a.DeepCopy())
			if upErr != nil {
				log.Error(upErr, errorUpdatingArtefactStatusMsg)
			}
			return ctrl.Result{}, nil
		}
	} else {
		if a.Status.State == "" {
			a.Status.State = v1beta1.ArtefactStateReconciling
			log.Info(">>> [Artefact] Initialized new CR state", "state", a.Status.State)
			upErr := r.Status().Update(ctx, a.DeepCopy())
			if upErr != nil {
				log.Error(upErr, errorUpdatingArtefactStatusMsg)
				return ctrl.Result{}, upErr
			}
		} else {
			if isRest {
				log.Info(">>> [Artefact] New CR state", "state", a.Status.State)
				if err := r.handleExternalArtefactCallback(ctx, &a, feder); err != nil {
					log.Error(err, "error handling artefact callback")
					a.Status.State = v1beta1.ArtefactStateError
					upErr := r.Status().Update(ctx, a.DeepCopy())
					if upErr != nil {
						log.Error(upErr, errorUpdatingArtefactStatusMsg)
					}
				}
			}
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ArtefactReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&opgewbiv1beta1.Artefact{}).
		Named("artefact").
		WatchesRawSource(
			source.Channel(
				k8s.ArtefactRemoteEvents,
				&handler.EnqueueRequestForObject{},
			),
		).
		Complete(r)
}

func (r *ArtefactReconciler) handelExternalArtefactUpdate(ctx context.Context, a *v1beta1.Artefact, feder *v1beta1.Federation, isRest bool) error {
	if isRest {
		log := log.FromContext(ctx)
		log.Info(">>> [Artefact] Current state is ", "state", a.Status.State)
		return nil
	} else {
		return r.K8sClient.SyncStatusWithHost(ctx, a, feder)
	}
}
func (r *ArtefactReconciler) handleExternalArtefactCreation(
	ctx context.Context, a *v1beta1.Artefact, feder *v1beta1.Federation, isRest bool,
) error {
	if isRest {
		return r.RestClient.CreateArtefact(ctx, a, feder)
	} else {
		return r.K8sClient.CreateArtefact(ctx, a, feder)
	}

}

func (r *ArtefactReconciler) handleExternalArtefactCallback(
	ctx context.Context, a *v1beta1.Artefact, feder *v1beta1.Federation,
) error {
	return r.RestClient.CallbackArtefact(ctx, a, feder)
}

func (r *ArtefactReconciler) handleExternalArtefactDeletion(
	ctx context.Context, a *v1beta1.Artefact, feder *v1beta1.Federation, isRest bool,
) error {
	if isRest {
		return r.RestClient.DeleteArtefact(ctx, a, feder)
	} else {
		return r.K8sClient.DeleteArtefact(ctx, a, feder)
	}
}
