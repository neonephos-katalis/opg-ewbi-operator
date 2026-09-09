/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rest

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	opgmodels "github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FederationReconciler reconciles a Federation object
type FederationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

func handleFederationProblemDetails(log logr.Logger, code int, p *opgmodels.ProblemDetails) {
	log.Info(">>> [Federation] Response with error", "error", code, "details", p)
}

func (r *FederationReconciler) CreateFederation(ctx context.Context, fed *v1beta1.Federation) error {
	log := ctrl.Log
	log.Info(">>> [Federation] Using OPG API to create federation", "name", fed.Name, "namespace", fed.Namespace)
	var partnerStatusLink string
	var tokenUrl string
	if fed.Spec.FederationData.RestOptions != nil {
		partnerStatusLink = fed.Spec.FederationData.RestOptions.PartnerStatusLink
		tokenUrl = fed.Spec.FederationData.RestOptions.TokenUrl
	}
	// Prepare the request payload for creating a federation
	origOPFedID := models.FederationIdentifier(fed.Spec.FederationData.OrigOPFederationId)
	fedReq := opgmodels.FederationRequestData{
		InitialDate:        fed.Spec.FederationData.InitialDate.Time,
		OrigOPCountryCode:  &fed.Spec.FederationData.OrigOPCountryCode,
		OrigOPFederationId: &origOPFedID,
		PartnerStatusLink:  partnerStatusLink,
	}
	if len(fed.Spec.FixedNetworkIds) > 0 {
		fedReq.OrigOPFixedNetworkCodes = &fed.Spec.FixedNetworkIds
	}
	if fed.Spec.MobileNetworkIds != nil {
		if fed.Spec.MobileNetworkIds.Mcc != "" || len(fed.Spec.MobileNetworkIds.Mncs) != 0 {
			fedReq.OrigOPMobileNetworkCodes = &opgmodels.MobileNetworkIds{
				Mcc:  &fed.Spec.MobileNetworkIds.Mcc,
				Mncs: &fed.Spec.MobileNetworkIds.Mncs,
			}
		}
	}
	// Get the OPG client for the federation
	opgClient := r.GetOPGClient(
		fed.Status.FederationContextId,
		tokenUrl,
		fed.Spec.FederationData.ClientId,
	)

	// Call the OPG API to create the federation
	res, err := opgClient.CreateFederationWithResponse(context.TODO(), fedReq)
	if err != nil {
		log.Error(err, ">>> [Federation][REST] Error Creating resource status", "name", fed.Name, "namespace", fed.Namespace)
		return err
	}

	// Handle the response based on the status code
	statusCode := res.StatusCode()

	switch {
	case statusCode == 200:
		federResponse := res.JSON200
		log.Info(">>> [Federation][REST] 200 (Create) - Federation meta-info request accepted.", "name", fed.Name, "namespace", fed.Namespace)
		var zones []v1beta1.ZoneDetails
		if federResponse.OfferedAvailabilityZones != nil {
			offeredZones := *federResponse.OfferedAvailabilityZones
			// Preallochiamo la slice per evitare riallocazioni di memoria durante il ciclo
			zones = make([]v1beta1.ZoneDetails, 0, len(offeredZones))

			for _, z := range offeredZones {
				var geoLocation string
				if z.Geolocation != nil {
					geoLocation = *z.Geolocation
				}
				zones = append(zones, v1beta1.ZoneDetails{
					GeographyDetails: z.GeographyDetails,
					Geolocation:      geoLocation,
					ZoneId:           z.ZoneId,
				})
			}
		}

		// Early return: se non ci sono cambiamenti sostanziali, usciamo prima di copiare tutti i campi
		if CompareSameAZs(fed.Status.ZoneDetails, zones) && fed.Status.State == v1beta1.FederationStateAvailable {
			return nil
		}

		fed.Status.ZoneDetails = zones

		if federResponse.FederationExpiryDate != nil {
			fed.Status.FederationExpiryDate = metav1.Time{Time: *federResponse.FederationExpiryDate}
		}
		if federResponse.FederationRenewalDate != nil {
			fed.Status.FederationRenewalDate = metav1.Time{Time: *federResponse.FederationRenewalDate}
		}
		if federResponse.FederationContextId != nil {
			fed.Status.FederationContextId = *federResponse.FederationContextId
		}
		if edse := federResponse.EdgeDiscoveryServiceEndPoint; edse != nil {
			endpoint := v1beta1.ServiceEndpoint{
				Fqdn: *edse.Fqdn,
				Port: edse.Port,
			}

			if edse.Ipv4Addresses != nil {
				source := *edse.Ipv4Addresses
				for i, ip := range source {
					// Conversione esplicita da models.Ipv4Addr a v1beta1.IPv4String
					endpoint.Ipv4Addresses[i] = ip
				}
			}

			if edse.Ipv6Addresses != nil {
				source := *edse.Ipv6Addresses
				for i, ip := range source {
					endpoint.Ipv6Addresses[i] = ip.(string)
				}
			}

			fed.Status.EdgeDiscoveryServiceEndPoint = &endpoint
		}

		if lse := federResponse.LcmServiceEndPoint; lse != nil {
			endpoint := v1beta1.ServiceEndpoint{
				Fqdn: *lse.Fqdn,
				Port: lse.Port,
			}

			if lse.Ipv4Addresses != nil {
				source := *lse.Ipv4Addresses
				for i, ip := range source {
					endpoint.Ipv4Addresses[i] = ip
				}
			}

			if lse.Ipv6Addresses != nil {
				source := *lse.Ipv6Addresses
				for i, ip := range source {
					endpoint.Ipv6Addresses[i] = ip.(string)
				}
			}

			fed.Status.LcmServiceEndPoint = &endpoint
		}

		if federResponse.PartnerOPFederationId != nil {
			fed.Status.PartnerOPFederationId = *federResponse.PartnerOPFederationId
		}
		if federResponse.PartnerOPCountryCode != nil {
			fed.Status.PartnerOPCountryCode = *federResponse.PartnerOPCountryCode
		}
		if federResponse.PartnerOPFixedNetworkCodes != nil {
			fed.Status.FixedNetworkIds = *federResponse.PartnerOPFixedNetworkCodes
		}

		if pmnc := federResponse.PartnerOPMobileNetworkCodes; pmnc != nil {
			fed.Status.MobileNetworkIds = &v1beta1.MobileNetworkIds{
				Mcc:  *pmnc.Mcc,
				Mncs: *pmnc.Mncs,
			}
		}
		fed.Status.FederationContextId = string(*federResponse.FederationContextId)
		fed.Status.State = v1beta1.FederationStateAvailable
	case statusCode == 400:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 401:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 404:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 408:
		return fmt.Errorf("Federation %s not established (408 Timeout)", fed.Name)
	case statusCode == 409:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 422:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 500:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 503:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 520:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		fed.Status.State = v1beta1.FederationStateFailed
	default:
		log.Info(">>> [Federation][REST] Unexpected Status Code", "status", statusCode, "body", string(res.Body))
		fed.Status.State = v1beta1.FederationStateNotAvailable
	}
	return nil
}
func (r *FederationReconciler) PatchFederation(ctx context.Context, fed *v1beta1.Federation) error {
	log := ctrl.Log
	log.Info(">>> [Federation][REST] Updating DATA for Federation.", "name", fed.Name, "namespace", fed.Namespace)
	// Get the OPG client for the federation
	opgClient := r.GetOPGClient(
		fed.Status.FederationContextId,
		fed.Spec.FederationData.RestOptions.TokenUrl,
		fed.Spec.FederationData.ClientId,
	)

	params := opgmodels.UpdateFederationJSONRequestBody{
		AddFixedNetworkIds:  &fed.Spec.UpdateData.AddFixedNetworkIds,
		AddMobileNetworkIds: &opgmodels.MobileNetworkIds{Mcc: &fed.Spec.UpdateData.AddMobileNetworkIds.Mcc, Mncs: &fed.Spec.UpdateData.AddMobileNetworkIds.Mncs},
		AssocAppPolicies: &opgmodels.AssocApplPolicies{
			PolicyId: fed.Spec.UpdateData.AssocAppPolicies.PolicyId,
			AppIdList: opgmodels.AppIdLocList{
				AppId:     fed.Spec.UpdateData.AssocAppPolicies.AppIdList.AppId,
				AppProvId: fed.Spec.UpdateData.AssocAppPolicies.AppIdList.AppProvId,
				ZoneIds:   &(fed.Spec.UpdateData.AssocAppPolicies.AppIdList.ZoneIds),
			},
		},
		AssocOpsPolicies: &opgmodels.AssocOpsPolicies{
			PolicyId: fed.Spec.UpdateData.AssocOpsPolicies.PolicyId,
			AppIdList: opgmodels.AppIdLocList{
				AppId:     fed.Spec.UpdateData.AssocOpsPolicies.AppIdList.AppId,
				AppProvId: fed.Spec.UpdateData.AssocOpsPolicies.AppIdList.AppProvId,
				ZoneIds:   &(fed.Spec.UpdateData.AssocOpsPolicies.AppIdList.ZoneIds),
			},
		},
		ModificationDate:       fed.Spec.UpdateData.ModificationDate.Time,
		ObjectType:             models.UpdateFederationJSONBodyObjectType(fed.Spec.UpdateData.ObjectType),
		OperationType:          models.UpdateFederationJSONBodyOperationType(fed.Spec.UpdateData.OperationType),
		RemoveFixedNetworkIds:  &fed.Spec.UpdateData.RemoveFixedNetworkIds,
		RemoveMobileNetworkIds: &opgmodels.MobileNetworkIds{Mcc: &fed.Spec.UpdateData.RemoveMobileNetworkIds.Mcc, Mncs: &fed.Spec.UpdateData.RemoveMobileNetworkIds.Mncs},
	}

	// Call the OPG API to get the platform capabilities
	res, err := opgClient.UpdateFederationWithResponse(
		context.TODO(),
		fed.Status.FederationContextId,
		params,
	)
	if err != nil {
		log.Error(err, ">>> [Federation][REST] Error UPDATING DATA for Federation.", "name", fed.Name, "namespace", fed.Namespace)
		return err
	}

	// Handle the response based on the status code
	statusCode := res.StatusCode()
	switch {
	case statusCode == 200:
		updateResponse := res.JSON200
		log.Info(">>> [Federation][REST] 200 (Update) - Federation meta-info request accepted", "name", fed.Name, "namespace", fed.Namespace)
		var zones []v1beta1.ZoneDetails
		offeredZones := *updateResponse.OfferedAvailabilityZones
		// Preallochiamo la slice per evitare riallocazioni di memoria durante il ciclo
		zones = make([]v1beta1.ZoneDetails, 0, len(offeredZones))
		for _, z := range offeredZones {
			var geoLocation string
			if z.Geolocation != nil {
				geoLocation = *z.Geolocation
			}
			zones = append(zones, v1beta1.ZoneDetails{
				GeographyDetails: z.GeographyDetails,
				Geolocation:      geoLocation,
				ZoneId:           z.ZoneId,
			})
		}
		// Early return: se non ci sono cambiamenti sostanziali, usciamo prima di copiare tutti i campi
		if CompareSameAZs(fed.Status.ZoneDetails, zones) && fed.Status.State == v1beta1.FederationStateAvailable {
			return nil
		}

		fed.Status.ZoneDetails = zones
		fed.Status.FixedNetworkIds = *updateResponse.AllowedFixedNetworkIds
		fed.Status.MobileNetworkIds = &v1beta1.MobileNetworkIds{
			Mcc:  *updateResponse.AllowedMobileNetworkIds.Mcc,
			Mncs: *updateResponse.AllowedMobileNetworkIds.Mncs,
		}
		fed.Status.EdgeDiscoveryServiceEndPoint = mapServiceEndpointToK8s(updateResponse.EdgeDiscoveryServiceEndPoint)
		fed.Status.LcmServiceEndPoint = mapServiceEndpointToK8s(updateResponse.LcmServiceEndPoint)

	case statusCode == 400:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 401:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 404:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 409:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 422:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 500:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 503:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 520:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		fed.Status.State = v1beta1.FederationStateFailed
	default:
		log.Info(">>> [Federation][REST] Unexpected Status Code", "name", fed.Name, "namespace", fed.Namespace, "status", statusCode)
		fed.Status.State = v1beta1.FederationStateFailed
	}
	return nil
}
func (r *FederationReconciler) GetHealthFederation(ctx context.Context, fed *v1beta1.Federation) error {
	log := ctrl.Log
	log.Info(">>> [Federation][REST] Getting HEALTH INFO for Federation.", "name", fed.Name, "namespace", fed.Namespace)
	// Get the OPG client for the federation
	opgClient := r.GetOPGClient(
		fed.Status.FederationContextId,
		fed.Spec.FederationData.RestOptions.TokenUrl,
		fed.Spec.FederationData.ClientId,
	)

	// Call the OPG API to get the health information for the federation
	res, err := opgClient.GetFederationHealthWithResponse(
		context.TODO(),
		fed.Status.FederationContextId,
	)
	if err != nil {
		log.Error(err, ">>> [Federation][REST] Error getting HEALTH INFO for Federation.", "name", fed.Name, "namespace", fed.Namespace)
		return err
	}

	// Handle the response based on the status code
	statusCode := res.StatusCode()
	switch {
	case statusCode == 200:
		log.Info(">>> [Federation][REST] 200 - Federation health status information object.", "name", fed.Name, "namespace", fed.Namespace)
		healthInfoResponse := res.JSON200.FederationHealthStatus
		fed.Status.FederationHealthInfo.DateAndTimeZoneObject = metav1.Time{Time: healthInfoResponse.FederationStartTime}
		fed.Status.FederationHealthInfo.AlarmState = string(healthInfoResponse.FederationStatus.AlarmState)
		fed.Status.FederationHealthInfo.NumOfAcceptedZones = string(healthInfoResponse.NumOfAcceptedZones)
		fed.Status.FederationHealthInfo.NumOfActiveAlarms = *healthInfoResponse.NumOfActiveAlarms
		fed.Status.FederationHealthInfo.NumOfApplications = *healthInfoResponse.NumOfApplications
	case statusCode == 400:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 401:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 404:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 409:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 422:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 500:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 503:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 520:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		fed.Status.State = v1beta1.FederationStateFailed
	default:
		log.Info(">>> [Federation][REST] Unexpected Status Code", "name", fed.Name, "namespace", fed.Namespace, "status", statusCode)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	}
	fed.Annotations[v1beta1.GetHealthInfoAnnotation] = "not-required"
	return nil
}
func (r *FederationReconciler) GetPlatformCapsFederation(ctx context.Context, fed *v1beta1.Federation) error {
	log := ctrl.Log
	log.Info(">>> [Federation][REST] Getting PLATFORM CAPS for Federation.", "name", fed.Name, "namespace", fed.Namespace)
	// Get the OPG client for the federation
	opgClient := r.GetOPGClient(
		fed.Status.FederationContextId,
		fed.Spec.FederationData.RestOptions.TokenUrl,
		fed.Spec.FederationData.ClientId,
	)

	params := &opgmodels.GetPlatformCapabilitiesParams{}
	if fed.Spec.CapType != "" {
		capTypeValue := opgmodels.CapabilityID(fed.Spec.CapType)
		params.CapType = &(capTypeValue)
	}

	// Call the OPG API to get the platform capabilities
	res, err := opgClient.GetPlatformCapabilitiesWithResponse(
		context.TODO(),
		fed.Status.FederationContextId,
		params,
	)
	if err != nil {
		log.Error(err, ">>> [Federation][REST] Error getting PLATFORM CAPS for Federation.", "name", fed.Name, "namespace", fed.Namespace)
		return err
	}

	// Handle the response based on the status code
	statusCode := res.StatusCode()
	switch {
	case statusCode == 200:
		log.Info(">>> [Federation][REST] Retrieved PLATFORM CAPS capabilities successfully", "name", fed.Name, "namespace", fed.Namespace)
		platformCapResonose := res.JSON200
		if platformCapResonose != nil {
			fed.Status.DeviceConnStatusChangeCap = &v1beta1.Caps{
				CapabilityId:      string(platformCapResonose.DeviceConnStatusChangeCap.CapabilityId),
				MaxiDetectionTime: platformCapResonose.DeviceConnStatusChangeCap.MaxiDetectionTime,
			}
			fed.Status.LocationRetrievalCap = &v1beta1.Caps{
				CapabilityId:     string(platformCapResonose.LocationRetrievalCap.CapabilityId),
				LocationType:     string(platformCapResonose.LocationRetrievalCap.LocationType),
				LocationAccuracy: string(*platformCapResonose.LocationRetrievalCap.LocationAccuracy),
			}
			fed.Status.UserPlaneMgmtEvtCap = &v1beta1.Caps{
				CapabilityId:        string(platformCapResonose.UserPlaneMgmtEvtCap.CapabilityId),
				MaxUserPlaneLatency: platformCapResonose.UserPlaneMgmtEvtCap.MaxUserPlaneLatency,
			}
			fed.Status.DynamicQoSCap = &v1beta1.Caps{
				CapabilityId: string(platformCapResonose.DynamicQoSCap.CapabilityId),
				SupportedQoS: platformCapResonose.DynamicQoSCap.SupportedQoS,
			}
		}
	case statusCode == 400:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	case statusCode == 401:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	case statusCode == 404:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	case statusCode == 409:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	case statusCode == 422:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	case statusCode == 500:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	case statusCode == 503:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	case statusCode == 520:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	default:
		log.Info(">>> [Federation][REST] Unexpected Status Code", "name", fed.Name, "namespace", fed.Namespace, "status", statusCode)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	}
	fed.Annotations[v1beta1.GetPlatformCapsAnnotation] = "not-required"
	return nil
}
func (r *FederationReconciler) GetServiceAPIFederation(ctx context.Context, fed *v1beta1.Federation) error {
	log := ctrl.Log
	log.Info(">>> [Federation][REST] Getting SERVICE APIs details.", "name", fed.Name, "namespace", fed.Namespace)
	// Get the OPG client for the federation
	opgClient := r.GetOPGClient(
		fed.Status.FederationContextId,
		fed.Spec.FederationData.RestOptions.TokenUrl,
		fed.Spec.FederationData.ClientId,
	)
	// Call the OPG API to get the platform capabilities
	res, err := opgClient.GetServiceAPIsDetailsWithResponse(
		context.TODO(),
		fed.Status.FederationContextId,
		"api_federation",
	)
	if err != nil {
		log.Error(err, ">>> [Federation][REST] Error getting SERVICE APIs details.", "name", fed.Name, "namespace", fed.Namespace)
		return err
	}

	// Handle the response based on the status code
	statusCode := res.StatusCode()
	switch {
	case statusCode == 200:
		log.Info(">>> [Federation][REST] 200 - List of SERVICE APIs names and associated configuration info as supported capabilities.", "name", fed.Name, "namespace", fed.Namespace)
		serviceResponse := res.JSON200
		fed.Status.Service = &v1beta1.Service{
			ServiceCaps:    serviceResponse.ServiceCaps,
			ServiceType:    string(*serviceResponse.ServiceType),
			ApiRoutingInfo: serviceResponse.ApiRoutingInfo,
		}
	case statusCode == 400:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	case statusCode == 401:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	case statusCode == 404:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	case statusCode == 409:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	case statusCode == 422:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	case statusCode == 500:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	case statusCode == 503:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	case statusCode == 520:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	default:
		log.Info(">>> [Federation][REST] Unexpected Status Code", "name", fed.Name, "namespace", fed.Namespace, "status", statusCode)
		fed.Status.State = v1beta1.FederationStateTemporaryFailure
	}
	fed.Annotations[v1beta1.GetServiceAPIsAnnotation] = "not-required"
	return nil
}
func (r *FederationReconciler) RenewalFederation(ctx context.Context, fed *v1beta1.Federation) error {
	log := ctrl.Log
	log.Info(">>> [Federation][REST] Getting RENEW FEDERATION.", "name", fed.Name, "namespace", fed.Namespace)
	// Get the OPG client for the federation
	opgClient := r.GetOPGClient(
		fed.Status.FederationContextId,
		fed.Spec.FederationData.RestOptions.TokenUrl,
		fed.Spec.FederationData.ClientId,
	)
	// Call the OPG API to get the platform capabilities
	res, err := opgClient.RenewFederationWithResponse(
		context.TODO(),
		fed.Status.FederationContextId,
	)
	if err != nil {
		log.Error(err, ">>> [Federation][REST] Error getting RENEWAL FEDERATION.", "name", fed.Name, "namespace", fed.Namespace)
		return err
	}

	// Handle the response based on the status code
	statusCode := res.StatusCode()
	switch {
	case statusCode == 200:
		log.Info(">>> [Federation][REST] 200 - Federation renewal request accepted.", "name", fed.Name, "namespace", fed.Namespace)
		renewalResponse := res.JSON200
		fed.Status.FederationExpiryDate = metav1.Time{Time: renewalResponse.FederationExpiryDate}
		fed.Status.FederationRenewalDate = metav1.Time{Time: renewalResponse.FederationRenewalDate}
		fed.Status.State = v1beta1.FederationStateAvailable
		fed.Status.FederationContextId = *renewalResponse.FederationContextId
	case statusCode == 400:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 401:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 404:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 409:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 422:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 500:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 503:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 520:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		fed.Status.State = v1beta1.FederationStateFailed
	default:
		log.Info(">>> [Federation][REST] Unexpected Status Code", "name", fed.Name, "namespace", fed.Namespace, "status", statusCode)
		fed.Status.State = v1beta1.FederationStateFailed
	}
	fed.Annotations[v1beta1.FederationRenewalAnnotation] = "not-required"
	return nil
}
func (r *FederationReconciler) DeleteFederation(ctx context.Context, fed *v1beta1.Federation) error {
	log := ctrl.Log
	log.Info(">>> [Federation][REST] Deleting external federation", "name", fed.Name, "namespace", fed.Namespace)
	// Get the OPG client for the federation
	opgClient := r.GetOPGClient(
		fed.Status.FederationContextId,
		fed.Spec.FederationData.RestOptions.TokenUrl,
		fed.Spec.FederationData.ClientId,
	)
	// Call the OPG API to delete the federation
	res, err := opgClient.DeleteFederationDetailsWithResponse(
		context.TODO(),
		fed.Status.FederationContextId,
	)
	if err != nil {
		log.Error(err, ">>> [Federation][REST] Error deleting external federation", "name", fed.Name, "namespace", fed.Namespace)
		return err
	}

	// Handle the response based on the status code
	statusCode := res.StatusCode()

	switch {
	case statusCode == 200:
		log.Info(">>> [Federation][REST] 200 - Federation removed successfully.", "name", fed.Name, "namespace", fed.Namespace)
	case statusCode == 400:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 401:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 404:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 409:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 422:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 500:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 503:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 520:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		fed.Status.State = v1beta1.FederationStateFailed
	default:
		log.Info(">>> [Federation][REST] Unexpected Status Code", "name", fed.Name, "namespace", fed.Namespace, "status", statusCode)
		fed.Status.State = v1beta1.FederationStateFailed
	}
	return nil
}

// Update (Callback)
func (r *FederationReconciler) UpdateFederationStatus(ctx context.Context, fed *v1beta1.Federation) error {
	log := ctrl.Log
	log.Info(">>> [Federation][REST] UpdateFederationStatus not implemented.", "name", fed.Name, "namespace", fed.Namespace)
	return nil
}

func (r *FederationReconciler) DetailsFederation(ctx context.Context, fed *v1beta1.Federation) error {
	log := ctrl.Log
	log.Info(">>> [Federation][REST] Calling DetailsFederation", "name", fed.Name, "namespace", fed.Namespace)

	callbackBody := opgmodels.PartnerDetailsJSONRequestBody{
		FederationContextId: &fed.Status.FederationContextId,
	}
	log.Info(">>> [Federation][REST] PartnerDetails body", "body", callbackBody)

	opgClient := r.GetOPGClient(
		fed.Status.FederationContextId,
		fed.Spec.FederationData.RestOptions.TokenUrl,
		fed.Spec.FederationData.ClientId,
	)

	res, err := opgClient.PartnerDetailsWithResponse(
		context.TODO(),
		fed.Spec.FederationData.ClientId,
		callbackBody,
	)

	if err != nil {
		log.Error(err, ">>> [Federation][REST] Error sending App callback")
		return err
	}
	statusCode := res.StatusCode()
	switch {
	case statusCode == 200:
		log.Info(">>> [Federation][REST] 200 - Expected response to a successful call back signup", "status", statusCode)
		if res.JSON200.EdgeDiscoveryServiceEndPoint != nil {
			fed.Status.EdgeDiscoveryServiceEndPoint = &v1beta1.ServiceEndpoint{
				Fqdn:          *res.JSON200.EdgeDiscoveryServiceEndPoint.Fqdn,
				Port:          res.JSON200.EdgeDiscoveryServiceEndPoint.Port,
				Ipv4Addresses: *res.JSON200.EdgeDiscoveryServiceEndPoint.Ipv4Addresses,
				// Ipv6Addresses: *res.JSON200.EdgeDiscoveryServiceEndPoint.Ipv6Addresses,
			}
		}
		if res.JSON200.LcmServiceEndPoint != nil {
			fed.Status.LcmServiceEndPoint = &v1beta1.ServiceEndpoint{
				Fqdn:          *res.JSON200.LcmServiceEndPoint.Fqdn,
				Port:          res.JSON200.LcmServiceEndPoint.Port,
				Ipv4Addresses: *res.JSON200.LcmServiceEndPoint.Ipv4Addresses,
				// Ipv6Addresses: *res.JSON200.LcmServiceEndPoint.Ipv6Addresses,
			}
		}
		if res.JSON200.OfferedAvailabilityZones != nil {
			offeredZones := make([]v1beta1.ZoneDetails, len(*res.JSON200.OfferedAvailabilityZones))
			for i, zd := range *res.JSON200.OfferedAvailabilityZones {
				var geoLocation string
				if zd.Geolocation != nil {
					geoLocation = *zd.Geolocation
				}
				offeredZones[i] = v1beta1.ZoneDetails{
					GeographyDetails: zd.GeographyDetails,
					Geolocation:      geoLocation,
					ZoneId:           zd.ZoneId,
				}
			}
			fed.Status.ZoneDetails = offeredZones
		}
		if res.JSON200.PartnerOPCountryCode != nil {
			fed.Status.PartnerOPCountryCode = *res.JSON200.PartnerOPCountryCode
		}
		if res.JSON200.PartnerOPFederationId != nil {
			fed.Status.PartnerOPFederationId = *res.JSON200.PartnerOPFederationId
		}
		if res.JSON200.PartnerOPFixedNetworkCodes != nil {
			fed.Status.FixedNetworkIds = *res.JSON200.PartnerOPFixedNetworkCodes
		}
		if res.JSON200.PartnerOPMobileNetworkCodes != nil {
			fed.Status.MobileNetworkIds = &v1beta1.MobileNetworkIds{
				Mcc:  *res.JSON200.PartnerOPMobileNetworkCodes.Mcc,
				Mncs: *res.JSON200.PartnerOPMobileNetworkCodes.Mncs,
			}
		}
		fed.Status.FederationExpiryDate = metav1.NewTime(res.JSON200.FederationExpiryDate)
		fed.Status.FederationRenewalDate = metav1.NewTime(res.JSON200.FederationRenewalDate)
		fed.Status.PlatformCaps = res.JSON200.PlatformCaps
	case statusCode == 400:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 401:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 404:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 409:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 422:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 500:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 503:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 520:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		fed.Status.State = v1beta1.FederationStateFailed
	default:
		log.Info(">>> [Federation][REST] Unexpected Status Code", "name", fed.Name, "namespace", fed.Namespace, "status", statusCode)
		fed.Status.State = v1beta1.FederationStateFailed
	}
	return nil
}

func (r *FederationReconciler) UpdateFederationDetailsStatus(ctx context.Context, fed *v1beta1.Federation) error {
	log := ctrl.Log

	// Check if callback is configured
	if fed.Spec.FederationData.RestOptions.PartnerStatusLink == "" {
		log.Info(">>> [AppDep][REST] No callback StatusLink configured in Federation, skipping callback")
		return nil
	}
	if fed.Status.PlatformCaps == nil {
		return fmt.Errorf("Missing platform caps for federation %s", fed.Name)
	}

	var offeredZones []models.ZoneDetails
	if len(fed.Status.ZoneDetails) > 0 {
		offeredZones := make([]models.ZoneDetails, len(fed.Status.ZoneDetails))
		for i, zd := range fed.Status.ZoneDetails {
			offeredZones[i] = models.ZoneDetails{
				ZoneId:           zd.ZoneId,
				Geolocation:      &zd.Geolocation,
				GeographyDetails: zd.GeographyDetails,
			}
		}
	}
	var partnerMobileNetCodes *models.MobileNetworkIds
	if fed.Status.MobileNetworkIds != nil {
		partnerMobileNetCodes = &models.MobileNetworkIds{
			Mcc:  &fed.Status.MobileNetworkIds.Mcc,
			Mncs: &fed.Status.MobileNetworkIds.Mncs,
		}
	}
	var edgeDiscoveryServiceEndPoint *models.ServiceEndpoint
	if fed.Status.EdgeDiscoveryServiceEndPoint != nil {
		edgeDiscoveryServiceEndPoint = &models.ServiceEndpoint{
			Fqdn:          &fed.Status.EdgeDiscoveryServiceEndPoint.Fqdn,
			Ipv4Addresses: &fed.Status.EdgeDiscoveryServiceEndPoint.Ipv4Addresses,
			// Ipv6Addresses: &k8sFed.Status.EdgeDiscoveryServiceEndPoint.Ipv6Addresses,
			Port: fed.Status.EdgeDiscoveryServiceEndPoint.Port,
		}
	}
	var lcmServiceEndPoint *models.ServiceEndpoint
	if fed.Status.LcmServiceEndPoint != nil {
		lcmServiceEndPoint = &models.ServiceEndpoint{
			Fqdn:          &fed.Status.LcmServiceEndPoint.Fqdn,
			Ipv4Addresses: &fed.Status.LcmServiceEndPoint.Ipv4Addresses,
			// Ipv6Addresses: &k8sFed.Status.LcmServiceEndPoint.Ipv6Addresses,
			Port: fed.Status.LcmServiceEndPoint.Port,
		}
	}

	log.Info(">>> [Federation][REST] Sending App callback to Guest", "name", fed.Name, "namespace", fed.Namespace, "callbackURL", fed.Spec.FederationData.RestOptions.PartnerStatusLink)

	callbackBody := opgmodels.PartnerDetailsCallbackJSONRequestBody{
		EdgeDiscoveryServiceEndPoint: edgeDiscoveryServiceEndPoint,
		LcmServiceEndPoint:           lcmServiceEndPoint,
		OfferedAvailabilityZones:     &offeredZones,
		PartnerOPMobileNetworkCodes:  partnerMobileNetCodes,
		FederationExpiryDate:         fed.Status.FederationExpiryDate.Time,
		FederationRenewalDate:        fed.Status.FederationRenewalDate.Time,
		PartnerOPCountryCode:         &fed.Status.PartnerOPCountryCode,
		PartnerOPFederationId:        &fed.Status.PartnerOPFederationId,
		PartnerOPFixedNetworkCodes:   &fed.Status.FixedNetworkIds,
		PlatformCaps:                 fed.Status.PlatformCaps,
	}
	log.Info(">>> [Federation][REST] Callback body", "body", callbackBody)
	// Get callback client (pointing to Guest's callback URL via Federation.spec.partner.statusLink)
	opgClient := r.GetOPGClient(
		fed.Status.FederationContextId,
		fed.Spec.FederationData.RestOptions.PartnerStatusLink,
		"host",
	)

	res, err := opgClient.PartnerDetailsCallbackWithResponse(
		context.TODO(),
		fed.Status.FederationContextId,
		callbackBody,
	)

	if err != nil {
		log.Error(err, ">>> [Federation][REST] Error sending App callback")
		return err
	}
	statusCode := res.StatusCode()
	switch {
	case statusCode == 204:
		log.Info(">>> [Federation][REST] 204 (Callback) - Expected response to a successful call back processing", "status", statusCode)
	case statusCode == 400:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON400)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 401:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON401)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 404:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON404)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 409:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON409)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 422:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON422)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 500:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON500)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 503:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON503)
		fed.Status.State = v1beta1.FederationStateFailed
	case statusCode == 520:
		handleFederationProblemDetails(log, statusCode, res.ApplicationproblemJSON520)
		fed.Status.State = v1beta1.FederationStateFailed
	default:
		log.Info(">>> [Federation][REST] Unexpected Status Code", "name", fed.Name, "namespace", fed.Namespace, "status", statusCode)
		fed.Status.State = v1beta1.FederationStateFailed
	}
	return nil
}
