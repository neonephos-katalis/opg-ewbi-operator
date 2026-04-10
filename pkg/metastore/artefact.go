package metastore

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	camara "github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/server"
	opgv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/v1beta1"
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

func (m *UploadArtefact) files() []string {
	out := []string{}
	for _, componentSpec := range m.ComponentSpec {
		out = append(out, componentSpec.Images...)
	}
	return out
}

func (m *UploadArtefact) componentSpec() []opgv1beta1.ComponentSpec {
	out := make([]opgv1beta1.ComponentSpec, len(m.ComponentSpec))
	for i, componentSpec := range m.ComponentSpec {
		exposeInterfacesInput := defaultIfNil(componentSpec.ExposedInterfaces)
		exposedInterfaces := make([]opgv1beta1.ExposedInterface, len(exposeInterfacesInput))
		for j, exposeInterface := range exposeInterfacesInput {
			exposedInterfaces[j] = opgv1beta1.ExposedInterface{
				Port:           int64(exposeInterface.CommPort),
				InterfaceId:    exposeInterface.InterfaceId,
				Protocol:       string(exposeInterface.CommProtocol),
				VisibilityType: string(exposeInterface.VisibilityType),
			}
		}

		out[i] = opgv1beta1.ComponentSpec{
			Name:   componentSpec.ComponentName,
			Images: componentSpec.Images,
			CommandLineParams: opgv1beta1.CommandLine{
				Command: componentSpec.CommandLineParams.Command,
				Args:    defaultIfNil(componentSpec.CommandLineParams.CommandArgs),
			},
			NumOfInstances:    int64(componentSpec.NumOfInstances),
			RestartPolicy:     string(componentSpec.RestartPolicy),
			ExposedInterfaces: exposedInterfaces,
			ComputeResourceProfile: opgv1beta1.ComputeResourceProfile{
				CPUArchType:    string(componentSpec.ComputeResourceProfile.CpuArchType),
				CPUExclusivity: defaultIfNil(componentSpec.ComputeResourceProfile.CpuExclusivity),
				Memory:         componentSpec.ComputeResourceProfile.Memory,
				NumCPU:         componentSpec.ComputeResourceProfile.NumCPU,
			},
		}
	}
	return out
}

func (m *UploadArtefact) k8sCustomResource(namespace string, opts ...Opt) (*opgv1beta1.Artefact, error) {
	obj := &opgv1beta1.Artefact{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sCustomResourceNameFromArtefactID(m.FederationContextId, m.ArtefactId),
			Namespace: namespace,
			Labels: map[string]string{
				opgLabel(federationContextIDLabel): m.FederationContextId,
				opgLabel(idLabel):                  m.ArtefactId,
				opgLabel(federationRelation):       host,
			},
		},
		Spec: opgv1beta1.ArtefactSpec{
			AppProviderId:   m.AppProviderId,
			ArtefactName:    m.ArtefactName,
			ArtefactVersion: m.ArtefactVersionInfo,
			DescriptorType:  string(m.ArtefactDescriptorType),
			VirtType:        string(m.ArtefactVirtType),
			ComponentSpec:   m.componentSpec(),
		},
	}
	for _, opt := range opts {
		if err := opt(&obj.ObjectMeta); err != nil {
			return nil, err
		}
	}

	return obj, nil
}

func k8sCustomResourceNameFromArtefactID(federationContextID, artefactID string) string {
	return fmt.Sprintf("%s-%s", artefactKind, uuidV5Fn(federationContextID+"/"+artefactID))
}

func artefactFromK8sCustomResource(artefact opgv1beta1.Artefact) (*Artefact, error) {
	componentSpec := make([]models.ComponentSpec, len(artefact.Spec.ComponentSpec))
	for i, cs := range artefact.Spec.ComponentSpec {
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
			ComponentName: cs.Name,
			CommandLineParams: &models.CommandLineParams{
				Command:     cs.CommandLineParams.Command,
				CommandArgs: &cs.CommandLineParams.Args,
			},
			Images:         cs.Images,
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
	return &Artefact{
		GetArtefact200JSONResponse: &camara.GetArtefact200JSONResponse{
			AppProviderId:          artefact.Spec.AppProviderId,
			ArtefactId:             artefact.Labels[opgLabel(idLabel)],
			ArtefactName:           artefact.Spec.ArtefactName,
			ArtefactDescriptorType: models.UploadArtefactMultipartBodyArtefactDescriptorType(artefact.Spec.DescriptorType),
			ArtefactVirtType:       models.UploadArtefactMultipartBodyArtefactVirtType(artefact.Spec.VirtType),
			ComponentSpec:          &componentSpec,
		},
		FederationContextId: artefact.Labels[opgLabel(federationContextIDLabel)],
	}, nil
}

func isValidArtefactStatus(status string) bool {
	switch opgv1beta1.ArtefactState(status) {
	case opgv1beta1.ArtefactStateReconciling, opgv1beta1.ArtefactStateReady, opgv1beta1.ArtefactStateError, opgv1beta1.ArtefactStateUnknown:
		return true
	}
	return false
}
