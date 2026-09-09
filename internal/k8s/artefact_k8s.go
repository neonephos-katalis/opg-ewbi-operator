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
	"reflect"

	v1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ArtefactReconciler reconciles an Artefact object
type ArtefactReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

func (r *ArtefactReconciler) CreateArtefact(ctx context.Context, art *v1beta1.Artefact, fed *v1beta1.Federation) error {
	var artefactBody v1beta1.ArtefactBody
	if !reflect.DeepEqual(art.Spec.ArtefactBody, v1beta1.ArtefactBody{}) {
		artefactBody = *art.Spec.ArtefactBody
	}
	artHost := &v1beta1.Artefact{
		TypeMeta: art.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "art-" + uuid.V5(art.Spec.ArtefactId+art.Spec.FederationContextId),
			Namespace: fed.Spec.FederationData.K8sOptions.Namespace,
		},
		Spec: v1beta1.ArtefactSpec{
			RelationType:        string(v1beta1.FederationRelationHost),
			FederationContextId: fed.Status.FederationContextId,
			ArtefactId:          art.Spec.ArtefactId,
			ArtefactBody:        &artefactBody,
		},
	}
	err := ApplyRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		artHost,
		&v1beta1.Artefact{},
		art.Name,
		art.Namespace,
		v1beta1.GroupVersion.Group,
		v1beta1.GroupVersion.Version,
		v1beta1.PluralArtefact,
		"artefact-controller",
		"[Artefact][K8s]",
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *ArtefactReconciler) UpdateArtefactStatus(ctx context.Context, art *v1beta1.Artefact, fed *v1beta1.Federation) error {
	artefactHost := &v1beta1.Artefact{}
	remoteName := "art-" + uuid.V5(art.Spec.ArtefactId+art.Spec.FederationContextId)
	if err := GetRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		artefactHost,
		remoteName,
		art.Name,
		art.Namespace,
		"[Artefact][K8s]",
	); err != nil {
		return err
	}
	art.Status = artefactHost.Status
	return nil
}

func (r *ArtefactReconciler) DeleteArtefact(ctx context.Context, art *v1beta1.Artefact, fed *v1beta1.Federation) error {
	remoteName := "art-" + uuid.V5(art.Spec.ArtefactId+art.Spec.FederationContextId)
	return DeleteRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		&v1beta1.Artefact{},
		remoteName,
		art.Name,
		art.Namespace,
		"[Artefact][K8s]",
	)
}
