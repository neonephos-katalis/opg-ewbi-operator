package metastore

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	opgv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
)

type Federation struct {
	*models.FederationRequestData
	ClientCredentials         ClientCredentials
	FederationContextId       models.FederationContextId
	AcceptedAvailabilityZones *[]models.ZoneIdentifier
	OfferedAvailabilityZones  *[]models.ZoneDetails
}

func (f *Federation) updatek8sCustomResource(fed *opgv1beta1.Federation) *opgv1beta1.Federation {
	var aaz []string
	if f.AcceptedAvailabilityZones != nil {
		aaz = *f.AcceptedAvailabilityZones
	}
	fed.ObjectMeta.Labels[opgLabel(federationContextIDLabel)] = f.FederationContextId
	fed.ObjectMeta.Labels[opgLabel(idLabel)] = f.FederationContextId
	fed.ObjectMeta.Labels[opgLabel(federationRelation)] = host
	fed.Spec.InitialDate = metav1.Time{Time: f.InitialDate}
	fed.Spec.OriginOP = opgv1beta1.Origin{
		CountryCode:       defaultIfNil(f.OrigOPCountryCode),
		FixedNetworkCodes: *f.OrigOPFixedNetworkCodes,
		MobileNetworkCodes: opgv1beta1.MobileNetworkCodes{
			MCC: *f.OrigOPMobileNetworkCodes.Mcc,
			MNC: *f.OrigOPMobileNetworkCodes.Mncs,
		},
	}
	fed.Spec.Partner = opgv1beta1.Partner{
		CallbackCredentials: opgv1beta1.FederationCredentials{
			ClientId: f.PartnerCallbackCredentials.ClientId,
			TokenUrl: f.PartnerCallbackCredentials.TokenUrl,
		},
		StatusLink: f.PartnerStatusLink,
	}
	fed.Spec.AcceptedAvailabilityZones = aaz
	return fed
}

func federationFromK8sCustomResource(fed *opgv1beta1.Federation) (*Federation, error) {
	offeredZones := make([]models.ZoneDetails, len(fed.Spec.OfferedAvailabilityZones))
	for i, z := range fed.Spec.OfferedAvailabilityZones {
		offeredZones[i] = models.ZoneDetails{
			ZoneId:           z.ZoneId,
			Geolocation:      z.Geolocation,
			GeographyDetails: z.GeographyDetails,
		}
	}

	return &Federation{
		FederationRequestData: &models.FederationRequestData{
			InitialDate:             fed.Spec.InitialDate.Time,
			OrigOPCountryCode:       &fed.Spec.OriginOP.CountryCode,
			OrigOPFixedNetworkCodes: &fed.Spec.OriginOP.FixedNetworkCodes,
			OrigOPMobileNetworkCodes: &models.MobileNetworkIds{
				Mcc:  &fed.Spec.OriginOP.MobileNetworkCodes.MCC,
				Mncs: &fed.Spec.OriginOP.MobileNetworkCodes.MNC,
			},
			PartnerCallbackCredentials: &models.CallbackCredentials{
				ClientId: fed.Spec.Partner.CallbackCredentials.ClientId,
				TokenUrl: fed.Spec.Partner.CallbackCredentials.TokenUrl,
			},
		},
		FederationContextId:       fed.Labels[opgLabel(federationContextIDLabel)],
		OfferedAvailabilityZones:  &offeredZones,
		AcceptedAvailabilityZones: &fed.Spec.AcceptedAvailabilityZones,
	}, nil
}

func isValidFederationStatus(status string) bool {
	switch opgv1beta1.FederationState(status) {
	case opgv1beta1.FederationStateFailed, opgv1beta1.FederationStateTemporaryFailure, opgv1beta1.FederationStateAvailable, opgv1beta1.FederationStateLocked, opgv1beta1.FederationStateNotAvailable:
		return true
	}
	return false
}
