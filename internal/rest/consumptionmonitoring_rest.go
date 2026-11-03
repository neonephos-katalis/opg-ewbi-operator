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

// ConsumptionMonitoringReconciler reconciles a ConsumptionMonitoring object
type ConsumptionMonitoringReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

// Create
func (r *ConsumptionMonitoringReconciler) CreateConsumptionMonitoring(ctx context.Context, rcm *v1beta1.ConsumptionMonitoring, fed *v1beta1.Federation) error {
	log := ctrl.Log
	log.Info(">>> [RCM][REST] CreateConsumptionMonitoring not implemented", "name", rcm.Name, "namespace", rcm.Namespace, "federation", fed.Name)
	return nil
}

// Delete
func (r *ConsumptionMonitoringReconciler) DeleteConsumptionMonitoring(ctx context.Context, rcm *v1beta1.ConsumptionMonitoring, fed *v1beta1.Federation) error {
	log := ctrl.Log
	log.Info(">>> [RCM][REST] DeleteConsumptionMonitoring not implemented", "name", rcm.Name, "namespace", rcm.Namespace)
	return nil
}

// Update (Callback)
func (r *ConsumptionMonitoringReconciler) UpdateConsumptionMonitoringStatus(ctx context.Context, rcm *v1beta1.ConsumptionMonitoring, fed *v1beta1.Federation) error {
	log := ctrl.Log
	log.Info(">>> [RCM][REST] UpdateConsumptionMonitoringStatus not implemented", "name", rcm.Name, "namespace", rcm.Namespace)
	return nil
}
