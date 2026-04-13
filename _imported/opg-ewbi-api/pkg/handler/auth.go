package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/neonephos-katalis/opg-ewbi-api/pkg/uuid"
)

func (h *handler) ValidateAuthHeaders(c echo.Context) (statusCode int, err error) {
	return http.StatusAccepted, nil
}

func (h *handler) generateFederationContextID(c echo.Context) string {
	userClientCredentials, _ := h.getRequestClientCredentialsFunc(c)
	return uuid.V5(userClientCredentials.ClientID)
}
