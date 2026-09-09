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
	zoneHome := &v1beta1.AvailabilityZone{
		TypeMeta: zone.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "zone-" + uuid.V5(zone.Spec.ZoneId+zone.Spec.FederationContextId),
			Namespace: fed.Spec.FederationData.K8sOptions.Namespace,
		},
		Spec: v1beta1.AvailabilityZoneSpec{
			RelationType:        string(v1beta1.FederationRelationHost),
			FederationContextId: fed.Status.FederationContextId,
			ZoneId:              zone.Spec.ZoneId,
		},
	}
	if err := ApplyRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		zoneHome,
		&v1beta1.AvailabilityZone{},
		zone.Name,
		zone.Namespace,
		v1beta1.GroupVersion.Group,
		v1beta1.GroupVersion.Version,
		v1beta1.PluralAvailabilityZone,
		"availabilityzone-controller",
		"[AZ][K8s]",
	); err != nil {
		return err
	}
	return nil
}

// Update
func (r *ZoneReconciler) UpdateZoneStatus(ctx context.Context, zone *v1beta1.AvailabilityZone, fed *v1beta1.Federation) error {
	zoneHost := &v1beta1.AvailabilityZone{}
	remoteName := "zone-" + uuid.V5(zone.Spec.ZoneId+zone.Spec.FederationContextId)
	if err := GetRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		zoneHost,
		remoteName,
		zone.Name,
		zone.Namespace,
		"[AZ][K8s]",
	); err != nil {
		return err
	}
	zone.Status = zoneHost.Status
	return nil
}

// Update (Watcher)
func (r *ZoneReconciler) DeleteZone(ctx context.Context, zone *v1beta1.AvailabilityZone, fed *v1beta1.Federation) error {
	remoteName := "zone-" + uuid.V5(zone.Spec.ZoneId+zone.Spec.FederationContextId)
	return DeleteRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed, &v1beta1.AvailabilityZone{},
		remoteName,
		zone.Name,
		zone.Namespace,
		"[AZ][K8s]",
	)

}
