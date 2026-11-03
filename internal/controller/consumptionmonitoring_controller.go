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

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	k8s "github.com/neonephos-katalis/opg-ewbi-operator/internal/k8s"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	rest "github.com/neonephos-katalis/opg-ewbi-operator/internal/rest"
	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/uuid"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ConsumptionMonitoringReconciler reconciles a ConsumptionMonitoring object
type ConsumptionMonitoringReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
	K8sClient  *k8s.ConsumptionMonitoringReconciler
	RestClient *rest.ConsumptionMonitoringReconciler
}
type ExternalConsumptionMonitoringClient interface {
	CreateConsumptionMonitoring(ctx context.Context, rcm *v1beta1.ConsumptionMonitoring, fed *v1beta1.Federation) error
	DeleteConsumptionMonitoring(ctx context.Context, rcm *v1beta1.ConsumptionMonitoring, fed *v1beta1.Federation) error
	UpdateConsumptionMonitoringStatus(ctx context.Context, rcm *v1beta1.ConsumptionMonitoring, fed *v1beta1.Federation) error //Callback for REST and GET for K8s
}

func (r *ConsumptionMonitoringReconciler) getExternalClient(isRest bool) ExternalConsumptionMonitoringClient {
	if isRest {
		return r.RestClient
	}
	return r.K8sClient
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConsumptionMonitoringReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.ConsumptionMonitoring{}).
		Named("consumptionmonitoring").
		WatchesRawSource(
			source.Channel(
				k8s.ConsumptionMonitoringRemoteEvents,
				&handler.EnqueueRequestForObject{},
			),
		).
		Complete(r)
}

// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=consumptionmonitorings,verbs=*,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=consumptionmonitorings/status,verbs=get;update;patch,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=consumptionmonitorings/finalizers,verbs=update,namespace=foo

func (r *ConsumptionMonitoringReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	log := ctrl.Log
	log.Info(">>> [RCM] Starting RECONCILE FUNCTION.", "name", req.Name, "namespace", req.Namespace)
	defer log.Info(">>> [RCM] End RECONCILE FUNCTION.", "name", req.Name, "namespace", req.Namespace)

	// Getting main ConsumptionMonitoring or requeue
	var rcm v1beta1.ConsumptionMonitoring
	if err := r.Get(ctx, req.NamespacedName, &rcm); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, ">>> [RCM] Error getting object.", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	//Helper function to set the status to Failed and update the resource
	originalRcm := rcm.DeepCopy()
	defer func() {
		isDeleting := !rcm.GetDeletionTimestamp().IsZero()
		if err != nil && !isDeleting {
			log.Error(err, ">>> [RCM] UNEXPECTED ERROR detected in Reconcile, setting state to Failed before patching", "name", rcm.Name, "namespace", rcm.Namespace)
			rcm.Status.State = v1beta1.ConsumptionMonitoringStateFailed
		}

		// Metadata Patch (Annotations, Labels, Finalizers)
		metaChanged := !reflect.DeepEqual(rcm.Annotations, originalRcm.Annotations) ||
			!reflect.DeepEqual(rcm.Labels, originalRcm.Labels) ||
			!reflect.DeepEqual(rcm.Finalizers, originalRcm.Finalizers)

		if metaChanged {
			currentStatus := rcm.Status.DeepCopy()
			// Using Patch instead of Update to avoid overwriting changes made by other controllers
			if patchErr := r.Patch(ctx, &rcm, client.MergeFrom(originalRcm)); patchErr != nil {
				if !apierrors.IsNotFound(patchErr) {
					log.Error(patchErr, ">>> [RCM] UNEXPECTED ERROR during ConsumptionMonitoring Metadata UPDATE.", "name", rcm.Name, "namespace", rcm.Namespace)
				}
				if err == nil {
					err = patchErr
				}
				return // If there's an error patching metadata, we return early to avoid patching status with potentially inconsistent data
			}
			if currentStatus != nil {
				rcm.Status = *currentStatus
			}
			// Alignment of the resource version after patching metadata
			originalRcm.SetResourceVersion(rcm.GetResourceVersion())
		}

		if isDeleting {
			return
		}

		// Status Update
		if patchErr := r.Status().Patch(ctx, &rcm, client.MergeFrom(originalRcm)); patchErr != nil {
			if !apierrors.IsNotFound(patchErr) {
				log.Error(patchErr, ">>> [RCM] UNEXPECTED ERROR during ConsumptionMonitoring Status UPDATE.", "name", rcm.Name, "namespace", rcm.Namespace)
			}
			if err == nil {
				err = patchErr
			}
		} else {
			if rcm.GetDeletionTimestamp().IsZero() {
				log.Info(">>> [RCM] SUCCESSFULLY Reconciled.", "name", rcm.Name, "namespace", rcm.Namespace)
			}
		}
	}()

	isGuest := IsGuestResource(rcm.Spec.RelationType)
	fed, isRest, err := GetFederation(ctx, isGuest, r.Client, rcm.Spec.FederationContextId, rcm.Namespace)
	extClient := r.getExternalClient(isRest) // Get the appropriate external client based on the federation technology
	if err != nil {
		log.Error(err, ">>> [RCM] Should always have a parent federation.", "name", rcm.Name, "namespace", rcm.Namespace)
		rcm.Status.State = v1beta1.ConsumptionMonitoringStateFailed
		return ctrl.Result{}, err
	}

	// Check if the federation is locked and stop the watcher if it is (K8s only) or stop the callbacks if it is (REST only)
	if !CheckFederationState(fed, isRest, "RCM", rcm.Name, rcm.Namespace) {
		return ctrl.Result{}, nil
	}

	// Handle deletion of the ConsumptionMonitoring resource
	if !rcm.GetDeletionTimestamp().IsZero() {
		if isGuest {
			if err := extClient.DeleteConsumptionMonitoring(ctx, &rcm, fed); err != nil {
				log.Error(err, ">>> [RCM] Error deleting ConsumptionMonitoring.", "name", rcm.Name, "namespace", rcm.Namespace)
				return ctrl.Result{}, err
			}
		}
		if controllerutil.RemoveFinalizer(&rcm, v1beta1.ConsumptionMonitoringFinalizer) {
			log.Info(">>> [RCM] Removed basic finalizer for ConsumptionMonitoring, exiting...", "name", rcm.Name, "namespace", rcm.Namespace)
		}
		return ctrl.Result{}, nil
	}

	// Handle creation/finalizer
	if controllerutil.AddFinalizer(&rcm, v1beta1.ConsumptionMonitoringFinalizer) {
		log.Info(">>> [RCM] Added finalizer to ConsumptionMonitoring.", "name", rcm.Name, "namespace", rcm.Namespace)
		return ctrl.Result{}, nil
	}

	if rcm.Labels == nil {
		rcm.Labels = make(map[string]string)
	}

	isNewRcm := rcm.Status.State == ""
	if !isGuest {
		// Host RCM handling
		if isNewRcm {
			rcm.Status.State = v1beta1.ConsumptionMonitoringStatePending
			rcm.Labels[v1beta1.ResourceIdLabel] = "rcm-" + uuid.V5(rcm.ResourceIdKey())
		} else {
			// Callback for REST and GET for K8s
			if isRest {
				if err := extClient.UpdateConsumptionMonitoringStatus(ctx, &rcm, fed); err != nil {
					log.Error(err, ">>> [RCM][REST] Error during CALLBACK OPERATION via OPG EWBI API.", "name", rcm.Name, "namespace", rcm.Namespace)
					return ctrl.Result{}, err
				}
			} else {
				log.Info(">>> [RCM][K8s] Resource updated (GUEST via watcher update through the resource)", "name", rcm.Name, "namespace", rcm.Namespace)
			}
		}
		return ctrl.Result{}, nil
	} else {
		// Guest RCM handling
		if isNewRcm {
			rcm.Status.State = v1beta1.ConsumptionMonitoringStatePending
			rcm.Labels[v1beta1.ResourceIdLabel] = "rcm-" + uuid.V5(rcm.ResourceIdKey())
			if !isRest && rcm.Spec.ResMonNotifLink != "" {
				log.Info(">>> [RCM][K8s] ResMonNotifLink is only used by REST federations, ignoring it.", "name", rcm.Name, "namespace", rcm.Namespace)
			}
			if err := extClient.CreateConsumptionMonitoring(ctx, &rcm, fed); err != nil {
				log.Error(err, ">>> [RCM] Error APPLYING/UPDATING SPEC ConsumptionMonitoring.", "name", rcm.Name, "namespace", rcm.Namespace)
				return ctrl.Result{}, err
			}
			log.Info(">>> [RCM] SUCCESSFULLY APPLIED SPEC AND SET INITIAL STATUS.", "name", rcm.Name, "namespace", rcm.Namespace)
		} else {
			if isRest {
				log.Info(">>> [RCM][REST] Received UPDATEs via CALLBACK OPERATION with OPG EWBI API.", "name", rcm.Name, "namespace", rcm.Namespace)
			} else {
				if err := extClient.UpdateConsumptionMonitoringStatus(ctx, &rcm, fed); err != nil {
					log.Error(err, ">>> [RCM][K8s] Error updating ConsumptionMonitoring.", "name", rcm.Name, "namespace", rcm.Namespace)
					return ctrl.Result{}, err
				}
			}
		}
	}
	return ctrl.Result{}, nil
}
