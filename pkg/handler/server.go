package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/server"
	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/deployment"
	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/metastore"
)

var _ server.ServerInterface = &handler{}

const (
	headerKeyClientID = "X-Client-ID"
)

func NewServer(apiRoot string, k8sClient client.Client, namespace string) *handler {
	return &handler{
		apiRoot:                         apiRoot,
		depClient:                       deployment.NewClient(k8sClient, namespace),
		getRequestClientCredentialsFunc: getRequestClientCredentials,
		getRequestContextFunc:           getRequestContext,
		metaStoreClient:                 metastore.NewK8sClient(k8sClient, namespace),
	}
}

type handler struct {
	apiRoot                         string
	depClient                       deployment.Client
	getRequestClientCredentialsFunc func(echo.Context) (metastore.ClientCredentials, error) // test purposes
	getRequestContextFunc           func(echo.Context) context.Context                      // test purposes
	metaStoreClient                 metastore.Client
}

func (h *handler) CreateFederation(c echo.Context) error {
	ctx := h.getRequestContextFunc(c)

	request, err := bindRequest[models.FederationRequestData](c)
	if err != nil {
		return sendErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	federationID := h.generateFederationContextID(c)
	userClientCredentials, _ := h.getRequestClientCredentialsFunc(c)
	var fed *metastore.Federation
	if fed, err = h.metaStoreClient.CreateFederation(ctx, &metastore.Federation{
		ClientCredentials:     userClientCredentials,
		FederationRequestData: request,
		FederationContextId:   federationID,
	}); err != nil {
		return sendErrorResponseFromError(c, err)
	}

	c.Response().Header().Set("Location", h.apiRoot+"/operatorplatform/federation/v1/partner/"+federationID)
	response := models.FederationResponseData{
		FederationContextId:      &federationID,
		OfferedAvailabilityZones: fed.OfferedAvailabilityZones,
		PlatformCaps:             []models.FederationResponseDataPlatformCaps{},
	}

	return c.JSON(http.StatusOK, response)
}

// Instantiates an application on a partner OP zone.
// (POST /{federationContextId}/application/lcm)
func (h *handler) InstallApp(c echo.Context, federationContextId models.FederationContextId) error {
	request := models.InstallAppJSONBody{}
	if err := c.Bind(&request); err != nil {
		detail := err.Error()
		return c.JSON(http.StatusBadRequest, &models.ProblemDetails{
			Detail: &detail,
		})
	}
	if _,_, err := h.depClient.Install(h.getRequestContextFunc(c), &deployment.InstallDeployment{
		InstallAppJSONBody:  &request,
		FederationContextID: federationContextId,
	}); err != nil {
		return sendErrorResponseFromError(c, err)
	}

	return c.JSON(http.StatusAccepted, nil)
}

// Terminate an application instance on a partner OP zone.
// (DELETE /{federationContextId}/application/lcm/app/{appId}/instance/{appInstanceId}/zone/{zoneId})
func (h *handler) RemoveApp(c echo.Context, federationContextId models.FederationContextId, appId models.AppIdentifier, appInstanceId models.InstanceIdentifier, zoneId models.ZoneIdentifier) error {
	if err := h.depClient.Uninstall(h.getRequestContextFunc(c), federationContextId, appInstanceId); err != nil {
		return sendErrorResponseFromError(c, err)
	}
	return c.JSON(http.StatusOK, nil)
}

// Retrieves an application instance details from partner OP.
// (GET /{federationContextId}/application/lcm/app/{appId}/instance/{appInstanceId}/zone/{zoneId})
func (h *handler) GetAppInstanceDetails(c echo.Context, federationContextId models.FederationContextId, appId models.AppIdentifier, appInstanceId models.InstanceIdentifier, zoneId models.ZoneIdentifier) error {
	appInst, err := h.metaStoreClient.GetApplicationInstanceDetails(h.getRequestContextFunc(c), federationContextId, appInstanceId)
	if err != nil {
		return sendErrorResponseFromError(c, err)
	}

	return c.JSON(http.StatusOK, appInst.GetAppInstanceDetails200JSONResponse)
}

// Submits an application details to a partner OP. Based on the details provided,  partner OP shall do bookkeeping, resource validation and other pre-deployment operations.
// (POST /{federationContextId}/application/onboarding)
func (h *handler) OnboardApplication(c echo.Context, federationContextId models.FederationContextId) error {
	ctx := h.getRequestContextFunc(c)

	request := models.OnboardApplicationJSONBody{}
	if err := c.Bind(&request); err != nil {
		detail := err.Error()
		return c.JSON(http.StatusBadRequest, &models.ProblemDetails{
			Detail: &detail,
		})
	}

	if _,err := h.metaStoreClient.OnboardApplication(ctx, &metastore.OnboardApplication{
		OnboardApplicationJSONBody: &request,
		FederationContextId:        federationContextId,
	}); err != nil {
		return sendErrorResponseFromError(c, err)
	}

	return c.JSON(http.StatusAccepted, nil)
}
// Deboards the application from any zones, if any, and deletes the App.
// (GET /{federationContextId}/application/onboarding/app/{appId})
func (h *handler) DeleteApp(c echo.Context, federationContextId models.FederationContextId, appId string) error {
	if err := h.metaStoreClient.RemoveApplication(h.getRequestContextFunc(c), federationContextId, appId); err != nil {
		return sendErrorResponseFromError(c, err)
	}
	return c.JSON(http.StatusOK, nil)
}

// Retrieves application details from partner OP
// (GET /{federationContextId}/application/onboarding/app/{appId})
func (h *handler) ViewApplication(c echo.Context, federationContextId models.FederationContextId, appId models.AppIdentifier) error {
	app, err := h.metaStoreClient.GetApplication(h.getRequestContextFunc(c), federationContextId, appId)
	if err != nil {
		return sendErrorResponseFromError(c, err)
	}

	return c.JSON(http.StatusOK, app.ViewApplication200JSONResponse)
}

// Deboards an application from partner OP zones
// (DELETE /{federationContextId}/application/onboarding/app/{appId}/zone/{zoneId})
func (h *handler) DeboardApplication(c echo.Context, federationContextId models.FederationContextId, appId models.AppIdentifier, zoneId models.ZoneIdentifier) error {
	if err := h.metaStoreClient.RemoveApplication(h.getRequestContextFunc(c), federationContextId, appId); err != nil {
		return sendErrorResponseFromError(c, err)
	}

	return c.JSON(http.StatusAccepted, nil)
}

// Uploads application artefact on partner OP. Artefact is a zip file containing  scripts and/or packaging files like Terraform or Helm which are required to create an instance of an application.
// (POST /{federationContextId}/artefact)
func (h *handler) UploadArtefact(c echo.Context, federationContextId models.FederationContextId) error {
	ctx := h.getRequestContextFunc(c)
	request, err := models.NewUploadArtefactMultipartBody(c)
	if err != nil {
		detail := err.Error()
		return c.JSON(http.StatusBadRequest, &models.ProblemDetails{
			Detail: &detail,
		})
	}

	if _,err := h.metaStoreClient.UploadArtefact(ctx, &metastore.UploadArtefact{
		UploadArtefactMultipartBody: request,
		FederationContextId:         federationContextId,
	}); err != nil {
		return sendErrorResponseFromError(c, err)
	}

	return c.JSON(http.StatusOK, nil)
}

// Removes an artefact from partner OP.
// (DELETE /{federationContextId}/artefact/{artefactId})
func (h *handler) RemoveArtefact(c echo.Context, federationContextId models.FederationContextId, artefactId models.ArtefactId) error {
	if err := h.metaStoreClient.RemoveArtefact(h.getRequestContextFunc(c), federationContextId, artefactId); err != nil {
		return sendErrorResponseFromError(c, err)
	}
	return c.JSON(http.StatusOK, nil)
}

// Retrieves details about an artefact.
// (GET /{federationContextId}/artefact/{artefactId})
func (h *handler) GetArtefact(c echo.Context, federationContextId models.FederationContextId, artefactId models.ArtefactId) error {
	artefact, err := h.metaStoreClient.GetArtefact(h.getRequestContextFunc(c), federationContextId, artefactId)
	if err != nil {
		return sendErrorResponseFromError(c, err)
	}

	return c.JSON(http.StatusOK, artefact.GetArtefact200JSONResponse)
}

// Uploads an image file. Originating OP uses this api to onboard an application image to partner OP.
// (POST /{federationContextId}/files)
func (h *handler) UploadFile(c echo.Context, federationContextId models.FederationContextId) error {
	ctx := h.getRequestContextFunc(c)
	request, err := models.NewUploadFileMultipartBody(c)
	if err != nil {
		detail := err.Error()
		return c.JSON(http.StatusBadRequest, &models.ProblemDetails{
			Detail: &detail,
		})
	}

	if _,err := h.metaStoreClient.UploadFile(ctx, &metastore.UploadFile{
		UploadFileMultipartBody: request,
		FederationContextId:     federationContextId,
	}); err != nil {
		return sendErrorResponseFromError(c, err)
	}

	return c.JSON(http.StatusOK, nil)
}

// Removes an image file from partner OP.
// (DELETE /{federationContextId}/files/{fileId})
func (h *handler) RemoveFile(c echo.Context, federationContextId models.FederationContextId, fileId models.FileId) error {
	if err := h.metaStoreClient.RemoveFile(h.getRequestContextFunc(c), federationContextId, fileId); err != nil {
		return sendErrorResponseFromError(c, err)
	}
	return c.JSON(http.StatusOK, nil)
}

// View an image file from partner OP.
// (GET /{federationContextId}/files/{fileId})
func (h *handler) ViewFile(c echo.Context, federationContextId models.FederationContextId, fileId models.FileId) error {
	file, err := h.metaStoreClient.GetFile(h.getRequestContextFunc(c), federationContextId, fileId)
	if err != nil {
		return sendErrorResponseFromError(c, err)
	}

	return c.JSON(http.StatusOK, file.ViewFile200JSONResponse)
}

// Remove existing federation with the partner OP
// (DELETE /{federationContextId}/partner)
func (h *handler) DeleteFederationDetails(c echo.Context, federationContextId models.FederationContextId) error {
	ctx := h.getRequestContextFunc(c)

	if err := h.metaStoreClient.RemoveFederation(ctx, federationContextId); err != nil {
		return sendErrorResponseFromError(c, err)
	}
	return c.JSON(http.StatusOK, nil)
}

// Retrieves details about the federation context with the partner OP. The response shall provide info about the zones offered by the partner, partner OP network codes, information about edge discovery and LCM service etc.
// (GET /{federationContextId}/partner)
func (h *handler) GetFederationDetails(c echo.Context, federationContextId models.FederationContextId) error {
	ctx := h.getRequestContextFunc(c)

	fed, err := h.metaStoreClient.GetFederation(ctx, federationContextId)
	if err != nil {
		return sendErrorResponseFromError(c, err)
	}

	response := server.GetFederationDetails200JSONResponse{
		AllowedFixedNetworkIds: fed.OrigOPFixedNetworkCodes,
		AllowedMobileNetworkIds: &models.MobileNetworkIds{
			Mcc:  fed.OrigOPMobileNetworkCodes.Mcc,
			Mncs: fed.OrigOPMobileNetworkCodes.Mncs,
		},
		OfferedAvailabilityZones: fed.OfferedAvailabilityZones,
	}

	return c.JSON(http.StatusOK, response)
}

// Originating OP informs partner OP that it is willing to access the specified zones  and partner OP shall reserve compute and network resources for these zones.
// (POST /{federationContextId}/zones)
func (h *handler) ZoneSubscribe(c echo.Context, federationContextId models.FederationContextId) error {
	ctx := h.getRequestContextFunc(c)

	// The request binding does nothing more than validate the format at the moment.
	// We are not using the request body anywhere.
	zoneRegistrationRequest := models.ZoneRegistrationRequestData{}
	if err := c.Bind(&zoneRegistrationRequest); err != nil {
		detail := err.Error()
		return c.JSON(http.StatusBadRequest, &models.ProblemDetails{
			Detail: &detail,
		})
	}
	fed, err := h.metaStoreClient.GetFederation(ctx, federationContextId)
	if err != nil {
		return sendErrorResponseFromError(c, err)
	}

	existingAvailabilityZones := make(map[string]struct{}, len(*fed.OfferedAvailabilityZones))
	for _, az := range *fed.OfferedAvailabilityZones {
		existingAvailabilityZones[az.ZoneId] = struct{}{}
	}
	for _, az := range zoneRegistrationRequest.AcceptedAvailabilityZones {
		if _, ok := existingAvailabilityZones[az]; !ok {
			detail := fmt.Sprintf("accepted availability zone '%s': not found", az)
			return c.JSON(http.StatusInternalServerError, &models.ProblemDetails{
				Detail: &detail,
			})
		}
	}

	// updateFederationWithAcceptedSites
	if err := h.metaStoreClient.AddAvailabilityZones(ctx, federationContextId, zoneRegistrationRequest.AcceptedAvailabilityZones); err != nil {
		return sendErrorResponseFromError(c, err)
	}

	registered := []models.ZoneRegisteredData{}
	for _, acc := range zoneRegistrationRequest.AcceptedAvailabilityZones {
		registered = append(registered, models.ZoneRegisteredData{
			ZoneId: acc,
		})
	}
	resp := models.ZoneRegistrationResponseData{
		AcceptedZoneResourceInfo: registered,
	}
	return c.JSON(http.StatusOK, &resp)
}

// Retrieves details about the computation and network resources that partner OP has reserved for this zone.
// (GET /{federationContextId}/zones/{zoneId})
func (h *handler) GetZoneData(c echo.Context, federationContextId models.FederationContextId, zoneId models.ZoneIdentifier) error {
	ctx := h.getRequestContextFunc(c)

	az, err := h.metaStoreClient.GetAvailabilityZone(ctx, federationContextId, zoneId)
	if err != nil {
		return sendErrorResponseFromError(c, err)
	}
	data := &models.ZoneRegisteredData{
		ZoneId: az.ZoneDetails.ZoneId,
	}
	return c.JSON(http.StatusOK, data)
}

func getRequestContext(c echo.Context) context.Context {
	return c.Request().Context()
}

func getRequestClientCredentials(c echo.Context) (metastore.ClientCredentials, error) {
	headerErrorResponse := func(header string) (metastore.ClientCredentials, error) {
		return metastore.ClientCredentials{}, fmt.Errorf("missing %s header", header)
	}

	clientID := c.Request().Header.Get(headerKeyClientID)
	if clientID == "" {
		return headerErrorResponse(headerKeyClientID)
	}
	return metastore.ClientCredentials{ClientID: clientID}, nil
}
