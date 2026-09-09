package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
)

// Notification payload.
// (POST /{federationCallbackId}/fileStatusCallbackLink)
func (h *handler) FileStatusCallbackLink(c echo.Context, federationCallbackId models.FederationCallbackId) error {
	ctx := h.getRequestContextFunc(c)

	request, err := bindRequest[models.FileStatusCallbackLinkJSONRequestBody](c)
	if err != nil {
		return sendErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	if err := h.metaStoreClient.UpdateImageStatus(ctx, federationCallbackId, request); err != nil {
		return sendErrorResponseFromError(c, err)
	}
	return c.JSON(http.StatusNoContent, nil)
}

// (POST /{federationCallbackId}/partnerDetailsCallbackLink')
func (h *handler) PartnerDetailsCallback(c echo.Context, federationCallbackId models.FederationCallbackId) error {
	ctx := h.getRequestContextFunc(c)

	request, err := bindRequest[models.PartnerDetailsCallbackJSONRequestBody](c)
	if err != nil {
		return sendErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	if _, err := h.metaStoreClient.PartnerDetailsCallback(ctx, federationCallbackId, request); err != nil {
		return sendErrorResponseFromError(c, err)
	}
	return h.GetFederationDetails(c, federationCallbackId)
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

	if err := h.metaStoreClient.UpdateApplicationDeploymentStatus(ctx, federationCallbackId, request); err != nil {
		return sendErrorResponseFromError(c, err)
	}
	return c.JSON(http.StatusNoContent, nil)
}

// Notification about resource availability.
// (POST /{federationCallbackId}/availZoneNotifLink)
func (h *handler) AvailZoneNotifLink(c echo.Context, federationCallbackId models.FederationCallbackId) error {
	return c.JSON(http.StatusNotImplemented, nil)
}

// (POST /{federationCallbackId}/partnerStatusLink)
func (h *handler) PartnerStatusLink(c echo.Context, federationCallbackId models.FederationCallbackId) error {
	ctx := h.getRequestContextFunc(c)

	request, err := bindRequest[models.PartnerStatusLinkJSONRequestBody](c)
	if err != nil {
		return sendErrorResponse(c, http.StatusBadRequest, err.Error())
	}

	if err := h.metaStoreClient.UpdateFederationStatus(ctx, federationCallbackId, request); err != nil {
		return sendErrorResponseFromError(c, err)
	}
	return c.JSON(http.StatusNoContent, nil)
}

// Notification payload.
// (POST /{federationCallbackId}/resourceReservationCallbackLink)
func (h *handler) ResourceReservationCallbackLink(c echo.Context, federationCallbackId models.FederationCallbackId) error {
	return c.JSON(http.StatusNotImplemented, nil)
}
