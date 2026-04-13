package metastore

import (
	"context"

	"github.com/neonephos-katalis/opg-ewbi-api/api/federation/models"
	"github.com/neonephos-katalis/opg-ewbi-api/pkg/uuid"
	opgv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/v1beta1"
)

var _ Client = &k8sClient{}

var uuidV5Fn = uuid.V5

type Client interface {
	GetFederation(ctx context.Context, federationContextID string) (*Federation, error)
	CreateFederation(ctx context.Context, fed *Federation) (*Federation, error)
	UpdateFederationStatus(ctx context.Context, federationCallbackID string, status models.Status) error
	RemoveFederation(ctx context.Context, federationContextID string) error

	GetFile(ctx context.Context, federationContextID, id string) (*File, error)
	UploadFile(ctx context.Context, file *UploadFile) (*opgv1beta1.File, error)
	UpdateFileStatus(ctx context.Context, federationCallbackID string, updates *models.FileStatusCallbackLinkJSONRequestBody) error

	RemoveFile(ctx context.Context, federationContextID, id string) error

	GetArtefact(ctx context.Context, federationContextID, id string) (*Artefact, error)
	UploadArtefact(ctx context.Context, artefact *UploadArtefact) (*opgv1beta1.Artefact, error)
	UpdateArtefactStatus(ctx context.Context, federationCallbackID string, updates *models.ArtefactStatusCallbackLinkJSONRequestBody) error
	RemoveArtefact(ctx context.Context, federationContextID, id string) error

	GetApplication(ctx context.Context, federationContextID, id string) (*Application, error)
	OnboardApplication(ctx context.Context, app *OnboardApplication) (*opgv1beta1.Application, error)
	UpdateApplicationStatus(ctx context.Context, federationCallbackID string, updates *models.AppStatusCallbackLinkJSONRequestBody) error
	RemoveApplication(ctx context.Context, federationContextID, id string) error

	AddApplicationInstance(ctx context.Context, dep *ApplicationInstance) (*opgv1beta1.ApplicationInstance, error)
	GetApplicationInstance(ctx context.Context, federationContextID, id string) (*ApplicationInstance, error)
	UpdateApplicationInstanceStatus(ctx context.Context, federationCallbackID string, updates *models.AppInstCallbackLinkJSONRequestBody) error
	RemoveApplicationInstance(ctx context.Context, federationContextID, id string) error

	GetApplicationInstanceDetails(ctx context.Context, federationContextID, id string) (*ApplicationInstanceDetails, error)

	AddAvailabilityZones(ctx context.Context, federationContextId string, azs []string) error
	AddAvailabilityZone(ctx context.Context, az *PartnerAvailabilityZone) error
	GetAvailabilityZone(ctx context.Context, federationContextID, id string) (*PartnerAvailabilityZone, error)
	ListAvailabilityZones(ctx context.Context) ([]*PartnerAvailabilityZone, error)
	RemoveAvailabilityZone(ctx context.Context, federationContextID, id string) error

	GetClientCredentials(ctx context.Context, ClientID string) (ClientCredentials, error)
}
