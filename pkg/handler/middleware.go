package handler

import (
	"github.com/labstack/echo/v4"
)

// AuthMiddleware ensures that every request has valid authentication headers.
func AuthMiddleware(h *handler) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// TODO: Integration with IdP
			if errCode, err := h.ValidateAuthHeaders(c); err != nil {
				return sendErrorResponse(c, errCode, err.Error())
			}
			return next(c)
		}
	}
}
