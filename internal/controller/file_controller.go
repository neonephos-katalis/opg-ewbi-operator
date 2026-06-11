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

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/indexer"
	k8s "github.com/neonephos-katalis/opg-ewbi-operator/internal/k8s"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	rest "github.com/neonephos-katalis/opg-ewbi-operator/internal/rest"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// FileReconciler reconciles a File object
type FileReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
	K8sClient  *k8s.FileReconciler
	RestClient *rest.FileReconciler
}

const (
	errorUpdatingFileStatusMsg = ">>> [File] Error Updating resource status"
	unexpectedStatusFileMsg    = ">>> [File] Unexpected Status Code"
)

// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=files,verbs=*,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=files/status,verbs=get;update;patch,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=files/finalizers,verbs=update,namespace=foo

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Modify the Reconcile function to compare the state specified by
// the File object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile
func (r *FileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("name", req.Name, "namespace", req.Namespace)
	log.Info(">>> [File] starting reconcile function for file")
	defer log.Info(">>> [File] end reconcile for file")

	// Getting main file or requeue
	var f v1beta1.File
	if err := r.Get(ctx, req.NamespacedName, &f); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info(">>>> [File] Object not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, ">>> [File] Error getting file object")
		return ctrl.Result{}, err
	}
	log.Info(">>> [File] Object obtained", "name", f.Spec.FileName, "version", f.Spec.FileVersion)

	// Getting file's federation or requeue by using federation-context-id label
	isGuest := IsGuestResource(f.Labels)
	extraLabels := map[string]string{}
	if isGuest {
		extraLabels[v1beta1.FederationRelationLabel] = string(v1beta1.FederationRelationGuest)
	} else {
		extraLabels[v1beta1.FederationRelationLabel] = string(v1beta1.FederationRelationHost)
	}

	feder, err := GetFederationByContextId(ctx, r.Client, f.Labels[v1beta1.FederationContextIdLabel], extraLabels)
	if err != nil {
		log.Error(err, ">>> [File] Should always have a parent federation")
		f.Status.State = v1beta1.FileStateError
		upErr := r.Status().Update(ctx, f.DeepCopy())
		if upErr != nil {
			log.Error(upErr, errorUpdatingFileStatusMsg)
		}
		return ctrl.Result{}, err
	}

	isRest := IsRestTechnology(feder.Labels)

	log.Info(">>> [File] Object obtained", "name", feder.Name)

	if f.GetDeletionTimestamp().IsZero() {
		if controllerutil.AddFinalizer(&f, v1beta1.FileFinalizer) {
			log.Info(">>> [File] Added finalizer to File")
			if err := r.Update(ctx, f.DeepCopy()); err != nil {
				log.Info(">>> [File] Unable to Update File with finalizer")
				return ctrl.Result{}, err
			}
			log.Info(">>> [File] Successfully added finalizer to file")
			return ctrl.Result{}, nil
		}
	} else {
		if isGuest {
			if err := r.handleExternalFileDeletion(ctx, &f, feder, isRest); err != nil {
				log.Error(err, ">>> [File] Error deleting file")
				f.Status.State = v1beta1.FileStateError
				upErr := r.Status().Update(ctx, f.DeepCopy())
				if upErr != nil {
					log.Error(upErr, errorUpdatingFileStatusMsg)
				}
				return ctrl.Result{}, err
			}
		}
		// if external file is correctly deleted, we can remove the finalizer
		if controllerutil.RemoveFinalizer(&f, v1beta1.FileFinalizer) {
			log.Info(">>> [File] Removed basic finalizer for File")
			if err := r.Update(ctx, f.DeepCopy()); err != nil {
				//log.Error(err, "update failed while removing finalizers") //Commented to reduce log noise
				return ctrl.Result{}, nil
			}
			log.Info(">>> [File] Removed all finalizers, exiting...")
			return ctrl.Result{}, nil
		}
	}

	// if federation is guest, send OPG API request
	if isGuest {
		if f.Status.State == "" {
			if err := r.handleExternalFileCreation(ctx, &f, feder, isRest); err != nil {
				log.Info(">>> [File] Error creating file")
				f.Status.State = v1beta1.FileStateError
				upErr := r.Status().Update(ctx, f.DeepCopy())
				if upErr != nil {
					log.Error(upErr, errorUpdatingFileStatusMsg)
				}
				return ctrl.Result{}, nil
			}
		}
		if err := r.handelExternalFileUpdate(ctx, &f, feder, isRest); err != nil {
			log.Info(">>> [File] Error updating File")
			f.Status.State = v1beta1.FileStateError
			upErr := r.Status().Update(ctx, f.DeepCopy())
			if upErr != nil {
				log.Error(upErr, errorUpdatingFileStatusMsg)
			}
			return ctrl.Result{}, nil
		}
	} else {
		if f.Status.State == "" {
			f.Status.State = v1beta1.FileStatePending
			log.Info(">>> [File] Initialized new CR state", "state", f.Status.State)
			upErr := r.Status().Update(ctx, f.DeepCopy())
			if upErr != nil {
				log.Error(upErr, errorUpdatingFileStatusMsg)
			}
		} else {
			if isRest {
				log.Info(">>> [File] Current CR state", "state", f.Status.State)
				if err := r.handleExternalFileCallback(ctx, &f, feder); err != nil {
					log.Error(err, ">>> [File] Error handling file callback")
					f.Status.State = v1beta1.FileStateError
					upErr := r.Status().Update(ctx, f.DeepCopy())
					if upErr != nil {
						log.Error(upErr, errorUpdatingFileStatusMsg)
					}
				}
			}
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := indexer.GetFederationIndexers(context.Background(), mgr); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.File{}).
		Named("file").
		WatchesRawSource(
			source.Channel(
				k8s.FileRemoteEvents,
				&handler.EnqueueRequestForObject{},
			),
		).
		Complete(r)
}

func (r *FileReconciler) handelExternalFileUpdate(ctx context.Context, f *v1beta1.File, feder *v1beta1.Federation, isRest bool) error {
	if isRest {
		log := log.FromContext(ctx)
		log.Info(">>> [File][REST] Current state is ", "state", f.Status.State)
		return nil
	} else {
		return r.K8sClient.SyncStatusWithHost(ctx, f, feder)
	}
}

func (r *FileReconciler) handleExternalFileCreation(
	ctx context.Context, f *v1beta1.File, feder *v1beta1.Federation, isRest bool,
) error {
	if isRest {
		return r.RestClient.CreateFile(ctx, f, feder)
	} else {
		return r.K8sClient.CreateFile(ctx, f, feder)
	}
}

func (r *FileReconciler) handleExternalFileCallback(
	ctx context.Context, f *v1beta1.File, feder *v1beta1.Federation,
) error {
	return r.RestClient.CallbackFile(ctx, f, feder)
}

func (r *FileReconciler) handleExternalFileDeletion(
	ctx context.Context, f *v1beta1.File, feder *v1beta1.Federation, isRest bool,
) error {
	if isRest {
		return r.RestClient.DeleteFile(ctx, f, feder)
	} else {
		return r.K8sClient.DeleteFile(ctx, f, feder)
	}
}
