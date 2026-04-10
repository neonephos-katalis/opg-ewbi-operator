package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/neonephos-katalis/opg-ewbi-api/api/federation/models"
)

// Notification payload.
// (POST /{federationCallbackId}/fileStatusCallbackLink)
func (h *handler) FileStatusCallbackLink(c echo.Context, federationCallbackId models.FederationCallbackId) error {
	ctx := h.getRequestContextFunc(c)

	request, err := bindRequest[models.FileStatusCallbackLinkJSONRequestBody](c)
	if err != nil {
		return sendErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	if err := h.metaStoreClient.UpdateFileStatus(ctx, federationCallbackId, request); err != nil {
		return sendErrorResponseFromError(c, err)
	}
	return c.JSON(http.StatusNoContent, nil)
}

// Notification payload.
// (POST /{federationCallbackId}/artefactStatusCallbackLink)
func (h *handler) ArtefactStatusCallbackLink(c echo.Context, federationCallbackId models.FederationCallbackId) error {
	ctx := h.getRequestContextFunc(c)

	request, err := bindRequest[models.ArtefactStatusCallbackLinkJSONRequestBody](c)
	if err != nil {
		return sendErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	if err := h.metaStoreClient.UpdateArtefactStatus(ctx, federationCallbackId, request); err != nil {
		return sendErrorResponseFromError(c, err)
	}
	return c.JSON(http.StatusNoContent, nil)
}

// Notification payload.
// (POST /{federationCallbackId}/appStatusCallbackLink)
func (h *handler) AppStatusCallbackLink(c echo.Context, federationCallbackId models.FederationCallbackId) error {
	ctx := h.getRequestContextFunc(c)

	request, err := bindRequest[models.AppStatusCallbackLinkJSONRequestBody](c)
	if err != nil {
		return sendErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	if err := h.metaStoreClient.UpdateApplicationStatus(ctx, federationCallbackId, request); err != nil {
		return sendErrorResponseFromError(c, err)
	}
	return c.JSON(http.StatusNoContent, nil)
}

// Notification payload.
// (POST /{federationCallbackId}/appInstCallbackLink)
func (h *handler) AppInstCallbackLink(c echo.Context, federationCallbackId models.FederationCallbackId) error {
	ctx := h.getRequestContextFunc(c)

	request, err := bindRequest[models.AppInstCallbackLinkJSONRequestBody](c)
	if err != nil {
		return sendErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	if err := h.metaStoreClient.UpdateApplicationInstanceStatus(ctx, federationCallbackId, request); err != nil {
		return sendErrorResponseFromError(c, err)
	}
	return c.JSON(http.StatusNoContent, nil)
}

// Notification about resource availability.
// (POST /{federationCallbackId}/availZoneNotifLink)
func (h *handler) AvailZoneNotifLink(c echo.Context, federationCallbackId models.FederationCallbackId) error {
	return c.JSON(http.StatusNotImplemented, nil)
}

// OP uses this callback api to notify partner OP about change in federation status, federation metadata or offered zone details. Allowed combinations of objectType and operationType are
// - FEDERATION - STATUS: Status specified by parameter 'federationStatus'.
// - ZONES - STATUS: Status specified by parameter 'zoneStatus'.
// - ZONES - ADD: Use parameter 'addZones' to define add new zones
// - ZONES - REMOVE: Use parameter 'removeZones' to define remove zones.
// - EDGE_DISCOVERY_SERVICE - UPDATE: Use parameter 'edgeDiscoverySvcEndPoint' to specify new endpoints
// - LCM_SERVICE - UPDATE: Use parameter 'lcmSvcEndPoint' to specify new endpoints
// - MOBILE_NETWORK_CODES - ADD: Use parameter 'addMobileNetworkIds' to define new mobile network codes.
// - MOBILE_NETWORK_CODES - REMOVE: Use parameter 'removeMobileNetworkIds' to remove mobile network codes.
// - FIXED_NETWORK_CODES - ADD: Use parameter 'addFixedNetworkIds' to define new fixed network codes.
// - FIXED_NETWORK_CODES - REMOVE: Use parameter 'removeFixedNetworkIds' to remove fixed network codes.
// (POST /{federationCallbackId}/partnerStatusLink)
func (h *handler) PartnerStatusLink(c echo.Context, federationCallbackId models.FederationCallbackId) error {
	ctx := h.getRequestContextFunc(c)

	request, err := bindRequest[models.PartnerStatusLinkJSONRequestBody](c)
	if err != nil {
		return sendErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	unsuportedOperationErr := func() error {
		return sendErrorResponse(c, http.StatusBadRequest, "unsuported operation type for objectType "+string(request.ObjectType))
	}
	switch request.ObjectType {
	case models.PartnerStatusLinkJSONBodyObjectTypeFEDERATION:
		if request.OperationType != models.PartnerStatusLinkJSONBodyOperationTypeSTATUS {
			return unsuportedOperationErr()
		}
		if request.FederationStatus == nil {
			return sendErrorResponse(c, http.StatusBadRequest, "missing federationStatus")
		}
		if err := h.metaStoreClient.UpdateFederationStatus(ctx, federationCallbackId, *request.FederationStatus); err != nil {
			return sendErrorResponseFromError(c, err)
		}
	default:
		return sendErrorResponse(c, http.StatusNotImplemented, "ObjectType not implemented")
	}

	return c.JSON(http.StatusNoContent, nil)
}

// Notification payload.
// (POST /{federationCallbackId}/resourceReservationCallbackLink)
func (h *handler) ResourceReservationCallbackLink(c echo.Context, federationCallbackId models.FederationCallbackId) error {
	return c.JSON(http.StatusNotImplemented, nil)
}
