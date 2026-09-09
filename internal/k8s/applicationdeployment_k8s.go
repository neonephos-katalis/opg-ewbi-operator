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

// ApplicationDeploymentReconciler reconciles an Artefact object
type ApplicationDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

func (r *ApplicationDeploymentReconciler) CreateApplicationDeployment(ctx context.Context, appDeploy *v1beta1.ApplicationDeployment, fed *v1beta1.Federation) error {
	appDeployHost := &v1beta1.ApplicationDeployment{
		TypeMeta: appDeploy.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "appdeploy-" + uuid.V5(appDeploy.Spec.FederationContextId+appDeploy.Spec.AppId+appDeploy.Spec.AppInstanceId),
			Namespace: fed.Spec.FederationData.K8sOptions.Namespace,
		},
		Spec: v1beta1.ApplicationDeploymentSpec{
			RelationType:        string(v1beta1.FederationRelationHost),
			FederationContextId: fed.Status.FederationContextId,
			AppId:               appDeploy.Spec.AppId,
			AppInstanceId:       appDeploy.Spec.AppInstanceId,
			ZoneId:              appDeploy.Spec.ZoneId,
			AppProviderId:       appDeploy.Spec.AppProviderId,
			AppDetails:          appDeploy.Spec.AppDetails,
		},
	}
	err := ApplyRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		appDeployHost,
		&v1beta1.ApplicationDeployment{},
		appDeploy.Name,
		appDeploy.Namespace,
		v1beta1.GroupVersion.Group,
		v1beta1.GroupVersion.Version,
		v1beta1.PluralApplicationDeployment,
		"app-deploy-controller",
		"[AppDeploy][K8s]",
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *ApplicationDeploymentReconciler) UpdateApplicationDeploymentStatus(ctx context.Context, appDeploy *v1beta1.ApplicationDeployment, fed *v1beta1.Federation) error {
	appDeployHost := &v1beta1.ApplicationDeployment{}
	remoteName := "appdeploy-" + uuid.V5(appDeploy.Spec.FederationContextId+appDeploy.Spec.AppId+appDeploy.Spec.AppInstanceId)
	if err := GetRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		appDeployHost,
		remoteName,
		appDeploy.Name,
		appDeploy.Namespace,
		"[AppDeploy][K8s]",
	); err != nil {
		return err
	}
	appDeploy.Status = appDeployHost.Status
	return nil
}

func (r *ApplicationDeploymentReconciler) DeleteApplicationDeployment(ctx context.Context, appDeploy *v1beta1.ApplicationDeployment, fed *v1beta1.Federation) error {
	remoteName := "appdeploy-" + uuid.V5(appDeploy.Spec.FederationContextId+appDeploy.Spec.AppId+appDeploy.Spec.AppInstanceId)
	return DeleteRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		&v1beta1.ApplicationDeployment{},
		remoteName,
		appDeploy.Name,
		appDeploy.Namespace,
		"[AppDeploy][K8s]",
	)
}
