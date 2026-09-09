package metastore

import (
	"context"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scli "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	camara "github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/server"
	v1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	uu "github.com/neonephos-katalis/opg-ewbi-operator/pkg/uuid"
	"github.com/pkg/errors"
)

type Image struct {
	*camara.ViewFile200JSONResponse
	FederationContextId models.FederationContextId
}
type UploadImage struct {
	*models.UploadFileMultipartBody
	FederationContextId models.FederationContextId
}

func (f *UploadImage) MarshalJSON() ([]byte, error) {
	cp := *f.UploadFileMultipartBody
	cp.File = nil
	return json.Marshal(&cp)
}

func isValidImageStatus(status string) bool {
	switch v1beta1.ImageState(status) {
	case v1beta1.ImageStatePending, v1beta1.ImageStateReady, v1beta1.ImageStateError, v1beta1.ImageStateUnknown:
		return true
	}
	return false
}

func (c *k8sClient) searchImage(ctx context.Context, federationContextId string, imageId string, role string) (*v1beta1.Image, error) {
	var imageList v1beta1.ImageList
	if err := c.kubernetes.List(ctx, &imageList, &k8scli.ListOptions{Namespace: c.getNamespace()}); err != nil {
		return nil, err
	}
	if len(imageList.Items) == 0 {
		return nil, errors.Errorf("Image not found for federationContextId: %s and imageId: %s and role: %s and namespace: %s", federationContextId, imageId, role, c.getNamespace())
	}
	for i := range imageList.Items {
		image := imageList.Items[i]
		if image.Spec.ImageId == imageId && image.Spec.FederationContextId == federationContextId && image.Spec.RelationType == role {
			return &image, nil
		}
	}
	return nil, errors.Errorf("Image not found for federationContextId: %s and imageId: %s and role: %s and namespace: %s", federationContextId, imageId, role, c.getNamespace())
}

func (c *k8sClient) UploadImage(ctx context.Context, image *UploadImage) (*v1beta1.Image, error) {
	if _, err := c.searchFederation(ctx, image.FederationContextId, "HOST"); err != nil {
		return nil, err
	}
	imageId := "image-" + uu.V5(image.FederationContextId+image.FileId)
	var repoType string
	if image.RepoType != nil {
		repoType = string(*image.RepoType)
	}
	var repoLocation *v1beta1.ImageRepoLocation
	if image.FileRepoLocation != nil {
		repoLocation = &v1beta1.ImageRepoLocation{
			RepoURL:  defaultIfNil(image.FileRepoLocation.RepoURL),
			Password: defaultIfNil(image.FileRepoLocation.Password),
			Token:    defaultIfNil(image.FileRepoLocation.Token),
			UserName: defaultIfNil(image.FileRepoLocation.UserName),
		}
	}
	var callbackLink string
	if image.FileNotifLink != nil {
		callbackLink = string(*image.FileNotifLink)
	}
	obj := &v1beta1.Image{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imageId,
			Namespace: c.getNamespace(),
		},
		Spec: v1beta1.ImageSpec{
			RelationType:        string(host),
			ImageId:             image.FileId,
			FederationContextId: image.FederationContextId,
			ImageNotifLink:      callbackLink,
			ImageBody: &v1beta1.ImageBody{
				AppProviderId:     image.AppProviderId,
				ImageName:         image.FileName,
				ImageVersionInfo:  image.FileVersionInfo,
				ImageType:         string(image.FileType),
				RepoType:          repoType,
				ImageRepoLocation: repoLocation,
				ImgInsSetArch:     string(image.ImgInsSetArch),
				ImgOSType: &v1beta1.ImgOSType{
					Architecture: string(image.ImgOSType.Architecture),
					Distribution: string(image.ImgOSType.Distribution),
					License:      string(image.ImgOSType.License),
					Version:      string(image.ImgOSType.Version),
				},
			},
		},
	}

	if err := c.createK8sObject(obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func (c *k8sClient) GetImage(ctx context.Context, federationContextID, id string) (*Image, error) {
	if _, err := c.searchFederation(ctx, federationContextID, "HOST"); err != nil {
		return nil, err
	}
	image, err := c.searchImage(ctx, federationContextID, id, "HOST")
	if err != nil {
		return nil, err
	}
	return &Image{
		ViewFile200JSONResponse: &camara.ViewFile200JSONResponse{
			AppProviderId: image.Spec.ImageBody.AppProviderId,
			FileId:        models.FileId(id),
			FileName:      image.Spec.ImageBody.ImageName,
			FileRepoLocation: &models.ObjectRepoLocation{
				Password: &image.Spec.ImageBody.ImageRepoLocation.Password,
				RepoURL:  &image.Spec.ImageBody.ImageRepoLocation.RepoURL,
				Token:    &image.Spec.ImageBody.ImageRepoLocation.Token,
				UserName: &image.Spec.ImageBody.ImageRepoLocation.UserName,
			},
			FileType:        models.VirtImageType(image.Spec.ImageBody.ImageType),
			FileVersionInfo: image.Spec.ImageBody.ImageVersionInfo,
			ImgInsSetArch:   models.CPUArchType(image.Spec.ImageBody.ImgInsSetArch),
			ImgOSType: models.OSType{
				Architecture: models.OSTypeArchitecture(image.Spec.ImageBody.ImgOSType.Architecture),
				Distribution: models.OSTypeDistribution(image.Spec.ImageBody.ImgOSType.Distribution),
				License:      models.OSTypeLicense(image.Spec.ImageBody.ImgOSType.License),
				Version:      models.OSTypeVersion(image.Spec.ImageBody.ImgOSType.Version),
			},
			RepoType: (*models.RepoType)(&image.Spec.ImageBody.RepoType),
		},
		FederationContextId: image.Labels[opgLabel(federationContextIDLabel)],
	}, nil
}

func (c *k8sClient) RemoveImage(ctx context.Context, federationContextID, id string) error {
	if _, err := c.searchFederation(ctx, federationContextID, "HOST"); err != nil {
		return err
	}
	image, err := c.searchImage(ctx, federationContextID, id, "HOST")
	if err != nil {
		return err
	}
	if err := c.kubernetes.Delete(context.TODO(), image, &k8scli.DeleteOptions{}); err != nil {
		return errors.Wrapf(err, "unable to remove image")
	}
	return nil
}

func (c *k8sClient) UpdateImageStatus(ctx context.Context, federationCallbackID string, updates *models.FileStatusCallbackLinkJSONRequestBody) error {
	image, err := c.searchImage(ctx, federationCallbackID, updates.FileId, "GUEST")
	if err != nil {
		return err
	}
	originalImage := image.DeepCopy()
	image.Status.State = v1beta1.ImageState(updates.UpdateStatus)
	if isValidImageStatus(string(image.Status.State)) {
		return c.patchK8sStatus(originalImage, image)
	}
	return nil
}
