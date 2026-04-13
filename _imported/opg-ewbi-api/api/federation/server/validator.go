package server

import (
	"os"

	"github.com/deepmap/oapi-codegen/pkg/middleware"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"

	"github.com/neonephos-katalis/opg-ewbi-api/api/federation/models"
)

func Validator() echo.MiddlewareFunc {
	// Use validation middleware to check all requests against the OpenAPI schema.
	swagger, err := models.GetSwagger()
	if err != nil {
		log.WithError(err).
			Fatal("failed loading swagger spec for server")
		os.Exit(1)
	}
	// Clear out the servers array in the swagger spec, that skips validating
	// that server names match. We don't know how this thing will be run.
	swagger.Servers = nil

	return middleware.OapiRequestValidatorWithOptions(swagger, models.ValidatorOption(swagger))
}
