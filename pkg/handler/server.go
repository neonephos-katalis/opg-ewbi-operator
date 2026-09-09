package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/server"
	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/deployment"
	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/metastore"
	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/uuid"
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
		Client:                          k8sClient,
	}
}

type handler struct {
	apiRoot                         string
	depClient                       deployment.Client
	getRequestClientCredentialsFunc func(echo.Context) (metastore.ClientCredentials, error) // test purposes
	getRequestContextFunc           func(echo.Context) context.Context                      // test purposes
	metaStoreClient                 metastore.Client
	client.Client
}

func mapServiceEndpoint(src *v1beta1.ServiceEndpoint) *models.ServiceEndpoint {
	if src == nil {
		return nil
	}

	// Helper inline per convertire le slice e restituire il puntatore
	mapIpv4 := func(ips []string) *[]models.Ipv4Addr {
		if len(ips) == 0 {
			return nil
		}
		res := make([]models.Ipv4Addr, len(ips))
		for i, ip := range ips {
			res[i] = models.Ipv4Addr(ip)
		}
		return &res
	}

	mapIpv6 := func(ips []string) *[]models.Ipv6Addr {
		if len(ips) == 0 {
			return nil
		}
		res := make([]models.Ipv6Addr, len(ips))
		for i, ip := range ips {
			res[i] = models.Ipv6Addr(ip)
		}
		return &res
	}

	return &models.ServiceEndpoint{
		Fqdn:          &src.Fqdn,
		Ipv4Addresses: mapIpv4(src.Ipv4Addresses),
		Ipv6Addresses: mapIpv6(src.Ipv6Addresses),
		Port:          src.Port,
	}
}

// POST /Partner
func (h *handler) CreateFederation(c echo.Context) error {
	ctx := h.getRequestContextFunc(c)

	request, err := bindRequest[models.FederationRequestData](c)
	if err != nil {
		return sendErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	userClientCredentials, _ := h.getRequestClientCredentialsFunc(c)
	var fed *v1beta1.Federation
	federationContextId := uuid.V5(*request.OrigOPFederationId + request.InitialDate.String() + *request.OrigOPCountryCode)
	if fed, err = h.metaStoreClient.CreateFederation(ctx, &metastore.Federation{
		ClientCredentials:     userClientCredentials,
		FederationRequestData: request,
		FederationContextId:   federationContextId,
	}); err != nil {
		return sendErrorResponseFromError(c, err)
	}

	offeredZones := make([]models.ZoneDetails, len(fed.Status.ZoneDetails))
	for i, zd := range fed.Status.ZoneDetails {
		offeredZones[i] = models.ZoneDetails{
			ZoneId:           zd.ZoneId,
			Geolocation:      &zd.Geolocation,
			GeographyDetails: zd.GeographyDetails,
		}
	}
	var partnerMobileNetCodes *models.MobileNetworkIds
	if fed.Status.MobileNetworkIds != nil {
		partnerMobileNetCodes = &models.MobileNetworkIds{
			Mcc:  &fed.Status.MobileNetworkIds.Mcc,
			Mncs: &fed.Status.MobileNetworkIds.Mncs,
		}
	}
	c.Response().Header().Set("Location", h.apiRoot+"/operatorplatform/federation/v1/partner/"+federationContextId)
	response := models.FederationResponseData{
		FederationContextId:      &federationContextId,
		OfferedAvailabilityZones: &offeredZones,
		PlatformCaps:             fed.Status.PlatformCaps,
		PartnerOPFederationId:    &fed.Status.PartnerOPFederationId,
		PartnerOPCountryCode:     &fed.Status.PartnerOPCountryCode,

		//OPTIONAL
		EdgeDiscoveryServiceEndPoint: mapServiceEndpoint(fed.Status.EdgeDiscoveryServiceEndPoint),
		LcmServiceEndPoint:           mapServiceEndpoint(fed.Status.LcmServiceEndPoint),
		PartnerOPMobileNetworkCodes:  partnerMobileNetCodes,
		PartnerOPFixedNetworkCodes:   &fed.Status.FixedNetworkIds,
		FederationExpiryDate:         &fed.Status.FederationExpiryDate.Time,
		FederationRenewalDate:        &fed.Status.FederationRenewalDate.Time,
	}

	return c.JSON(http.StatusOK, response)
}

// Instantiates an application on a partner OP zone.
// (POST /{federationContextId}/application/lcm)
func (h *handler) InstallApp(c echo.Context, federationContextId models.FederationContextId, params models.InstallAppParams) error {
	request := models.InstallAppJSONBody{}
	if err := c.Bind(&request); err != nil {
		detail := err.Error()
		return c.JSON(http.StatusBadRequest, &models.ProblemDetails{
			Detail: &detail,
		})
	}
	if _, _, err := h.depClient.Install(h.getRequestContextFunc(c), &deployment.InstallDeployment{
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
	if err := h.depClient.Uninstall(h.getRequestContextFunc(c), federationContextId, appId, appInstanceId); err != nil {
		return sendErrorResponseFromError(c, err)
	}
	return c.JSON(http.StatusOK, nil)
}

// Retrieves an application instance details from partner OP.
// (GET /{federationContextId}/application/lcm/app/{appId}/instance/{appInstanceId}/zone/{zoneId})
func (h *handler) GetAppInstanceDetails(c echo.Context, federationContextId models.FederationContextId, appId models.AppIdentifier, appInstanceId models.InstanceIdentifier, zoneId models.ZoneIdentifier) error {
	appInst, err := h.metaStoreClient.GetApplicationDeploymentDetails(h.getRequestContextFunc(c), federationContextId, appInstanceId)
	if err != nil {
		return sendErrorResponseFromError(c, err)
	}

	return c.JSON(http.StatusOK, appInst.GetAppInstanceDetails200JSONResponse)
}

// Registers a callback to be called when there are updates on details about the federation context with the partner OP. The callback body shall provide info about the zones offered by the partner, partner OP network codes, information about edge discovery and LCM service etc.
// (POST /{federationContextId}/partner)
func (h *handler) PartnerDetails(c echo.Context, federationContextId models.FederationContextId) error {
	ctx := h.getRequestContextFunc(c)

	k8sFed, err := h.metaStoreClient.GetK8SFederation(ctx, federationContextId)
	if err != nil {
		return sendErrorResponseFromError(c, err)
	}

	var offeredZones []models.ZoneDetails
	if len(k8sFed.Status.ZoneDetails) > 0 {
		offeredZones := make([]models.ZoneDetails, len(k8sFed.Status.ZoneDetails))
		for i, zd := range k8sFed.Status.ZoneDetails {
			offeredZones[i] = models.ZoneDetails{
				ZoneId:           zd.ZoneId,
				Geolocation:      &zd.Geolocation,
				GeographyDetails: zd.GeographyDetails,
			}
		}
	}
	var partnerMobileNetCodes *models.MobileNetworkIds
	if k8sFed.Status.MobileNetworkIds != nil {
		partnerMobileNetCodes = &models.MobileNetworkIds{
			Mcc:  &k8sFed.Status.MobileNetworkIds.Mcc,
			Mncs: &k8sFed.Status.MobileNetworkIds.Mncs,
		}
	}
	var edgeDiscoveryServiceEndPoint *models.ServiceEndpoint
	if k8sFed.Status.EdgeDiscoveryServiceEndPoint != nil {
		edgeDiscoveryServiceEndPoint = &models.ServiceEndpoint{
			Fqdn:          &k8sFed.Status.EdgeDiscoveryServiceEndPoint.Fqdn,
			Ipv4Addresses: &k8sFed.Status.EdgeDiscoveryServiceEndPoint.Ipv4Addresses,
			// Ipv6Addresses: &k8sFed.Status.EdgeDiscoveryServiceEndPoint.Ipv6Addresses,
			Port: k8sFed.Status.EdgeDiscoveryServiceEndPoint.Port,
		}
	}
	var lcmServiceEndPoint *models.ServiceEndpoint
	if k8sFed.Status.LcmServiceEndPoint != nil {
		lcmServiceEndPoint = &models.ServiceEndpoint{
			Fqdn:          &k8sFed.Status.LcmServiceEndPoint.Fqdn,
			Ipv4Addresses: &k8sFed.Status.LcmServiceEndPoint.Ipv4Addresses,
			// Ipv6Addresses: &k8sFed.Status.LcmServiceEndPoint.Ipv6Addresses,
			Port: k8sFed.Status.LcmServiceEndPoint.Port,
		}
	}

	response := server.PartnerDetails200JSONResponse{
		EdgeDiscoveryServiceEndPoint: edgeDiscoveryServiceEndPoint,
		LcmServiceEndPoint:           lcmServiceEndPoint,
		FederationExpiryDate:         k8sFed.Status.FederationExpiryDate.Time,
		FederationRenewalDate:        k8sFed.Status.FederationRenewalDate.Time,
		PartnerOPCountryCode:         &k8sFed.Status.PartnerOPCountryCode,
		PartnerOPFederationId:        &k8sFed.Status.PartnerOPFederationId,
		PartnerOPFixedNetworkCodes:   &k8sFed.Status.FixedNetworkIds,
		PartnerOPMobileNetworkCodes:  partnerMobileNetCodes,
		OfferedAvailabilityZones:     &offeredZones,
		PlatformCaps:                 k8sFed.Status.PlatformCaps,
	}

	return c.JSON(http.StatusOK, response)
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

	if _, err := h.metaStoreClient.OnboardApplication(ctx, &metastore.OnboardApplication{
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

	if _, err := h.metaStoreClient.UploadArtefact(ctx, &metastore.UploadArtefact{
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
	if _, err := h.metaStoreClient.UploadImage(ctx, &metastore.UploadImage{
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
	if err := h.metaStoreClient.RemoveImage(h.getRequestContextFunc(c), federationContextId, fileId); err != nil {
		return sendErrorResponseFromError(c, err)
	}
	return c.JSON(http.StatusOK, nil)
}

// View an image file from partner OP.
// (GET /{federationContextId}/files/{fileId})
func (h *handler) ViewFile(c echo.Context, federationContextId models.FederationContextId, fileId models.FileId) error {
	file, err := h.metaStoreClient.GetImage(h.getRequestContextFunc(c), federationContextId, fileId)
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

	registered := []models.ZoneRegisteredData{}
	for _, acc := range zoneRegistrationRequest.AcceptedAvailabilityZones {
		registered = append(registered, models.ZoneRegisteredData{
			ZoneId: acc,
		})
	}

	// updateFederationWithAcceptedSites
	if err := h.metaStoreClient.AddAvailabilityZones(ctx, federationContextId, zoneRegistrationRequest.AcceptedAvailabilityZones); err != nil {
		return sendErrorResponseFromError(c, err)
	}

	resp := models.ZoneRegistrationResponseData{
		AcceptedZoneResourceInfo: registered,
	}
	return c.JSON(http.StatusOK, &resp)
}

// Retrieves details about the computation and network resources that partner OP has reserved for this zone.
// (GET /{federationContextId}/zones/{zoneId})
func (h *handler) GetZoneData(c echo.Context, federationContextId models.FederationContextId, params models.GetZoneDataParams) error {
	ctx := h.getRequestContextFunc(c)

	az, err := h.metaStoreClient.GetAvailabilityZone(ctx, federationContextId, string(*params.ZoneId))
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
