package metastore

import (
	"context"
	"reflect"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	v1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/uuid"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8scli "sigs.k8s.io/controller-runtime/pkg/client"
)

type Federation struct {
	*models.FederationRequestData
	ClientCredentials        ClientCredentials
	FederationContextId      models.FederationContextId
	OfferedAvailabilityZones *[]models.ZoneDetails
}

// func isValidFederationStatus(status string) bool {
// 	switch v1beta1.FederationState(status) {
// 	case v1beta1.FederationStateFailed, v1beta1.FederationStateTemporaryFailure, v1beta1.FederationStateAvailable, v1beta1.FederationStateLocked, v1beta1.FederationStateNotAvailable:
// 		return true
// 	}
// 	return false
// }

func (c *k8sClient) searchFederation(ctx context.Context, federationContextId string, role string) (*v1beta1.Federation, error) {
	var fedList v1beta1.FederationList
	if err := c.kubernetes.List(ctx, &fedList, client.InNamespace(c.namespace)); err != nil {
		return nil, err
	}
	if len(fedList.Items) == 0 {
		return nil, errors.Errorf("Federation not found for federationContextId: %s and role: %s", federationContextId, role)
	}
	for i := range fedList.Items {
		fed := fedList.Items[i]
		if fed.Spec.FederationData.RelationType == role && fed.Status.FederationContextId == federationContextId {
			return &fed, nil
		}
	}
	return nil, errors.Errorf("Federation not found for federationContextId: %s and role: %s", federationContextId, role)
}

func (c *k8sClient) CreateFederation(ctx context.Context, fed *Federation) (*v1beta1.Federation, error) {
	fedId := uuid.V5(*fed.OrigOPFederationId + *fed.OrigOPCountryCode)
	fedHost := &v1beta1.Federation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fed-" + fedId,
			Namespace: c.namespace,
		},
		Spec: v1beta1.FederationSpec{
			FederationData: &v1beta1.FederationData{
				RelationType:       string(v1beta1.FederationRelationHost),
				TechnologyType:     string(v1beta1.FederationTechnologyRest),
				OrigOPFederationId: *fed.OrigOPFederationId,
				InitialDate:        metav1.Time{Time: fed.InitialDate},
				OrigOPCountryCode:  string(*fed.OrigOPCountryCode),
				RestOptions: &v1beta1.RestOptions{
					PartnerStatusLink: fed.PartnerStatusLink,
				},
			},
		},
	}
	if fed.OrigOPFixedNetworkCodes != nil {
		fedHost.Spec.FixedNetworkIds = *fed.OrigOPFixedNetworkCodes
	}
	if fed.OrigOPMobileNetworkCodes != nil {
		fedHost.Spec.MobileNetworkIds = &v1beta1.MobileNetworkIds{}

		if fed.OrigOPMobileNetworkCodes.Mcc != nil {
			fedHost.Spec.MobileNetworkIds.Mcc = *fed.OrigOPMobileNetworkCodes.Mcc
		}
		if fed.OrigOPMobileNetworkCodes.Mncs != nil {
			fedHost.Spec.MobileNetworkIds.Mncs = *fed.OrigOPMobileNetworkCodes.Mncs
		}
	}
	if err := c.createK8sObject(fedHost); err != nil {
		return nil, err
	}
	return fedHost, nil
}

func (c *k8sClient) GetFederation(ctx context.Context, federationContextID string) (*Federation, error) {
	fed, err := c.searchFederation(ctx, federationContextID, "HOST")
	if err != nil {
		return nil, err
	}
	offeredZones := make([]models.ZoneDetails, len(fed.Status.ZoneDetails))
	for i, z := range fed.Status.ZoneDetails {
		offeredZones[i] = models.ZoneDetails{
			ZoneId:           z.ZoneId,
			Geolocation:      &(z.Geolocation),
			GeographyDetails: z.GeographyDetails,
		}
	}

	return &Federation{
		FederationRequestData: &models.FederationRequestData{
			InitialDate:             fed.Spec.FederationData.InitialDate.Time,
			OrigOPCountryCode:       &fed.Spec.FederationData.OrigOPCountryCode,
			OrigOPFixedNetworkCodes: &fed.Spec.FixedNetworkIds,
			OrigOPMobileNetworkCodes: &models.MobileNetworkIds{
				Mcc:  &fed.Spec.MobileNetworkIds.Mcc,
				Mncs: &fed.Spec.MobileNetworkIds.Mncs,
			},
			PartnerStatusLink: fed.Spec.FederationData.RestOptions.PartnerStatusLink,
		},
		FederationContextId:      fed.Labels[opgLabel(federationContextIDLabel)],
		OfferedAvailabilityZones: &offeredZones,
	}, nil
}

func (c *k8sClient) GetK8SFederation(ctx context.Context, federationContextID string) (*v1beta1.Federation, error) {
	fed, err := c.searchFederation(ctx, federationContextID, "HOST")
	if err != nil {
		return nil, err
	}
	return fed, nil
}
func (c *k8sClient) UpdateFederationStatus(ctx context.Context, federationCallbackID string, updates *models.PartnerStatusLinkJSONRequestBody) error {
	fed, err := c.searchFederation(ctx, federationCallbackID, "GUEST")
	if err != nil {
		return err
	}
	originalFed := fed.DeepCopy()
	zones := fed.Status.ZoneDetails

	if updates.ZoneStatus != nil {
		// Creiamo una mappa per una ricerca più veloce e pulita
		statusMap := make(map[string]string)
		for _, uz := range *updates.ZoneStatus {
			statusMap[uz.ZoneId] = string(uz.Status)
		}

		// Iteriamo con l'indice (i) per modificare direttamente l'elemento nell'array
		for i, z := range zones {
			if newStatus, exists := statusMap[z.ZoneId]; exists {
				zones[i].Status = newStatus
			}
		}
	}
	if updates.RemoveZones != nil {
		// Creiamo una mappa degli ID da rimuovere per comodità
		toRemove := make(map[string]bool)
		for _, rz := range *updates.RemoveZones {
			toRemove[rz] = true // Assumo che rz abbia un campo ZoneId o sia esso stesso l'ID
		}

		var filteredZones []v1beta1.ZoneDetails
		for _, z := range zones {
			if !toRemove[z.ZoneId] {
				filteredZones = append(filteredZones, z) // Teniamo solo quelle non rimosse
			}
		}
		zones = filteredZones
	}

	if updates.AddZones != nil {
		for _, az := range *updates.AddZones {
			// Aggiungiamo le nuove zone
			zones = append(zones, v1beta1.ZoneDetails{
				ZoneId:           az.ZoneId,
				Geolocation:      *az.Geolocation,
				GeographyDetails: az.GeographyDetails,
				Status:           string(v1beta1.ZoneStateNotAvailable),
			})
		}
	}
	removeFromSlice := func(slice []string, toRemove []string) []string {
		removeMap := make(map[string]bool)
		for _, val := range toRemove {
			removeMap[val] = true
		}
		var result []string
		for _, val := range slice {
			if !removeMap[val] {
				result = append(result, val)
			}
		}
		return result
	}
	fed.Status.ZoneDetails = zones
	if updates.RemoveMobileNetworkIds != nil {
		if updates.RemoveMobileNetworkIds.Mcc != nil {
			if fed.Status.MobileNetworkIds.Mcc == *updates.RemoveMobileNetworkIds.Mcc {
				fed.Status.MobileNetworkIds.Mcc = ""
			}
		}
		if updates.RemoveMobileNetworkIds.Mncs != nil {
			fed.Status.MobileNetworkIds.Mncs = removeFromSlice(fed.Status.MobileNetworkIds.Mncs, *updates.RemoveMobileNetworkIds.Mncs)
		}
	}

	if updates.AddMobileNetworkIds != nil {
		if !reflect.DeepEqual(fed.Status.MobileNetworkIds, &v1beta1.MobileNetworkIds{}) {
			fed.Status.MobileNetworkIds = &v1beta1.MobileNetworkIds{}
		}
		if updates.AddMobileNetworkIds.Mcc != nil {
			fed.Status.MobileNetworkIds.Mcc = *updates.AddMobileNetworkIds.Mcc
		}
		if updates.AddMobileNetworkIds.Mncs != nil {
			fed.Status.MobileNetworkIds.Mncs = append(fed.Status.MobileNetworkIds.Mncs, *updates.AddMobileNetworkIds.Mncs...)
		}
	}
	if updates.EdgeDiscoverySvcEndPoint != nil {
		fed.Status.EdgeDiscoveryServiceEndPoint = mapServiceEndpointToK8s(updates.EdgeDiscoverySvcEndPoint)
	}
	if updates.LcmSvcEndPoint != nil {
		fed.Status.LcmServiceEndPoint = mapServiceEndpointToK8s(updates.LcmSvcEndPoint)
	}
	fed.Status.UpdateDetails = &v1beta1.UpdateDetails{
		UpdateDate:    metav1.Time{Time: updates.ModificationDate},
		ObjectType:    string(updates.ObjectType),
		OperationType: string(updates.OperationType),
	}
	return c.patchK8sStatus(originalFed, fed)

}

func (c *k8sClient) RemoveFederation(ctx context.Context, federationContextID string) error {
	fed, err := c.searchFederation(ctx, federationContextID, "HOST")
	if err != nil {
		return err
	}
	if err := c.kubernetes.Delete(context.TODO(), fed, &k8scli.DeleteOptions{}); err != nil {
		return errors.Wrapf(err, "unable to remove federation")
	}
	return nil
}

func (c *k8sClient) AddAvailabilityZones(ctx context.Context, federationContextId string, azs []string) error {
	fed, err := c.searchFederation(ctx, federationContextId, "HOST")
	if err != nil {
		return err
	}
	originalFed := fed.DeepCopy()
	zones := fed.Status.ZoneDetails
	if azs != nil {
		for _, az := range azs {
			for _, z := range zones {
				if z.ZoneId == az {
					z.Status = string(v1beta1.ZoneStateAvailable)
				}
			}
		}
	}
	return c.patchK8sStatus(originalFed, fed)
}

func (c *k8sClient) PartnerDetailsCallback(ctx context.Context, federationCallbackId models.FederationContextId, request *models.PartnerDetailsCallbackJSONRequestBody) (*v1beta1.Federation, error) {
	fed, err := c.searchFederation(ctx, federationCallbackId, "HOST")
	if err != nil {
		return nil, err
	}
	origFed := fed.DeepCopy()
	if request.EdgeDiscoveryServiceEndPoint != nil {
		fed.Status.EdgeDiscoveryServiceEndPoint = &v1beta1.ServiceEndpoint{
			Fqdn:          *request.EdgeDiscoveryServiceEndPoint.Fqdn,
			Port:          request.EdgeDiscoveryServiceEndPoint.Port,
			Ipv4Addresses: *request.EdgeDiscoveryServiceEndPoint.Ipv4Addresses,
			// Ipv6Addresses: *request.EdgeDiscoveryServiceEndPoint.Ipv6Addresses,
		}
	}
	if request.LcmServiceEndPoint != nil {
		fed.Status.LcmServiceEndPoint = &v1beta1.ServiceEndpoint{
			Fqdn:          *request.LcmServiceEndPoint.Fqdn,
			Port:          request.LcmServiceEndPoint.Port,
			Ipv4Addresses: *request.LcmServiceEndPoint.Ipv4Addresses,
			// Ipv6Addresses: *request.LcmServiceEndPoint.Ipv6Addresses,
		}
	}
	if request.OfferedAvailabilityZones != nil {
		offeredZones := make([]v1beta1.ZoneDetails, len(*request.OfferedAvailabilityZones))
		for i, zd := range *request.OfferedAvailabilityZones {
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
	if request.PartnerOPCountryCode != nil {
		fed.Status.PartnerOPCountryCode = *request.PartnerOPCountryCode
	}
	if request.PartnerOPFederationId != nil {
		fed.Status.PartnerOPFederationId = *request.PartnerOPFederationId
	}
	if request.PartnerOPFixedNetworkCodes != nil {
		fed.Status.FixedNetworkIds = *request.PartnerOPFixedNetworkCodes
	}
	if request.PartnerOPMobileNetworkCodes != nil {
		fed.Status.MobileNetworkIds = &v1beta1.MobileNetworkIds{
			Mcc:  *request.PartnerOPMobileNetworkCodes.Mcc,
			Mncs: *request.PartnerOPMobileNetworkCodes.Mncs,
		}
	}
	fed.Status.FederationExpiryDate = metav1.NewTime(request.FederationExpiryDate)
	fed.Status.FederationRenewalDate = metav1.NewTime(request.FederationRenewalDate)
	fed.Status.PlatformCaps = request.PlatformCaps

	if err := c.patchK8sStatus(origFed, fed); err != nil {
		return nil, err
	}
	return fed, nil
}

func mapServiceEndpointToK8s(apiEndpoint *models.ServiceEndpoint) *v1beta1.ServiceEndpoint {
	if apiEndpoint == nil {
		return nil // Se l'API non ci dà nulla, restituiamo nil
	}
	k8sEndpoint := &v1beta1.ServiceEndpoint{
		Port: apiEndpoint.Port,
	}

	if apiEndpoint.Fqdn != nil {
		k8sEndpoint.Fqdn = *apiEndpoint.Fqdn
	}
	if apiEndpoint.Ipv4Addresses != nil {
		source := *apiEndpoint.Ipv4Addresses
		for i, ip := range source {
			k8sEndpoint.Ipv4Addresses[i] = ip
		}
	}

	if apiEndpoint.Ipv6Addresses != nil {
		source := *apiEndpoint.Ipv6Addresses
		for i, ip := range source {
			k8sEndpoint.Ipv6Addresses[i] = ip.(string)
		}
	}

	return k8sEndpoint
}
