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

// AvailabilityZoneReconciler reconciles a AvailabilityZone object
type ZoneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
	RestClient *rest.ZoneReconciler
}

type ExternalAzClient interface {
	AcceptZone(ctx context.Context, az *v1beta1.AvailabilityZone, fed *v1beta1.Federation) error
	DeleteZone(ctx context.Context, az *v1beta1.AvailabilityZone, fed *v1beta1.Federation) error
	UpdateZoneStatus(ctx context.Context, az *v1beta1.AvailabilityZone, fed *v1beta1.Federation) error //Callback for REST and GET for K8s
}

func (r *ZoneReconciler) getExternalClient(isRest bool) ExternalAzClient {
	if isRest {
		return r.RestClient
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ZoneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.AvailabilityZone{}).
		Named("availabilityzone").
		Complete(r)
}

// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=availabilityzones,verbs=*,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=availabilityzones/status,verbs=get;update;patch,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=availabilityzones/finalizers,verbs=update,namespace=foo

func (r *ZoneReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (res ctrl.Result, err error) {
	log := ctrl.Log
	log.Info(">>> [AZ] Starting RECONCILE FUNCTION.", "name", req.Name, "namespace", req.Namespace)
	defer log.Info(">>> [AZ] End RECONCILE FUNCTION.", "name", req.Name, "namespace", req.Namespace)

	// Getting main AZ or requeue
	var zone v1beta1.AvailabilityZone
	if err := r.Get(ctx, req.NamespacedName, &zone); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, ">>> [AZ] Error getting resource.", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	//Helper function to set the status to NotAvailable and update the resource
	originalZone := zone.DeepCopy()
	defer func() {
		isDeleting := !zone.GetDeletionTimestamp().IsZero()
		if err != nil && !isDeleting {
			log.Error(err, ">>> [AZ] UNEXPECTED ERROR detected in Reconcile, setting state to Failed before patching", "name", zone.Name, "namespace", zone.Namespace)
			zone.Status.State = v1beta1.ZoneStateFailed
		}

		// Metadata Patch (Annotations, Labels, Finalizers)
		metaChanged := !reflect.DeepEqual(zone.Annotations, originalZone.Annotations) ||
			!reflect.DeepEqual(zone.Labels, originalZone.Labels) ||
			!reflect.DeepEqual(zone.Finalizers, originalZone.Finalizers)

		if metaChanged {
			currentStatus := zone.Status.DeepCopy()
			// Using Patch instead of Update to avoid overwriting changes made by other controllers
			if patchErr := r.Patch(ctx, &zone, client.MergeFrom(originalZone)); patchErr != nil {
				if !apierrors.IsNotFound(patchErr) {
					log.Error(patchErr, ">>> [AZ] UNEXPECTED ERROR during AZ Metadata UPDATE.", "name", zone.Name, "namespace", zone.Namespace)
				}
				if err == nil {
					err = patchErr
				}
				return // If there's an error patching metadata, we return early to avoid patching status with potentially inconsistent data
			}
			if currentStatus != nil {
				zone.Status = *currentStatus
			}
			// Alignment of the resource version after patching metadata
			originalZone.SetResourceVersion(zone.GetResourceVersion())
		}
		if isDeleting {
			return
		}
		// Status Update
		if patchErr := r.Status().Patch(ctx, &zone, client.MergeFrom(originalZone)); patchErr != nil {
			if !apierrors.IsNotFound(patchErr) {
				log.Error(patchErr, ">>> [AZ] UNEXPECTED ERROR during AZ Status UPDATE.", "name", zone.Name, "namespace", zone.Namespace)
			}
			if err == nil {
				err = patchErr
			}
		} else {
			log.Info(">>> [AZ] SUCCESSFULLY Reconciled.", "name", zone.Name, "namespace", zone.Namespace)
		}
	}()

	isGuest := IsGuestResource(zone.Spec.RelationType)
	fed, isRest, err := GetFederation(ctx, isGuest, r.Client, zone.Spec.FederationContextId, zone.Namespace)
	extClient := r.getExternalClient(isRest)
	if err != nil {
		log.Error(err, ">>> [AZ] Should always have a parent federation.", "name", zone.Name, "namespace", zone.Namespace)
		return ctrl.Result{}, err
	}

	// Check if the federation is locked and stop the watcher if it is (K8s only) or stop the callbacks if it is (REST only)
	if !CheckFederationState(fed, isRest, "AZ", zone.Name, zone.Namespace) {
		return ctrl.Result{}, nil
	}

	// Handle deletion of the AZ resource
	if !zone.GetDeletionTimestamp().IsZero() {
		if isGuest {
			if err := extClient.DeleteZone(ctx, &zone, fed); err != nil {
				log.Error(err, ">>> [AZ] Error deleting external AZ.", "name", zone.Name, "namespace", zone.Namespace)
				return ctrl.Result{}, err
			}
			if controllerutil.RemoveFinalizer(&zone, v1beta1.AvailabilityZoneFinalizer) {
				log.Info(">>> [AZ] Removed basic finalizer for AZ, exiting...", "name", zone.Name, "namespace", zone.Namespace)
			}
		}
		return ctrl.Result{}, nil
	}

	// Handle creation/finalizer
	if controllerutil.AddFinalizer(&zone, v1beta1.AvailabilityZoneFinalizer) {
		log.Info(">>> [AZ] Added finalizer to AZ", "name", zone.Name, "namespace", zone.Namespace)
		return ctrl.Result{}, nil
	}

	if zone.Labels == nil {
		zone.Labels = make(map[string]string)
	}

	isNewZone := zone.Status.State == ""
	if !isGuest {
		// Host AZ handling
		if isNewZone {
			zone.Status.State = v1beta1.ZoneStateNotAvailable
			zone.Labels[v1beta1.ResourceIdLabel] = "zone-" + uuid.V5(zone.Spec.ZoneId+zone.Spec.FederationContextId)
		} else {
			if isRest {
				//CALLBACK from REST
				if err := extClient.UpdateZoneStatus(ctx, &zone, fed); err != nil {
					log.Error(err, ">>> [AZ][REST] Error during CALLBACK OPERATION via OPG EWBI API.", "name", zone.Name, "namespace", zone.Namespace)
					return ctrl.Result{}, err
				}
			} else {
				log.Info(">>> [AZ][K8s] Resource updated (GUEST via watcher update through the resource)", "name", zone.Name, "namespace", zone.Namespace)
			}
		}
	} else {
		// Guest AZ handling
		if isNewZone {
			zone.Status.State = v1beta1.ZoneStateNotAvailable
			zone.Labels[v1beta1.ResourceIdLabel] = "zone-" + uuid.V5(zone.Spec.ZoneId+zone.Spec.FederationContextId)
			if err := extClient.AcceptZone(ctx, &zone, fed); err != nil {
				log.Error(err, ">>> [AZ] Error accepting Zone", "name", zone.Name, "namespace", zone.Namespace)
				return ctrl.Result{}, err
			}
			log.Info(">>> [AZ] SUCCESSFULLY APPLIED SPEC AND SET INITIAL STATUS.", "name", zone.Name, "namespace", zone.Namespace)
		} else {
			if isRest {
				log.Info(">>> [AZ][REST] Received UPDATEs via CALLBACK OPERATION with OPG EWBI API", "name", zone.Name, "namespace", zone.Namespace)
			} else {
				// WATCHER for K8s
				if err := extClient.UpdateZoneStatus(ctx, &zone, fed); err != nil {
					log.Error(err, ">>> [AZ][K8s] Error updating Zone.", "name", zone.Name, "namespace", zone.Namespace)
					return ctrl.Result{}, err
				}
			}
		}
	}
	return ctrl.Result{}, nil
}
