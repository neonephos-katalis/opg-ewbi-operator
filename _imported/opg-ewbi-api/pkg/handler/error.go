package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/neonephos-katalis/opg-ewbi-api/api/federation/models"
	"github.com/neonephos-katalis/opg-ewbi-api/pkg/metastore"
)

// sendErrorResponse sends a JSON response with a specified status code and error detail.
func sendErrorResponse(c echo.Context, statusCode int, detail string) error {
	return c.JSON(statusCode, &models.ProblemDetails{
		Detail: &detail,
	})
}

// sendErrorResponseFromError sends a JSON response based on an error.
// It determines the appropriate HTTP status code from the error.
func sendErrorResponseFromError(c echo.Context, err error) error {
	detail := err.Error()
	statusCode := statusCodeFromError(err)
	return c.JSON(statusCode, models.ProblemDetails{
		Detail: &detail,
	})
}

// statusCodeFromError maps specific errors to HTTP status codes.
// It uses errors.Is to properly detect wrapped errors.
func statusCodeFromError(err error) int {
	switch {
	case errors.Is(err, metastore.ErrAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, metastore.ErrBadRequest):
		return http.StatusBadRequest
	case errors.Is(err, metastore.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, metastore.ErrUnauthorized):
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
