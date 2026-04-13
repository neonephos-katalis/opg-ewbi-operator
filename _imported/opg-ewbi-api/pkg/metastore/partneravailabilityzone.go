package metastore

import (
	opgv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/v1beta1"

	"github.com/neonephos-katalis/opg-ewbi-api/api/federation/models"
)

type PartnerAvailabilityZone struct {
	DeviceID            string                     `json:"deviceId"`
	SiteID              string                     `json:"siteId"`
	ZoneDetails         *models.ZoneDetails        `json:"zoneDetails"`
	ZoneRegisteredData  *models.ZoneRegisteredData `json:"zoneRegisteredData"`
	HostID              string                     `json:"hostId"`
	FederationContextID string                     `json:"federationContextId"`
	FederationURL       string                     `json:"federationURL"`
	FederationClientID  string                     `json:"federationClientId"`
}

// mergeUnique merges two string slices into one, removing duplicates
// while preserving the order of first appearance.
func mergeUnique(slice1, slice2 []string) []string {
	unique := make(map[string]bool)
	result := []string{}

	// Add elements from both slices to the map
	for _, val := range append(slice1, slice2...) {
		if !unique[val] {
			unique[val] = true
			result = append(result, val)
		}
	}
	return result
}

func partnerAvailabilityZoneFromK8sAvailabilityZone(az *opgv1beta1.AvailabilityZone) (*PartnerAvailabilityZone, error) {
	return &PartnerAvailabilityZone{
		ZoneDetails: &models.ZoneDetails{
			ZoneId: az.Name,
		},
	}, nil
}
