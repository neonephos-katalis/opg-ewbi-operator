package metastore

import (
	"context"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	v1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/uuid"
)

var _ Client = &k8sClient{}

var uuidV5Fn = uuid.V5

type Client interface {
	GetFederation(ctx context.Context, federationContextID string) (*Federation, error)
	GetK8SFederation(ctx context.Context, federationContextID string) (*v1beta1.Federation, error)
	CreateFederation(ctx context.Context, fed *Federation) (*v1beta1.Federation, error)
	UpdateFederationStatus(ctx context.Context, federationCallbackID string, updates *models.PartnerStatusLinkJSONRequestBody) error
	RemoveFederation(ctx context.Context, federationContextID string) error
	PartnerDetailsCallback(ctx context.Context, federationContextId models.FederationContextId, request *models.PartnerDetailsCallbackJSONRequestBody) (*v1beta1.Federation, error)

	GetImage(ctx context.Context, federationContextID, id string) (*Image, error)
	UploadImage(ctx context.Context, file *UploadImage) (*v1beta1.Image, error)
	UpdateImageStatus(ctx context.Context, federationCallbackID string, updates *models.FileStatusCallbackLinkJSONRequestBody) error

	RemoveImage(ctx context.Context, federationContextID, id string) error

	GetArtefact(ctx context.Context, federationContextID, id string) (*Artefact, error)
	UploadArtefact(ctx context.Context, artefact *UploadArtefact) (*v1beta1.Artefact, error)
	UpdateArtefactStatus(ctx context.Context, federationCallbackID string, updates *models.ArtefactStatusCallbackLinkJSONRequestBody) error
	RemoveArtefact(ctx context.Context, federationContextID, id string) error

	GetApplication(ctx context.Context, federationContextID, id string) (*Application, error)
	OnboardApplication(ctx context.Context, app *OnboardApplication) (*v1beta1.ApplicationOnboarding, error)
	UpdateApplicationStatus(ctx context.Context, federationCallbackID string, updates *models.AppStatusCallbackLinkJSONRequestBody) error
	RemoveApplication(ctx context.Context, federationContextID, id string) error

	AddApplicationDeployment(ctx context.Context, dep *ApplicationInstance) (*v1beta1.ApplicationDeployment, error)
	GetApplicationDeployment(ctx context.Context, federationContextID, id string) (*ApplicationInstance, error)
	UpdateApplicationDeploymentStatus(ctx context.Context, federationCallbackID string, updates *models.AppInstCallbackLinkJSONRequestBody) error
	RemoveApplicationDeployment(ctx context.Context, federationContextID, appInstanceId, appId string) error

	GetApplicationDeploymentDetails(ctx context.Context, federationContextID, id string) (*ApplicationInstanceDetails, error)

	AddAvailabilityZones(ctx context.Context, federationContextId string, azs []string) error //Update ZoneDetails status in Federation CR
	AddAvailabilityZone(ctx context.Context, az *PartnerAvailabilityZone) error               // Create AvailabilityZone CR in K8s
	GetAvailabilityZone(ctx context.Context, federationContextID, id string) (*PartnerAvailabilityZone, error)
	ListAvailabilityZones(ctx context.Context) ([]*PartnerAvailabilityZone, error)
	RemoveAvailabilityZone(ctx context.Context, federationContextID, id string) error

	GetClientCredentials(ctx context.Context, ClientID string) (ClientCredentials, error)
}
