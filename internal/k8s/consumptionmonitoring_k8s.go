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

package k8s

import (
	"context"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// CreateConsumptionMonitoring
func (r *ConsumptionMonitoringReconciler) CreateConsumptionMonitoring(ctx context.Context, rcm *v1beta1.ConsumptionMonitoring, fed *v1beta1.Federation) error {
	rcmHost := &v1beta1.ConsumptionMonitoring{
		TypeMeta: rcm.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rcm-" + uuid.V5(rcm.ResourceIdKey()),
			Namespace: fed.Spec.FederationData.K8sOptions.Namespace,
		},
		Spec: v1beta1.ConsumptionMonitoringSpec{
			RelationType:        string(v1beta1.FederationRelationHost),
			FederationContextId: fed.Status.FederationContextId,
			ZoneId:              rcm.Spec.ZoneId,
			MonType:             rcm.Spec.MonType,
			Periodicity:         rcm.Spec.Periodicity.DeepCopy(),
			Application:         rcm.Spec.Application.DeepCopy(),
		},
	}
	if err := ApplyRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		rcmHost,
		&v1beta1.ConsumptionMonitoring{},
		rcm.Name,
		rcm.Namespace,
		v1beta1.GroupVersion.Group,
		v1beta1.GroupVersion.Version,
		v1beta1.PluralConsumptionMonitoring,
		"consumptionmonitoring-controller",
		"[RCM][K8s]",
	); err != nil {
		return err
	}
	return nil
}

// UpdateConsumptionMonitoringStatus
func (r *ConsumptionMonitoringReconciler) UpdateConsumptionMonitoringStatus(ctx context.Context, rcm *v1beta1.ConsumptionMonitoring, fed *v1beta1.Federation) error {
	log := ctrl.Log
	rcmHost := &v1beta1.ConsumptionMonitoring{}
	remoteName := "rcm-" + uuid.V5(rcm.ResourceIdKey())
	if err := GetRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		rcmHost,
		remoteName,
		rcm.Name,
		rcm.Namespace,
		"[RCM][K8s]",
	); err != nil {
		return err
	}
	if rcmHost.Status.State == "" {
		log.Info(">>> [RCM][K8s] Remote status not initialized yet, skipping.", "name", rcm.Name, "namespace", rcm.Namespace)
		return nil
	}
	rcm.Status = rcmHost.Status
	return nil
}

// DeleteConsumptionMonitoring
func (r *ConsumptionMonitoringReconciler) DeleteConsumptionMonitoring(ctx context.Context, rcm *v1beta1.ConsumptionMonitoring, fed *v1beta1.Federation) error {
	remoteName := "rcm-" + uuid.V5(rcm.ResourceIdKey())
	return DeleteRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		&v1beta1.ConsumptionMonitoring{},
		remoteName,
		rcm.Name,
		rcm.Namespace,
		"[RCM][K8s]",
	)
}
