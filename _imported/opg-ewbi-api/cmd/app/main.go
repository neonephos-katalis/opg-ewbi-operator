package main

import (
	"encoding/json"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/neonephos-katalis/opg-ewbi-api/api/federation/server"
	"github.com/neonephos-katalis/opg-ewbi-api/cmd/app/config"
	"github.com/neonephos-katalis/opg-ewbi-api/pkg/handler"
	opgv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/v1beta1"
)

func main() {
	conf := config.GetConf()

	switch conf.Camara.LogLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	e := echo.New()
	// Captures request and response payloads and log them
	if conf.Camara.LogLevel == "debug" {
		e.Use(middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
			var reqBodyMap, resBodyMap map[string]any
			json.Unmarshal([]byte(reqBody), &reqBodyMap)
			json.Unmarshal([]byte(resBody), &resBodyMap)
			log.WithContext(c.Request().Context()).WithFields(
				logrus.Fields{
					"reqBody:": reqBodyMap,
					"resBody:": resBodyMap,
				}).Debug("request")
		}))
	}
	// Log all requests
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:    true,
		LogURI:       true,
		LogStatus:    true,
		LogError:     true,
		LogUserAgent: true,
		LogLatency:   true,
		LogValuesFunc: func(c echo.Context, values middleware.RequestLoggerValues) error {
			log.WithError(values.Error).WithFields(
				logrus.Fields{
					"method:":             values.Method,
					"uri":                 values.URI,
					"status":              values.Status,
					"userAgent":           values.UserAgent,
					"latencyMicroseconds": values.Latency.Microseconds(),
				}).Debug("request")
			return nil
		},
	}))
	// Validate request and return errors using the expected models.ProblemDetails format
	e.Use(server.Validator())

	scheme := runtime.NewScheme()
	utilruntime.Must(opgv1beta1.AddToScheme(scheme))

	config := ctrl.GetConfigOrDie()
	k8sClient, err := client.New(config, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		log.WithError(err).
			Fatal("failed to create k8sclient")
	}

	h := handler.NewServer(conf.Camara.ApiRoot, k8sClient, conf.Controller.Namespace)
	server.RegisterHandlers(e, h)
	e.Use(handler.AuthMiddleware(h))

	if err := e.Start(conf.Camara.HostAgentAddr); err != nil {
		log.WithError(err).
			Fatal("failed to run server")
	}
}
