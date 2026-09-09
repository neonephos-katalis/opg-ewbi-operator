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

// FederationReconciler reconciles a Federation object
type FederationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

func (r *FederationReconciler) CreateFederation(ctx context.Context, fed *v1beta1.Federation) error {
	fedHost := &v1beta1.Federation{
		TypeMeta: fed.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fed-" + uuid.V5(fed.Spec.FederationData.OrigOPFederationId+fed.Spec.FederationData.OrigOPCountryCode),
			Namespace: fed.Spec.FederationData.K8sOptions.Namespace,
		},
		Spec: v1beta1.FederationSpec{
			FederationData: &v1beta1.FederationData{
				OrigOPFederationId: fed.Spec.FederationData.OrigOPFederationId,
				InitialDate:        fed.Spec.FederationData.InitialDate,
				RelationType:       string(v1beta1.FederationRelationHost),
				TechnologyType:     string(v1beta1.FederationTechnologyK8s),
				K8sOptions: &v1beta1.K8sOptions{
					Namespace: fed.Spec.FederationData.K8sOptions.Namespace,
				},
			},
		},
	}
	err := ApplyRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		fedHost,
		&v1beta1.Federation{},
		fed.Name,
		fed.Namespace,
		v1beta1.GroupVersion.Group,
		v1beta1.GroupVersion.Version,
		v1beta1.PluralFederation,
		"federation-controller",
		"[Federation][K8s]",
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *FederationReconciler) PatchFederation(ctx context.Context, fed *v1beta1.Federation) error {
	fedHost := &v1beta1.Federation{
		TypeMeta: fed.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fed-" + uuid.V5(fed.Spec.FederationData.OrigOPFederationId+fed.Spec.FederationData.OrigOPCountryCode),
			Namespace: fed.Spec.FederationData.K8sOptions.Namespace,
		},
	}
	// DEVE AGGIUNGERE L'ANNOTAZIONE PER IL RINNOVO DELLA FEDERAZIONE LATO HOST

	if err := PatchRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		fedHost,
		fed.Name,
		fed.Namespace,
		v1beta1.GroupVersion.Group,
		v1beta1.GroupVersion.Version,
		v1beta1.PluralFederation,
		"[Federation][K8s]",
	); err != nil {
		return err
	}
	return nil
}

func (r *FederationReconciler) GetHealthFederation(ctx context.Context, fed *v1beta1.Federation) error {
	fedHost := &v1beta1.Federation{
		TypeMeta: fed.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fed-" + uuid.V5(fed.Spec.FederationData.OrigOPFederationId+fed.Spec.FederationData.OrigOPCountryCode),
			Namespace: fed.Spec.FederationData.K8sOptions.Namespace,
		},
	}
	err := AddRemoteAnnotation(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		fedHost,
		fed.Name,
		fed.Namespace,
		v1beta1.GroupVersion.Group,
		v1beta1.GroupVersion.Version,
		v1beta1.PluralFederation,
		"[Federation][K8s]",
		v1beta1.GetHealthInfoAnnotation,
		fed.Annotations[v1beta1.GetHealthInfoAnnotation],
	)
	if err != nil {
		return err
	}
	return nil
}
func (r *FederationReconciler) GetPlatformCapsFederation(ctx context.Context, fed *v1beta1.Federation) error {
	fedHost := &v1beta1.Federation{
		TypeMeta: fed.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fed-" + uuid.V5(fed.Spec.FederationData.OrigOPFederationId+fed.Spec.FederationData.OrigOPCountryCode),
			Namespace: fed.Spec.FederationData.K8sOptions.Namespace,
		},
	}
	err := AddRemoteAnnotation(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		fedHost,
		fed.Name,
		fed.Namespace,
		v1beta1.GroupVersion.Group,
		v1beta1.GroupVersion.Version,
		v1beta1.PluralFederation,
		"[Federation][K8s]",
		v1beta1.GetPlatformCapsAnnotation,
		fed.Annotations[v1beta1.GetPlatformCapsAnnotation],
	)
	if err != nil {
		return err
	}
	return nil
}
func (r *FederationReconciler) GetServiceAPIFederation(ctx context.Context, fed *v1beta1.Federation) error {
	fedHost := &v1beta1.Federation{
		TypeMeta: fed.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fed-" + uuid.V5(fed.Spec.FederationData.OrigOPFederationId+fed.Spec.FederationData.OrigOPCountryCode),
			Namespace: fed.Spec.FederationData.K8sOptions.Namespace,
		},
	}
	err := AddRemoteAnnotation(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		fedHost,
		fed.Name,
		fed.Namespace,
		v1beta1.GroupVersion.Group,
		v1beta1.GroupVersion.Version,
		v1beta1.PluralFederation,
		"[Federation][K8s]",
		v1beta1.GetServiceAPIsAnnotation,
		fed.Annotations[v1beta1.GetServiceAPIsAnnotation],
	)
	if err != nil {
		return err
	}
	return nil
}
func (r *FederationReconciler) RenewalFederation(ctx context.Context, fed *v1beta1.Federation) error {
	fedHost := &v1beta1.Federation{
		TypeMeta: fed.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fed-" + uuid.V5(fed.Spec.FederationData.OrigOPFederationId+fed.Spec.FederationData.OrigOPCountryCode),
			Namespace: fed.Spec.FederationData.K8sOptions.Namespace,
		},
	}
	err := AddRemoteAnnotation(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		fedHost,
		fed.Name,
		fed.Namespace,
		v1beta1.GroupVersion.Group,
		v1beta1.GroupVersion.Version,
		v1beta1.PluralFederation,
		"[Federation][K8s]",
		v1beta1.FederationRenewalAnnotation,
		fed.Annotations[v1beta1.FederationRenewalAnnotation],
	)
	if err != nil {
		return err
	}
	return nil
}

// WATCHER
func (r *FederationReconciler) UpdateFederationStatus(ctx context.Context, fed *v1beta1.Federation) error {
	fedHost := &v1beta1.Federation{}
	remoteName := "fed-" + uuid.V5(fed.Spec.FederationData.OrigOPFederationId+fed.Spec.FederationData.OrigOPCountryCode)
	if err := GetRemoteResource(ctx, r.Client, r.Scheme, fed, fedHost, remoteName, fed.Name, fed.Namespace, "[Federation][K8s]"); err != nil {
		return err
	}
	if len(fedHost.Status.ZoneDetails) != 0 {
		if CompareSameAZs(fed.Status.ZoneDetails, fedHost.Status.ZoneDetails) && fed.Status.State == v1beta1.FederationStateAvailable {
			return nil
		}
	}
	fed.Status = fedHost.Status
	return nil
}

// DeleteFederation
func (r *FederationReconciler) DeleteFederation(ctx context.Context, fed *v1beta1.Federation) error {
	remoteName := "fed-" + uuid.V5(fed.Spec.FederationData.OrigOPFederationId+fed.Spec.FederationData.OrigOPCountryCode)
	return DeleteRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		&v1beta1.Federation{},
		remoteName,
		fed.Name,
		fed.Namespace,
		"[Federation][K8s]",
	)
}

func (r *FederationReconciler) DetailsFederation(ctx context.Context, f *v1beta1.Federation) error {
	return nil
}
func (r *FederationReconciler) UpdateFederationDetailsStatus(ctx context.Context, f *v1beta1.Federation) error {
	return nil
}
