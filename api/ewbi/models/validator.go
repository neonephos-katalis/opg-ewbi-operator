package models

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/deepmap/oapi-codegen/pkg/middleware"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
)

func ValidatorOption(swagger *openapi3.T) *middleware.Options {
	return &middleware.Options{
		// Options: openapi3filter.Options{
		// }
		Skipper: func(c echo.Context) bool {
			return strings.HasPrefix(c.Request().Header.Get("Content-Type"), "multipart/form-data")
		},
		ErrorHandler: func(c echo.Context, err *echo.HTTPError) error {
			title := fmt.Sprintf("%d - %s", err.Code, http.StatusText(err.Code))
			problem := ProblemDetails{
				Title: &title,
				// Parsing the message for invalid params is not so easy, skipping this for now
				// InvalidParams *[]InvalidParam
			}
			substrings := strings.SplitN(err.Message.(string), ": ", 2)
			if len(substrings) >= 1 {
				problem.Cause = &substrings[0]
			}
			if len(substrings) >= 2 {
				problem.Detail = &substrings[1]
			}
			return c.JSON(http.StatusBadRequest, &problem)
		},
	}
}
