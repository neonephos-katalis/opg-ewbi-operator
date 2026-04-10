package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/neonephos-katalis/opg-ewbi-api/api/federation/models"
)

// Validates the authenticity of a roaming user from home OP
// (GET /{federationContextId}/roaminguserauth/device/{deviceId}/token/{authToken})
func (s *handler) AuthenticateDevice(c echo.Context, federationContextId models.FederationContextId, deviceId models.DeviceId, authToken models.AuthorizationToken) error {
	return c.JSON(http.StatusNotImplemented, nil)
}

// Reserves resources (compute, network and storage) on a partner OP zone.
// ISVs registered with home OP reserves resources on a partner OP zone.
// (POST /{federationContextId}/isv/resource/zone/{zoneId}/appProvider/{appProviderId})
func (s *handler) CreateResourcePools(c echo.Context, federationContextId models.FederationContextId, zoneId models.ZoneIdentifier, appProviderId models.AppProviderId) error {
	return c.JSON(http.StatusNotImplemented, nil)
}

// Retrieves all application instances of partner OP
// (GET /{federationContextId}/application/lcm/app/{appId}/appProvider/{appProviderId})
func (s *handler) GetAllAppInstances(c echo.Context, federationContextId models.FederationContextId, appId models.AppIdentifier, appProviderId models.AppProviderId) error {
	return c.JSON(http.StatusNotImplemented, nil)
}

// Edge discovery procedures towards partner OP over E/WBI.
// Originating OP requests partner OP to provide a list of candidate zones
// where an application instance can be created. Partner OP applies a set
// of filtering criteria to select candidate zones.
// (POST /{federationContextId}/edgenodesharing/edgeDiscovery)
func (s *handler) GetCandidateZones(c echo.Context, federationContextId models.FederationContextId) error {
	return c.JSON(http.StatusNotImplemented, nil)
}

// Retrieves the resource pool reserved by an ISV
// (GET /{federationContextId}/isv/resource/zone/{zoneId}/appProvider/{appProviderId})
func (s *handler) ViewISVResPool(c echo.Context, federationContextId models.FederationContextId, zoneId models.ZoneIdentifier, appProviderId models.AppProviderId) error {
	return c.JSON(http.StatusNotImplemented, nil)
}

// Forbid/allow application instantiation on a partner zone
// (POST /{federationContextId}/application/onboarding/app/{appId}/zoneForbid)
func (s *handler) LockUnlockApplicationZone(c echo.Context, federationContextId models.FederationContextId, appId models.AppIdentifier) error {
	return c.JSON(http.StatusNotImplemented, nil)
}

// Onboards an existing application to a new zone within partner OP.
// (POST /{federationContextId}/application/onboarding/app/{appId}/additionalZones)
func (s *handler) OnboardExistingAppNewZones(c echo.Context, federationContextId models.FederationContextId, appId models.AppIdentifier) error {
	return c.JSON(http.StatusNotImplemented, nil)
}

// Deletes the resource pool reserved by an ISV
// (DELETE /{federationContextId}/isv/resource/zone/{zoneId}/appProvider/{appProviderId}/pool/{poolId})
func (s *handler) RemoveISVResPool(c echo.Context, federationContextId models.FederationContextId, zoneId models.ZoneIdentifier, appProviderId models.AppProviderId, poolId models.PoolId) error {
	return c.JSON(http.StatusNotImplemented, nil)
}

// Asservate usage of a partner OP zone.
// Originating OP informs partner OP that it will no longer access the specified zone.
// (DELETE /{federationContextId}/zones/{zoneId})
func (s *handler) ZoneUnsubscribe(c echo.Context, federationContextId models.FederationContextId, zoneId models.ZoneIdentifier) error {
	return c.JSON(http.StatusNotImplemented, nil)
}

// Updates partner OP about changes in application compute resource requirements,
// QOS Profile, associated descriptor, or change in associated components
// (PATCH /{federationContextId}/application/onboarding/app/{appId})
func (s *handler) UpdateApplication(c echo.Context, federationContextId models.FederationContextId, appId models.AppIdentifier) error {
	return c.JSON(http.StatusNotImplemented, nil)
}

// API used by the Originating OP towards the partner OP, to update the parameters associated to the existing federation
// (PATCH /{federationContextId}/partner)
func (h *handler) UpdateFederation(c echo.Context, federationContextId models.FederationContextId) error {
	return c.JSON(http.StatusNotImplemented, nil)
}

// Updates resources reserved for a pool by an ISV
// (PATCH /{federationContextId}/isv/resource/zone/{zoneId}/appProvider/{appProviderId}/pool/{poolId})
func (s *handler) UpdateISVResPool(c echo.Context, federationContextId models.FederationContextId, zoneId models.ZoneIdentifier, appProviderId models.AppProviderId, poolId models.PoolId) error {
	return c.JSON(http.StatusNotImplemented, nil)
}
