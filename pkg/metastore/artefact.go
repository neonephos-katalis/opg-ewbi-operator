package metastore

import (
	"context"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scli "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	opgmodels "github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	camara "github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/server"
	v1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	uu "github.com/neonephos-katalis/opg-ewbi-operator/pkg/uuid"
	"github.com/pkg/errors"
)

type Artefact struct {
	*camara.GetArtefact200JSONResponse
	FederationContextId models.FederationContextId
}

type UploadArtefact struct {
	*models.UploadArtefactMultipartBody
	FederationContextId models.FederationContextId
}

func (a *UploadArtefact) MarshalJSON() ([]byte, error) {
	cp := *a.UploadArtefactMultipartBody
	cp.ArtefactFile = nil
	return json.Marshal(&cp)
}

func isValidArtefactStatus(status string) bool {
	switch v1beta1.ArtefactState(status) {
	case v1beta1.ArtefactStatePending, v1beta1.ArtefactStateReady, v1beta1.ArtefactStateError, v1beta1.ArtefactStateUnknown:
		return true
	}
	return false
}

func (m *UploadArtefact) componentSpec() []v1beta1.ComponentSpec {
	out := make([]v1beta1.ComponentSpec, len(m.ComponentSpec))
	for i, componentSpec := range m.ComponentSpec {
		exposeInterfacesInput := defaultIfNil(componentSpec.ExposedInterfaces)
		exposedInterfaces := make([]v1beta1.ExposedInterfaceInfo, len(exposeInterfacesInput))
		for j, exposeInterface := range exposeInterfacesInput {
			exposedInterfaces[j] = v1beta1.ExposedInterfaceInfo{
				Port:           int32(exposeInterface.CommPort),
				InterfaceId:    exposeInterface.InterfaceId,
				Protocol:       string(exposeInterface.CommProtocol),
				VisibilityType: string(exposeInterface.VisibilityType),
			}
		}
		images := []string{}
		for _, image := range componentSpec.Images {
			images = append(images, image)
		}
		out[i] = v1beta1.ComponentSpec{
			ComponentName: componentSpec.ComponentName,
			Images:        images,
			CommandLineParams: &v1beta1.CommandLineParams{
				Command:     componentSpec.CommandLineParams.Command,
				CommandArgs: defaultIfNil(componentSpec.CommandLineParams.CommandArgs),
			},
			NumOfInstances:    int32(componentSpec.NumOfInstances),
			RestartPolicy:     string(componentSpec.RestartPolicy),
			ExposedInterfaces: exposedInterfaces,
			ComputeResourceProfile: &v1beta1.ComputeResourceProfile{
				CPUArchType:    string(componentSpec.ComputeResourceProfile.CpuArchType),
				CPUExclusivity: defaultIfNil(componentSpec.ComputeResourceProfile.CpuExclusivity),
				Memory:         componentSpec.ComputeResourceProfile.Memory,
				NumCPU:         componentSpec.ComputeResourceProfile.NumCPU,
			},
		}
	}
	return out
}

func (c *k8sClient) searchArtefact(ctx context.Context, federationContextId string, artefactId string, role string) (*v1beta1.Artefact, error) {
	var artefactList v1beta1.ArtefactList
	if err := c.kubernetes.List(ctx, &artefactList, &k8scli.ListOptions{Namespace: c.getNamespace()}); err != nil {
		return nil, err
	}
	if len(artefactList.Items) == 0 {
		return nil, errors.Errorf("Artefact not found for federationContextId: %s and artefactId: %s and role: %s", federationContextId, artefactId, role)
	}
	for i := range artefactList.Items {
		artefact := artefactList.Items[i]
		if artefact.Spec.ArtefactId == artefactId && artefact.Spec.FederationContextId == federationContextId && artefact.Spec.RelationType == role {
			return &artefact, nil
		}
	}
	return nil, errors.Errorf("Artefact not found for federationContextId: %s and artefactId: %s and role: %s", federationContextId, artefactId, role)
}

func (c *k8sClient) UploadArtefact(ctx context.Context, artefact *UploadArtefact) (*v1beta1.Artefact, error) {
	if _, err := c.searchFederation(ctx, artefact.FederationContextId, "HOST"); err != nil {
		return nil, err
	}
	var callbackLink string
	if artefact.ArtefactNotifLink != nil {
		callbackLink = string(*artefact.ArtefactNotifLink)
	}
	artefactId := "artefact-" + uu.V5(artefact.FederationContextId+artefact.AppProviderId+artefact.ArtefactId)
	artefactHost := &v1beta1.Artefact{
		ObjectMeta: metav1.ObjectMeta{
			Name:      artefactId,
			Namespace: c.getNamespace(),
		},
		Spec: v1beta1.ArtefactSpec{
			RelationType:        string(v1beta1.FederationRelationHost),
			FederationContextId: artefact.FederationContextId,
			ArtefactId:          artefact.ArtefactId,
			ArtefactNotifLink:   callbackLink,
			ArtefactBody: &v1beta1.ArtefactBody{
				AppProviderId:          artefact.AppProviderId,
				ArtefactName:           artefact.ArtefactName,
				ArtefactVersionInfo:    artefact.ArtefactVersionInfo,
				ArtefactDescription:    string(artefact.ArtefactDescriptorType),
				ArtefactVirtType:       string(artefact.ArtefactVirtType),
				ComponentSpec:          artefact.componentSpec(),
				ArtefactDescriptorType: string(artefact.ArtefactDescriptorType),
			},
		},
	}

	if err := c.createK8sObject(artefactHost); err != nil {
		return nil, err
	}
	return artefactHost, nil
}

func (c *k8sClient) GetArtefact(ctx context.Context, federationContextID, id string) (*Artefact, error) {
	artefact, err := c.searchArtefact(ctx, federationContextID, id, "GUEST")
	if err != nil {
		return nil, err
	}
	componentSpec := make([]models.ComponentSpec, len(artefact.Spec.ArtefactBody.ComponentSpec))
	for i, cs := range artefact.Spec.ArtefactBody.ComponentSpec {
		imagesIds := []opgmodels.FileId{}
		for _, i := range cs.Images {
			imagesIds = append(imagesIds, opgmodels.FileId(i))
		}
		exposeInterfaces := make([]models.InterfaceDetails, len(cs.ExposedInterfaces))
		for j, ei := range cs.ExposedInterfaces {
			exposeInterfaces[j] = models.InterfaceDetails{
				CommPort:       int32(ei.Port),
				CommProtocol:   models.InterfaceDetailsCommProtocol(ei.Protocol),
				InterfaceId:    ei.InterfaceId,
				VisibilityType: models.InterfaceDetailsVisibilityType(ei.VisibilityType),
			}
		}
		componentSpec[i] = models.ComponentSpec{
			ComponentName: cs.ComponentName,
			CommandLineParams: &models.CommandLineParams{
				Command:     cs.CommandLineParams.Command,
				CommandArgs: &cs.CommandLineParams.CommandArgs,
			},
			Images:         imagesIds,
			NumOfInstances: int32(cs.NumOfInstances),
			RestartPolicy:  models.ComponentSpecRestartPolicy(cs.RestartPolicy),
			ComputeResourceProfile: models.ComputeResourceInfo{
				CpuArchType:    models.ComputeResourceInfoCpuArchType(cs.ComputeResourceProfile.CPUArchType),
				CpuExclusivity: &cs.ComputeResourceProfile.CPUExclusivity,
				Memory:         cs.ComputeResourceProfile.Memory,
				NumCPU:         cs.ComputeResourceProfile.NumCPU,
			},
			ExposedInterfaces: &exposeInterfaces,
		}
	}
	if err != nil {
		return nil, err
	}
	return &Artefact{
		GetArtefact200JSONResponse: &camara.GetArtefact200JSONResponse{
			AppProviderId:          artefact.Spec.ArtefactBody.AppProviderId,
			ArtefactId:             models.ArtefactId(id),
			ArtefactName:           artefact.Spec.ArtefactBody.ArtefactName,
			ArtefactDescriptorType: models.ArtefactDescriptorType(artefact.Spec.ArtefactBody.ArtefactDescriptorType),
			ArtefactVirtType:       models.ArtefactVirtType(artefact.Spec.ArtefactBody.ArtefactVirtType),
		},
		FederationContextId: artefact.Labels[opgLabel(federationContextIDLabel)],
	}, nil
}

func (c *k8sClient) RemoveArtefact(ctx context.Context, federationContextID, id string) error {
	if _, err := c.searchFederation(ctx, federationContextID, "HOST"); err != nil {
		return err
	}
	image, err := c.searchArtefact(ctx, federationContextID, id, "HOST")
	if err != nil {
		return err
	}
	if err := c.kubernetes.Delete(context.TODO(), image, &k8scli.DeleteOptions{}); err != nil {
		return errors.Wrapf(err, "unable to remove artefact")
	}
	return nil
}

func (c *k8sClient) UpdateArtefactStatus(ctx context.Context, federationCallbackID string, updates *models.ArtefactStatusCallbackLinkJSONRequestBody) error {
	art, err := c.searchArtefact(ctx, federationCallbackID, updates.ArtefactId, "GUEST")
	if err != nil {
		return err
	}
	originalArt := art.DeepCopy()
	art.Status.State = v1beta1.ArtefactState(updates.UpdateStatus)
	if isValidArtefactStatus(string(art.Status.State)) {
		return c.patchK8sStatus(originalArt, art)
	}
	return nil
}
