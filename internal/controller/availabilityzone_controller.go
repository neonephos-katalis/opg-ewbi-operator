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
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/nbycomp/neonephos-opg-ewbi-operator/api/v1beta1"
	opgewbiv1beta1 "github.com/nbycomp/neonephos-opg-ewbi-operator/api/v1beta1"
	"github.com/nbycomp/neonephos-opg-ewbi-operator/internal/opg"
)

// AvailabilityZoneReconciler reconciles a AvailabilityZone object
type AvailabilityZoneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=availabilityzones,verbs=*,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=availabilityzones/status,verbs=get;update;patch,namespace=foo
// +kubebuilder:rbac:groups=opg.ewbi.nby.one,resources=availabilityzones/finalizers,verbs=update,namespace=foo

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AvailabilityZone object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile
func (r *AvailabilityZoneReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("starting reconcile function for az", "name", req.Name, "namespace", req.Namespace)
	defer log.Info("end reconcile for az", "name", req.Name, "namespace", req.Namespace)

	// Getting main AZ or requeue
	var az v1beta1.AvailabilityZone
	if err := r.Get(ctx, req.NamespacedName, &az); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("AZ object not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "error getting AZ object")
		return ctrl.Result{}, err
	}
	log.Info("AZ object obtained", "name", az.Name)

	az.Status.Phase = v1beta1.AvailabilityZonePhaseReady
	upErr := r.Status().Update(ctx, az.DeepCopy())
	if upErr != nil {
		log.Error(upErr, errorUpdatingResourceStatusMsg)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AvailabilityZoneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&opgewbiv1beta1.AvailabilityZone{}).
		Named("availabilityzone").
		Complete(r)
}
