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
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/k8s"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/rest"

	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// FederationReconciler reconciles a Federation object
type FederationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
	K8sClient  *k8s.FederationReconciler
	RestClient *rest.FederationReconciler
}

type ExternalFederationClient interface {
	CreateFederation(ctx context.Context, f *v1beta1.Federation) error
	DeleteFederation(ctx context.Context, f *v1beta1.Federation) error
	UpdateFederationStatus(ctx context.Context, f *v1beta1.Federation) error        //CALLBACK when REST (HOST Side), FUNCTION FOR THE WATCHER for K8s (GUEST Side)
	DetailsFederation(ctx context.Context, f *v1beta1.Federation) error             //POST /{federationContextId}/partner
	UpdateFederationDetailsStatus(ctx context.Context, f *v1beta1.Federation) error //Callback /{federationContextId}/partner
	PatchFederation(ctx context.Context, f *v1beta1.Federation) error
	GetHealthFederation(ctx context.Context, f *v1beta1.Federation) error
	GetPlatformCapsFederation(ctx context.Context, f *v1beta1.Federation) error
	GetServiceAPIFederation(ctx context.Context, f *v1beta1.Federation) error
	RenewalFederation(ctx context.Context, f *v1beta1.Federation) error
}

func (r *FederationReconciler) getExternalClient(isRest bool) ExternalFederationClient {
	if isRest {
		return r.RestClient
	}
	return r.K8sClient
}

// SetupWithManager sets up the controller with the Manager.
func (r *FederationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Federation{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldFed := e.ObjectOld.(*v1beta1.Federation)
				newFed := e.ObjectNew.(*v1beta1.Federation)
				if len(oldFed.GetFinalizers()) != len(newFed.GetFinalizers()) {
					return true
				}
				if newFed.Generation != oldFed.Generation {
					return true
				}
				if newFed.Annotations[v1beta1.FederationPolicyAnnotation] == "not-set" ||
					newFed.Annotations[v1beta1.FederationRenewalAnnotation] == "required" ||
					newFed.Annotations[v1beta1.GetHealthInfoAnnotation] == "required" ||
					newFed.Annotations[v1beta1.GetPlatformCapsAnnotation] == "required" ||
					newFed.Annotations[v1beta1.GetUpdateDetailsAnnotation] == "required" ||
					newFed.Annotations[v1beta1.UpdateDataAnnotation] == "required" {
					return true
				}
				isGuest := newFed.Spec.FederationData.RelationType == "GUEST"
				if !isGuest && oldFed.Status.State != newFed.Status.State {
					return true
				}
				if !isGuest && !reflect.DeepEqual(oldFed.Status.UpdateDetails, newFed.Status.UpdateDetails) {
					return true
				}
				return false
			},
		})).
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

// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=federations,verbs=*,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=federations/status,verbs=get;update;patch,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.katalis.com,resources=federations/finalizers,verbs=update,namespace=foo
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch,namespace=foo

func (r *FederationReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (res ctrl.Result, err error) {
	log := ctrl.Log
	log.Info(">>> [Federation] Starting RECONCILE FUNCTION", "name", req.Name, "namespace", req.Namespace)
	defer log.Info(">>> [Federation] End RECONCILE FUNCTION", "name", req.Name, "namespace", req.Namespace)

	// Getting main federation or requeue
	var fed v1beta1.Federation
	if err := r.Get(ctx, req.NamespacedName, &fed); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, ">>> [Federation] Error getting resource", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	isGuest := IsGuestResource(fed.Spec.FederationData.RelationType)
	isRest := IsRestTechnology(fed.Spec.FederationData.TechnologyType)
	extClient := r.getExternalClient(isRest) // Get the appropriate external client based on the federation technology

	if fed.Annotations == nil {
		fed.Annotations = make(map[string]string)
	}
	if fed.Labels == nil {
		fed.Labels = make(map[string]string)
	}
	annotations := fed.Annotations
	//Helper function to set the status to NotAvailable and update the resource
	originalFed := fed.DeepCopy()
	defer func() {
		isDeleting := !fed.GetDeletionTimestamp().IsZero()
		if err != nil && !isDeleting {
			log.Error(err, ">>> [Federation] UNEXPECTED ERROR detected in Reconcile, setting state to Failed before patching", "name", fed.Name, "namespace", fed.Namespace)
			fed.Status.State = v1beta1.FederationStateFailed
		}

		// Metadata Patch (Annotations, Labels, Finalizers)
		metaChanged := !reflect.DeepEqual(fed.Annotations, originalFed.Annotations) ||
			!reflect.DeepEqual(fed.Labels, originalFed.Labels) ||
			!reflect.DeepEqual(fed.Finalizers, originalFed.Finalizers)

		if metaChanged {
			currentStatus := fed.Status.DeepCopy()
			// Using Patch instead of Update to avoid overwriting changes made by other controllers
			if patchErr := r.Patch(ctx, &fed, client.MergeFrom(originalFed)); patchErr != nil {
				if !apierrors.IsNotFound(patchErr) {
					log.Error(patchErr, ">>> [Federation] UNEXPECTED ERROR during Federation Metadata UPDATE.", "name", fed.Name, "namespace", fed.Namespace)
				}
				if err == nil {
					err = patchErr
				}
				return // If there's an error patching metadata, we return early to avoid patching status with potentially inconsistent data
			}
			if currentStatus != nil {
				fed.Status = *currentStatus
			}
			// Alignment of the resource version after patching metadata
			originalFed.SetResourceVersion(fed.GetResourceVersion())
		}

		if isDeleting {
			return
		}

		// Status Update
		if patchErr := r.Status().Patch(ctx, &fed, client.MergeFrom(originalFed)); patchErr != nil {
			if !apierrors.IsNotFound(patchErr) {
				log.Error(patchErr, ">>> [Federation] UNEXPECTED ERROR during Federation Status UPDATE.", "name", fed.Name, "namespace", fed.Namespace)
			}
			if err == nil {
				err = patchErr
			}
		} else {
			log.Info(">>> [Federation] SUCCESSFULLY Reconciled.", "name", fed.Name, "namespace", fed.Namespace)
		}
	}()

	// Handle deletion of the federation resource
	if !fed.GetDeletionTimestamp().IsZero() {
		if isGuest {
			if err := extClient.DeleteFederation(ctx, &fed); err != nil {
				log.Error(err, ">>> [Federation] Error during the deletion.", "name", fed.Name, "namespace", fed.Namespace)
				return ctrl.Result{}, err
			}
		}
		if controllerutil.RemoveFinalizer(&fed, v1beta1.FederationFinalizer) {
			log.Info(">>> [Federation] Removed basic finalizer for Federation, exiting...", "name", fed.Name, "namespace", fed.Namespace)

		}
		// skipStatusPatch = true
		return ctrl.Result{}, nil
	}

	// Handle creation/finalizer
	if controllerutil.AddFinalizer(&fed, v1beta1.FederationFinalizer) {
		log.Info(">>> [Federation] Added finalizer to Federation", "name", fed.Name, "namespace", fed.Namespace)
		fed.Annotations[v1beta1.FederationPolicyAnnotation] = "not-set"
		if isGuest && !isRest {
			fed.Annotations[v1beta1.FederationWatcherAnnotation] = "not-stopped"
		}
		return ctrl.Result{}, nil
	}

	// Policy management for federation context ID
	if fed.Status.FederationContextId != "" && fed.Annotations[v1beta1.FederationPolicyAnnotation] == "not-set" {
		fed.Annotations[v1beta1.FederationPolicyAnnotation] = "set"
		return ctrl.Result{}, nil
	}

	isNewFed := fed.Status.State == ""
	watchers := []client.ObjectList{
		&v1beta1.ImageList{},
		&v1beta1.AvailabilityZoneList{},
		&v1beta1.ArtefactList{},
		&v1beta1.ApplicationDeploymentList{},
		&v1beta1.ApplicationOnboardingList{},
	}
	if !isGuest {
		// Host federation handling
		if isNewFed {
			fed.Status.FederationContextId = uuid.V5(fed.Spec.FederationData.OrigOPFederationId + fed.Spec.FederationData.InitialDate.String() + fed.Spec.FederationData.OrigOPCountryCode)
			fed.Labels[v1beta1.ResourceIdLabel] = "fed-" + uuid.V5(fed.Spec.FederationData.OrigOPFederationId+fed.Spec.FederationData.OrigOPCountryCode)
			// If the federation is new, we set the initial state to "AVAILABLE" and set the expiry and renewal dates based on the initial date provided in the spec. We also set the policy label to "false" to indicate that the policy has not been created yet.
			if fed.Status.FederationExpiryDate.IsZero() {
				fed.Status.FederationExpiryDate = metav1.NewTime(fed.Spec.FederationData.InitialDate.Add(24 * time.Hour))
			}
			// If the renewal date is zero, it means that the federation does not have a specific renewal date, so we set it to 24 hours after the initial date. If the current date matches the expiry date, it means that the federation has expired, so we set the state to "LOCKED". If the expiry date is updated, it means that the federation has been renewed, so we set the state to "AVAILABLE".
			if fed.Status.FederationRenewalDate.IsZero() {
				fed.Status.FederationRenewalDate = metav1.NewTime(fed.Spec.FederationData.InitialDate.Add(23 * time.Hour))
			}
			fed.Status.State = v1beta1.FederationStateAvailable
		} else {
			result, stop := r.federationExpiry(&fed)
			if stop {
				return result, nil
			}
			// If isRest is true, the host send a CALLBACK to the GUEST to update the federation status via OPG EWBI API
			if isRest {
				log.Info(">>> [Federation][REST] Execution CALLBACK OPERATION via OPG EWBI API", "name", fed.Name, "namespace", fed.Namespace)
				if err := extClient.UpdateFederationStatus(ctx, &fed); err != nil {
					log.Error(err, ">>> [Federation][REST] Error during CALLBACK OPERATION via OPG EWBI API.", "name", fed.Name, "namespace", fed.Namespace)
					return ctrl.Result{}, err
				}
			}
			if err := r.hostFederationActions(ctx, &fed, isRest, extClient, annotations); err != nil {
				return ctrl.Result{}, err
			}
			return result, nil
		}
	} else {
		// Guest federation handling
		// New Federation: Create it on the host
		if isNewFed {
			fed.Status.State = v1beta1.FederationStateNotAvailable
			fed.Labels[v1beta1.ResourceIdLabel] = "fed-" + uuid.V5(fed.Spec.FederationData.OrigOPFederationId+fed.Spec.FederationData.OrigOPCountryCode)
			if err := extClient.CreateFederation(ctx, &fed); err != nil {
				log.Error(err, ">>> [Federation] Error APPLYING/UPDATING SPEC Federation", "name", fed.Name, "namespace", fed.Namespace)
				return ctrl.Result{}, err
			}
		} else {
			switch fed.Status.State {
			case v1beta1.FederationStateLocked:
				log.Info(">>> [Federation] Federation is LOCKED", "name", fed.Name, "namespace", fed.Namespace)
				if fed.Annotations[v1beta1.FederationWatcherAnnotation] == "not-stopped" {
					if !isRest {
						if err := k8s.StopAllRemoteWatchers(ctx, r.Client, &fed, fed.Namespace, fed.Status.FederationContextId, watchers...); err != nil {
							log.Error(err, ">>> [Federation][K8s] Error STOPPING remote watchers.", "name", fed.Name, "namespace", fed.Namespace)
							return ctrl.Result{}, err
						}
					}
					fed.Annotations[v1beta1.FederationWatcherAnnotation] = "stopped"
				}
				return ctrl.Result{}, nil
			case v1beta1.FederationStateAvailable:
				log.Info(">>> [Federation] Federation is AVAILABLE.", "name", fed.Name, "namespace", fed.Namespace)
				if isRest {
					log.Info(">>> [Federation][REST] Received UPDATEs via CALLBACK OPERATION with OPG EWBI API.", "name", fed.Name, "namespace", fed.Namespace)
				} else {
					if fed.Annotations[v1beta1.FederationWatcherAnnotation] == "not-stopped" {
						if err := k8s.RestartAllRemoteWatcher(ctx, r.Client, &fed, r.Scheme, fed.Namespace, fed.Status.FederationContextId, watchers...); err != nil {
							log.Error(err, ">>> [Federation][K8s] Error RESTARTING remote watchers.", "name", fed.Name, "namespace", fed.Namespace)
							return ctrl.Result{}, err
						}
						fed.Annotations[v1beta1.FederationWatcherAnnotation] = "stopped"
						// if err := r.Update(ctx, &fed); err != nil {
						// 	log.Error(err, ">>> [Federation] Failed to update", "name", fed.Name, "namespace", fed.Namespace)
						// 	return ctrl.Result{}, err
						// }
						// skipStatusPatch = true
						return ctrl.Result{}, nil
					}
					log.Info(">>> [Federation][K8s] Syncing status...", "name", fed.Name, "namespace", fed.Namespace)
					if err := r.K8sClient.UpdateFederationStatus(ctx, &fed); err != nil {
						return ctrl.Result{}, err
					}
				}
			default:
				if isRest {
					log.Info(">>> [Federation][REST] Received UPDATEs via CALLBACK OPERATION with OPG EWBI API.", "name", fed.Name, "namespace", fed.Namespace)
				} else {
					log.Info(">>> [Federation][K8s] Syncing status...", "name", fed.Name, "namespace", fed.Namespace)
					if err := r.K8sClient.UpdateFederationStatus(ctx, &fed); err != nil {
						log.Error(err, ">>> [Federation][K8s] Error syncing status.", "name", fed.Name, "namespace", fed.Namespace)
						return ctrl.Result{}, err
					}
				}
				log.Info(">>> [Federation] Current state", "name", fed.Name, "namespace", fed.Namespace, "state", fed.Status.State)
			}
			if err := r.guestFederationActions(ctx, &fed, isRest, extClient, annotations); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *FederationReconciler) guestFederationActions(ctx context.Context, fed *v1beta1.Federation, isRest bool, extClient ExternalFederationClient, annotations map[string]string) error {
	log := ctrl.Log
	if fed.Status.State == v1beta1.FederationStateAvailable || fed.Status.State == v1beta1.FederationStateLocked {
		if _, exists := annotations[v1beta1.FederationRenewalAnnotation]; exists && annotations[v1beta1.FederationRenewalAnnotation] == "required" {
			log.Info(">>> [Federation][K8s] Renewal Federation", "name", fed.Name, "namespace", fed.Namespace)
			if err := extClient.RenewalFederation(ctx, fed); err != nil {
				log.Error(err, ">>> [Federation][K8s] Error (Renewal Federation).", "name", fed.Name, "namespace", fed.Namespace)
				return err
			}
			fed.Status.State = v1beta1.FederationStateNotAvailable
			fed.Annotations[v1beta1.FederationRenewalAnnotation] = "not-required"
		}
	}
	if fed.Status.State == v1beta1.FederationStateAvailable {
		if _, exists := annotations[v1beta1.UpdateDataAnnotation]; exists && fed.Annotations[v1beta1.UpdateDataAnnotation] == "required" {
			if isRest {
				// PATCH OPG EWBI API for UpdateData
				log.Info(">>> [Federation][REST] Applying UpdateDataRevision -> Performing PATCH operation", "name", fed.Name, "namespace", fed.Namespace)
			} else {
				log.Info(">>> [Federation][K8s] Applying UpdateDataRevision -> Performing K8s operation", "name", fed.Name, "namespace", fed.Namespace)
			}
			if err := extClient.PatchFederation(ctx, fed); err != nil {
				log.Error(err, ">>> [Federation] Error (UpdateFederationData)", "name", fed.Name, "namespace", fed.Namespace)
				return err
			}
			fed.Annotations[v1beta1.UpdateDataAnnotation] = "not-required"
		}
		if _, exists := annotations[v1beta1.GetHealthInfoAnnotation]; exists && fed.Annotations[v1beta1.GetHealthInfoAnnotation] == "required" {
			if isRest {
				// GET OPG EWBI API for HealthInfo
				log.Info(">>> [Federation][REST] Applying HealthCheckRevision -> Performing GET operation", "name", fed.Name, "namespace", fed.Namespace)
			} else {
				log.Info(">>> [Federation][K8s] Applying HealthCheckRevision -> Performing K8s operation", "name", fed.Name, "namespace", fed.Namespace)
			}
			if err := extClient.GetHealthFederation(ctx, fed); err != nil {
				log.Error(err, ">>> [Federation] Error (HealthCheckInfo)", "name", fed.Name, "namespace", fed.Namespace)
				return err
			}
			fed.Annotations[v1beta1.GetHealthInfoAnnotation] = "not-required"
		}
		if _, exists := annotations[v1beta1.GetPlatformCapsAnnotation]; exists && fed.Annotations[v1beta1.GetPlatformCapsAnnotation] == "required" {
			if isRest {
				//  GET OPG EWBI API for CapType
				log.Info(">>> [Federation][REST] Applying CapTypeRevision -> Performing GET operation", "name", fed.Name, "namespace", fed.Namespace)
			} else {
				log.Info(">>> [Federation][K8s] Applying CapTypeRevision -> Performing K8s operation", "name", fed.Name, "namespace", fed.Namespace)
			}
			if err := extClient.GetPlatformCapsFederation(ctx, fed); err != nil {
				log.Error(err, ">>> [Federation] Error (CapTypeInfo)", "name", fed.Name, "namespace", fed.Namespace)
				return err
			}
			fed.Annotations[v1beta1.GetPlatformCapsAnnotation] = "not-required"
		}
		if _, exists := annotations[v1beta1.GetUpdateDetailsAnnotation]; exists && fed.Annotations[v1beta1.GetUpdateDetailsAnnotation] == "required" {
			if isRest {
				// GET OPG EWBI API for SupportedServerAPI
				log.Info(">>> [Federation][REST] Applying SupportedServerAPIRevision -> Performing GET operation", "name", fed.Name, "namespace", fed.Namespace)
			} else {
				log.Info(">>> [Federation][K8s] Applying SupportedServerAPIRevision -> Performing K8s operation", "name", fed.Name, "namespace", fed.Namespace)
			}
			if err := extClient.GetServiceAPIFederation(ctx, fed); err != nil {
				log.Error(err, ">>> [Federation] Error (SupportedServerAPIInfo)", "name", fed.Name, "namespace", fed.Namespace)
				return err
			}
			fed.Annotations[v1beta1.GetUpdateDetailsAnnotation] = "not-required"
		}
	}
	return nil
}

func (r *FederationReconciler) hostFederationActions(ctx context.Context, fed *v1beta1.Federation, isRest bool, extClient ExternalFederationClient, annotations map[string]string) error {
	log := ctrl.Log
	if fed.Status.State == v1beta1.FederationStateAvailable || fed.Status.State == v1beta1.FederationStateLocked {
		if _, exists := annotations[v1beta1.FederationRenewalAnnotation]; exists && annotations[v1beta1.FederationRenewalAnnotation] == "required" {
			log.Info(">>> [Federation] Received renewal request for Federation", "name", fed.Name, "namespace", fed.Namespace)
		}
	}
	if fed.Status.State == v1beta1.FederationStateAvailable {
		if _, exists := annotations[v1beta1.UpdateDataAnnotation]; exists && fed.Annotations[v1beta1.UpdateDataAnnotation] == "required" {
			log.Info(">>> [Federation] Received update to UPDATE FEDERATION DATA", "name", fed.Name, "namespace", fed.Namespace)
		}
		if _, exists := annotations[v1beta1.GetHealthInfoAnnotation]; exists && fed.Annotations[v1beta1.GetHealthInfoAnnotation] == "required" {
			log.Info(">>> [Federation] Received request to GET HEALTH INFO", "name", fed.Name, "namespace", fed.Namespace)
		}
		if _, exists := annotations[v1beta1.GetPlatformCapsAnnotation]; exists && fed.Annotations[v1beta1.GetPlatformCapsAnnotation] == "required" {
			log.Info(">>> [Federation] Received request to GET PLATFORM CAPABILITIES", "name", fed.Name, "namespace", fed.Namespace)
		}
		if _, exists := annotations[v1beta1.GetUpdateDetailsAnnotation]; exists && fed.Annotations[v1beta1.GetUpdateDetailsAnnotation] == "required" {
			log.Info(">>> [Federation] Received request to GET SUPPORTED SERVER API", "name", fed.Name, "namespace", fed.Namespace)
		}
	}
	return nil
}

func (r *FederationReconciler) federationExpiry(fed *v1beta1.Federation) (ctrl.Result, bool) {
	// Before the operation, check if the federation becaome Locked,
	// For REST: set the state to LOCKED, send the callback to UPDATE the federation in the GUEST, and stop the future CALLBACKs
	// For K8s, set the state to LOCKED, the GUEST via WATCHER will be notified and will stop the future WATCHING
	log := ctrl.Log
	var result ctrl.Result
	now := time.Now()
	var requeueAfter time.Duration
	needsStatusUpdate := false

	// Check if the federation has an expiry date set and if the current time is after that date.
	if !fed.Status.FederationExpiryDate.IsZero() {
		if now.After(fed.Status.FederationExpiryDate.Time) {
			fed.Status.State = v1beta1.FederationStateLocked
			needsStatusUpdate = true
		} else if now.Before(fed.Status.FederationExpiryDate.Time) {
			timeUntilExpiry := fed.Status.FederationExpiryDate.Sub(now)
			if requeueAfter == 0 || timeUntilExpiry < requeueAfter {
				requeueAfter = timeUntilExpiry
			}
		}
	}

	// Check if the federation has a renewal date set and if the current time is after that date.
	// If so, we update the expiry date to be 24 hours after the renewal date and reset the renewal date.
	// If the current time is before the renewal date, we calculate the time until renewal and set the requeueAfter duration accordingly.
	if !fed.Status.FederationRenewalDate.IsZero() {
		if now.After(fed.Status.FederationRenewalDate.Time) {
			fed.Status.FederationExpiryDate = metav1.NewTime(fed.Status.FederationRenewalDate.Time.Add(24 * time.Hour))
			fed.Status.FederationRenewalDate = metav1.Time{}
			log.Info(">>> [Federation] Federation has been renewed, updating expiry date. ONLY ONE TIME ", "name", fed.Name, "namespace", fed.Namespace, "newExpiryDate", fed.Status.FederationExpiryDate)
		} else if now.Before(fed.Status.FederationRenewalDate.Time) {
			timeUntilRenewal := fed.Status.FederationRenewalDate.Time.Sub(now)
			if requeueAfter == 0 || timeUntilRenewal < requeueAfter {
				requeueAfter = timeUntilRenewal
			}
		}
	}

	// If the federation has expired and there is no renewal date set, we lock the federation and clear the expiry date.
	//  This ensures that the federation cannot be used until it is renewed.
	if fed.Status.FederationRenewalDate.IsZero() && now.After(fed.Status.FederationExpiryDate.Time) {
		fed.Status.State = v1beta1.FederationStateLocked
		fed.Status.FederationExpiryDate = metav1.Time{}
		return ctrl.Result{}, true // <-- Stop Reconcile
	}

	if needsStatusUpdate {
		return ctrl.Result{}, true // <-- Stop Reconcile
	}

	// Set the requeueAfter duration to ensure the next reconciliation happens after the calculated time
	if requeueAfter > 0 {
		requeueAfter += time.Second
		result.RequeueAfter = requeueAfter
		log.Info(">>> [Federation] Check Federation Expiry Data", "requeueAfter", requeueAfter)
	}

	// Other checks and actions can be added here as needed
	if !fed.Status.FederationExpiryDate.IsZero() && fed.Status.FederationExpiryDate.Time.Before(now) {
		fed.Status.State = v1beta1.FederationStateLocked
	}
	return result, false
}
