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

// ImageReconciler reconciles an Image object
type ImageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

// CreateImage
func (r *ImageReconciler) CreateImage(ctx context.Context, image *v1beta1.Image, fed *v1beta1.Federation) error {
	var imageBody v1beta1.ImageBody
	if image.Spec.ImageBody != nil {
		imageBody = *image.Spec.ImageBody
	}
	imageHost := &v1beta1.Image{
		TypeMeta: image.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "image-" + uuid.V5(image.Spec.ImageId+image.Spec.FederationContextId),
			Namespace: fed.Spec.FederationData.K8sOptions.Namespace,
		},
		Spec: v1beta1.ImageSpec{
			RelationType:        string(v1beta1.FederationRelationHost),
			FederationContextId: fed.Status.FederationContextId,
			ImageId:             image.Spec.ImageId,
			ImageBody:           &imageBody,
		},
	}
	if err := ApplyRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		imageHost,
		&v1beta1.Image{},
		image.Name,
		image.Namespace,
		v1beta1.GroupVersion.Group,
		v1beta1.GroupVersion.Version,
		v1beta1.PluralImage,
		"image-controller",
		"[Image][K8s]",
	); err != nil {
		return err
	}
	return nil
}

// UpdateImageStatus
func (r *ImageReconciler) UpdateImageStatus(ctx context.Context, image *v1beta1.Image, fed *v1beta1.Federation) error {
	imageHost := &v1beta1.Image{}
	remoteName := "image-" + uuid.V5(image.Spec.ImageId+image.Spec.FederationContextId)
	if err := GetRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		imageHost,
		remoteName,
		image.Name,
		image.Namespace,
		"[Image][K8s]",
	); err != nil {
		return err
	}
	image.Status = imageHost.Status
	return nil
}

// DeleteImage
func (r *ImageReconciler) DeleteImage(ctx context.Context, image *v1beta1.Image, fed *v1beta1.Federation) error {
	remoteName := "image-" + uuid.V5(image.Spec.ImageId+image.Spec.FederationContextId)
	return DeleteRemoteResource(
		ctx,
		r.Client,
		r.Scheme,
		fed,
		&v1beta1.Image{},
		remoteName,
		image.Name,
		image.Namespace,
		"[Image][K8s]",
	)
}
