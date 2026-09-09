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

package rest

import (
	"context"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ZoneReconciler reconciles a Zone object
type ZoneReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

// Accept/Create
func (r *ZoneReconciler) AcceptZone(ctx context.Context, zone *v1beta1.AvailabilityZone, fed *v1beta1.Federation) error {
	log := ctrl.Log
	log.Info(">>> [AZ][REST] AcceptZone not implemented", "zone", zone.Name, "namespace", zone.Namespace, "federation", fed.Name)
	return nil
}

// Delete
func (r *ZoneReconciler) DeleteZone(ctx context.Context, zone *v1beta1.AvailabilityZone, fed *v1beta1.Federation) error {
	log := ctrl.Log
	log.Info(">>> [AZ][REST] DeleteZone not implemented", "name", zone.Name, "namespace", zone.Namespace)
	return nil
}

// Update (Callback)
func (r *ZoneReconciler) UpdateZoneStatus(ctx context.Context, zone *v1beta1.AvailabilityZone, fed *v1beta1.Federation) error {
	log := ctrl.Log
	log.Info(">>> [AZ][REST] UpdateZone not implemented", "name", zone.Name, "namespace", zone.Namespace)
	return nil
}
