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

// ApplicationOnboardingReconciler reconciles an Artefact object
type ApplicationOnboardingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

func (r *ApplicationOnboardingReconciler) CreateApplicationOnboarding(ctx context.Context, appOnboard *v1beta1.ApplicationOnboarding, fed *v1beta1.Federation) error {
	appOnboardHost := &v1beta1.ApplicationOnboarding{
		TypeMeta: appOnboard.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apponboard-" + uuid.V5(appOnboard.Spec.AppInfo.AppId+appOnboard.Spec.FederationContextId),
			Namespace: fed.Spec.FederationData.K8sOptions.Namespace,
		},
		Spec: v1beta1.ApplicationOnboardingSpec{
			RelationType:        string(v1beta1.FederationRelationHost),
			FederationContextId: fed.Status.FederationContextId,
			AppInfo:             appOnboard.Spec.AppInfo,
		},
	}
	err := ApplyRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		appOnboardHost,
		&v1beta1.ApplicationOnboarding{},
		appOnboard.Name,
		appOnboard.Namespace,
		v1beta1.GroupVersion.Group,
		v1beta1.GroupVersion.Version,
		v1beta1.PluralApplicationOnboarding,
		"app-onboard-controller",
		"[AppOnboard][K8s]",
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *ApplicationOnboardingReconciler) UpdateApplicationOnboardingStatus(ctx context.Context, appOnboard *v1beta1.ApplicationOnboarding, fed *v1beta1.Federation) error {
	appOnboardHost := &v1beta1.ApplicationOnboarding{}
	remoteName := "apponboard-" + uuid.V5(appOnboard.Spec.AppInfo.AppId+appOnboard.Spec.FederationContextId)
	if err := GetRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		appOnboardHost,
		remoteName,
		appOnboard.Name,
		appOnboard.Namespace,
		"[AppOnboard][K8s]",
	); err != nil {
		return err
	}
	appOnboard.Status = appOnboardHost.Status
	return nil
}

func (r *ApplicationOnboardingReconciler) DeleteApplicationOnboarding(ctx context.Context, appOnboard *v1beta1.ApplicationOnboarding, fed *v1beta1.Federation) error {
	remoteName := "apponboard-" + uuid.V5(appOnboard.Spec.AppInfo.AppId+appOnboard.Spec.FederationContextId)
	return DeleteRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		&v1beta1.ApplicationOnboarding{},
		remoteName,
		appOnboard.Name,
		appOnboard.Namespace,
		"[AppOnboard][K8s]",
	)
}
