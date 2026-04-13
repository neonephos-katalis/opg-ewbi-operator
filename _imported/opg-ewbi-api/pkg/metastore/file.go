package metastore

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/neonephos-katalis/opg-ewbi-api/api/federation/models"
	camara "github.com/neonephos-katalis/opg-ewbi-api/api/federation/server"
	opgv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/v1beta1"
)

type File struct {
	*camara.ViewFile200JSONResponse
	FederationContextId models.FederationContextId
}

func fileFromK8sCustomResource(fileID string, file opgv1beta1.File) (*File, error) {
	return &File{
		ViewFile200JSONResponse: &camara.ViewFile200JSONResponse{
			AppProviderId: file.Spec.AppProviderId,
			FileId:        fileID,
			FileName:      file.Spec.FileName,
			FileRepoLocation: &models.ObjectRepoLocation{
				Password: &file.Spec.Repo.Password,
				RepoURL:  &file.Spec.Repo.URL,
				Token:    &file.Spec.Repo.Token,
				UserName: &file.Spec.Repo.UserName,
			},
			FileType:        models.VirtImageType(file.Spec.FileType),
			FileVersionInfo: file.Spec.FileVersion,
			ImgInsSetArch:   models.CPUArchType(file.Spec.Image.InstructionSetArchitecture),
			ImgOSType: models.OSType{
				Architecture: models.OSTypeArchitecture(file.Spec.Image.OS.Architecture),
				Distribution: models.OSTypeDistribution(file.Spec.Image.OS.Distribution),
				License:      models.OSTypeLicense(file.Spec.Image.OS.License),
				Version:      models.OSTypeVersion(file.Spec.Image.OS.Version),
			},
			RepoType: (*models.UploadFileMultipartBodyRepoType)(&file.Spec.Repo.Type),
		},
		FederationContextId: file.Labels[opgLabel(federationContextIDLabel)],
	}, nil
}

type UploadFile struct {
	*models.UploadFileMultipartBody
	FederationContextId models.FederationContextId
}

func (f *UploadFile) MarshalJSON() ([]byte, error) {
	cp := *f.UploadFileMultipartBody
	cp.File = nil
	return json.Marshal(&cp)
}

func (m *UploadFile) k8sCustomResource(namespace string, opts ...Opt) (*opgv1beta1.File, error) {
	obj := &opgv1beta1.File{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sCustomResourceNameFromFileID(m.FederationContextId, m.FileId),
			Namespace: namespace,
			Labels: map[string]string{
				opgLabel(federationContextIDLabel): m.FederationContextId,
				opgLabel(idLabel):                  m.FileId,
				opgLabel(federationRelation):       host,
			},
		},
		Spec: opgv1beta1.FileSpec{
			AppProviderId: m.AppProviderId,
			FileName:      m.FileName,
			FileVersion:   m.FileVersionInfo,
			FileType:      string(m.FileType),
			Repo: opgv1beta1.Repo{
				Type:     defaultIfNil((*string)(m.RepoType)),
				URL:      defaultIfNil(m.FileRepoLocation.RepoURL),
				Password: defaultIfNil(m.FileRepoLocation.Password),
				Token:    defaultIfNil(m.FileRepoLocation.Token),
				UserName: defaultIfNil(m.FileRepoLocation.UserName),
			},
			Image: opgv1beta1.Image{
				InstructionSetArchitecture: string(m.ImgInsSetArch),
				OS: opgv1beta1.OS{
					Architecture: string(m.ImgOSType.Architecture),
					Distribution: string(m.ImgOSType.Distribution),
					License:      string(m.ImgOSType.License),
					Version:      string(m.ImgOSType.Version),
				},
			},
		},
	}
	for _, opt := range opts {
		if err := opt(&obj.ObjectMeta); err != nil {
			return nil, err
		}
	}

	return obj, nil
}

func k8sCustomResourceNameFromFileID(federationContextID, fileID string) string {
	return fmt.Sprintf("%s-%s", fileKind, uuidV5Fn(federationContextID+"/"+fileID))
}

func isValidFileStatus(status string) bool {
	switch opgv1beta1.FileState(status) {
	case opgv1beta1.FileStatePending, opgv1beta1.FileStateReady, opgv1beta1.FileStateError, opgv1beta1.FileStateUnknown:
		return true
	}
	return false
}
