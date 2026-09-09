package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
)

// Validates the authenticity of a roaming user from home OP
// (GET /{federationContextId}/roaminguserauth/device/{deviceId}/token/{authToken})
// func (s *handler) AuthenticateDevice(c echo.Context, federationContextId models.FederationContextId, deviceId models.DeviceId, authToken models.AuthorizationToken) error {
// 	return c.JSON(http.StatusNotImplemented, nil)
// }

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

func (s *handler) APIForwarding(ctx echo.Context, federationContextId models.FederationContextId, serviceAPINameVal models.ServiceAPINameVal) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}

func (s *handler) ApplyApplicationPolicy(ctx echo.Context, federationContextId models.FederationContextId, applPolicySubsId models.EventSubscriptionIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}

func (s *handler) ApplyOperationPolicy(ctx echo.Context, federationContextId models.FederationContextId, opsPolicySubsId models.EventSubscriptionIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}

func (s *handler) CreateAlarmReportingSubscription(ctx echo.Context, federationContextId models.FederationContextId) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}

func (s *handler) CreateApplicationEventSubscription(ctx echo.Context, federationContextId models.FederationContextId) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}

func (s *handler) CreateApplicationPolicySubscription(ctx echo.Context, federationContextId models.FederationContextId) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}

func (s *handler) CreateEventSubscription(ctx echo.Context, federationContextId models.FederationContextId) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) DeleteAlarmSubscription(ctx echo.Context, federationContextId models.FederationContextId, alarmSubsId models.SubscriptionIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}

func (s *handler) GetAlarmsList(ctx echo.Context, federationContextId models.FederationContextId, alarmSubsId models.SubscriptionIdentifier, params models.GetAlarmsListParams) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}

func (s *handler) DeleteEventSubscription(ctx echo.Context, federationContextId models.FederationContextId, eventSubsId models.EventSubscriptionIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}

func (s *handler) GetEventsList(ctx echo.Context, federationContextId models.FederationContextId, eventSubsId models.EventSubscriptionIdentifier, params models.GetEventsListParams) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}

func (s *handler) CreateEventCriterion(ctx echo.Context, federationContextId models.FederationContextId, eventSubsId models.EventSubscriptionIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}

func (s *handler) DeleteEventCriterion(ctx echo.Context, federationContextId models.FederationContextId, eventSubsId models.EventSubscriptionIdentifier, eventId models.EventIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}

func (s *handler) SubscribeMonitoringInfo(ctx echo.Context, federationContextId models.FederationContextId, params models.SubscribeMonitoringInfoParams) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) CreateNetworkCapsEventSubscription(ctx echo.Context, federationContextId models.FederationContextId) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) DeleteNwEventNotifSubscription(ctx echo.Context, federationContextId models.FederationContextId, nwEventSubsId models.EventSubscriptionIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) CreateNetworkCapEvent(ctx echo.Context, federationContextId models.FederationContextId, nwEventSubsId models.EventSubscriptionIdentifier, params models.CreateNetworkCapEventParams) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) DeleteNetworkCapSubscription(ctx echo.Context, federationContextId models.FederationContextId, nwEventSubsId models.EventSubscriptionIdentifier, params models.DeleteNetworkCapSubscriptionParams) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) GetNetworkCapsSubscribedList(ctx echo.Context, federationContextId models.FederationContextId, nwEventSubsId models.EventSubscriptionIdentifier, params models.GetNetworkCapsSubscribedListParams) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) CreateOperationPolicySubscription(ctx echo.Context, federationContextId models.FederationContextId) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}

func (s *handler) DeleteApplNotifSubscription(ctx echo.Context, federationContextId models.FederationContextId, appNotifSubsId models.EventSubscriptionIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}

func (s *handler) GetFederationAPIs(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}

func (s *handler) GetFederationContextId(ctx echo.Context) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) GetFederationHealth(ctx echo.Context, federationContextId models.FederationContextId) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) GetPlatformCapabilities(ctx echo.Context, federationContextId models.FederationContextId, params models.GetPlatformCapabilitiesParams) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) GetServiceAPISessionInfo(ctx echo.Context, federationContextId models.FederationContextId, connectID models.ConnectID, customerID models.CustomerID) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) GetServiceAPIsDetails(ctx echo.Context, federationContextId models.FederationContextId, serviceType models.ServiceType) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) GetZoneDetails(ctx echo.Context, federationContextId models.FederationContextId, zoneId models.ZoneIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) ModifyApplEventNotifSubscription(ctx echo.Context, federationContextId models.FederationContextId, appNotifSubsId models.EventSubscriptionIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) ModifyApplicationPolicy(ctx echo.Context, federationContextId models.FederationContextId, applPolicySubsId models.EventSubscriptionIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) ModifyOperationPolicy(ctx echo.Context, federationContextId models.FederationContextId, opsPolicySubsId models.EventSubscriptionIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) RegisterApplicationPolicy(ctx echo.Context, federationContextId models.FederationContextId, applPolicySubsId models.EventSubscriptionIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) RegisterOperationPolicy(ctx echo.Context, federationContextId models.FederationContextId, opsPolicySubsId models.EventSubscriptionIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) RemoveApplicationPolicies(ctx echo.Context, federationContextId models.FederationContextId, applPolicySubsId models.EventSubscriptionIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) RemoveAppsEventSubscription(ctx echo.Context, federationContextId models.FederationContextId, appNotifSubsId models.EventSubscriptionIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) RemoveOperationPolicies(ctx echo.Context, federationContextId models.FederationContextId, opsPolicySubsId models.EventSubscriptionIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) RemoveServiceAPISession(ctx echo.Context, federationContextId models.FederationContextId, connectID models.ConnectID, customerID models.CustomerID) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) RenewFederation(ctx echo.Context, federationContextId models.FederationContextId) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) RetrieveAppPolicyTemplates(ctx echo.Context, federationContextId models.FederationContextId, applPolicySubsId models.EventSubscriptionIdentifier, params models.RetrieveAppPolicyTemplatesParams) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) RetrieveApplSubsMetaInfo(ctx echo.Context, federationContextId models.FederationContextId, appNotifSubsId models.EventSubscriptionIdentifier, params models.RetrieveApplSubsMetaInfoParams) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) RetrieveApplicationPolicy(ctx echo.Context, federationContextId models.FederationContextId, applPolicySubsId models.EventSubscriptionIdentifier, params models.RetrieveApplicationPolicyParams) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) RetrieveAppsEventsInfo(ctx echo.Context, federationContextId models.FederationContextId, appNotifSubsId models.EventSubscriptionIdentifier) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) RetrieveOperationPolicy(ctx echo.Context, federationContextId models.FederationContextId, opsPolicySubsId models.EventSubscriptionIdentifier, params models.RetrieveOperationPolicyParams) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) RetrieveOpsPolicyTemplates(ctx echo.Context, federationContextId models.FederationContextId, opsPolicySubsId models.EventSubscriptionIdentifier, params models.RetrieveOpsPolicyTemplatesParams) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
func (s *handler) SubscribeApplsEvtNotif(ctx echo.Context, federationContextId models.FederationContextId, appNotifSubsId models.EventSubscriptionIdentifier, params models.SubscribeApplsEvtNotifParams) error {
	return ctx.JSON(http.StatusNotImplemented, nil)
}
