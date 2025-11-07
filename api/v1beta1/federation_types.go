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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// finalizers
const (
	FederationFinalizer = "federation.opg.ewbi.finalizer.nby.one"
)

// labels
const (
	FederationContextIdLabel = "opg.ewbi.nby.one/federation-context-id"
	FederationRelationLabel  = "opg.ewbi.nby.one/federation-relation"
	FederationGuestUrlLabel  = "opg.ewbi.nby.one/federation-guest-url"
	ExternalIdLabel          = "opg.ewbi.nby.one/id"
)

// fields
const (
	FederationStatusContextIDField = ".status.federationContextId"
)

type FederationRelation string

const (
	FederationRelationGuest FederationRelation = "guest"
	FederationRelationHost  FederationRelation = "host"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FederationSpec defines the desired state of Federation.
type FederationSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Initial Date and time, time zone info of the federation initiated by the Originating OP
	// e.g. "2025-01-10T09:50:32.571Z"
	InitialDate metav1.Time `json:"initialDate,omitempty"`

	OriginOP Origin `json:"originOP,omitempty"`

	Partner Partner `json:"partner,omitempty"`

	// OfferedAvailabilityZones, list of AvailabilityZones the hostOP offers to the guestOP
	// as part of this Federation
	OfferedAvailabilityZones []ZoneDetails `json:"offeredAvailabilityZones,omitempty"`

	// AcceptedAvailabilityZones, subset the GuestOP accepts of the  AvailabilityZones
	// the OP offered for this Federation
	AcceptedAvailabilityZones []string `json:"acceptedAvailabilityZones,omitempty"`

	// Federation GuestPartner creds for the client to register (temporary, to be replaced by e.g. keycloack)
	GuestPartnerCredentials FederationCredentials `json:"guestPartnerCredentials,omitempty"`
}

type Origin struct {
	// CountryCode as in "JN"
	CountryCode string `json:"countryCode,omitempty"`

	// FixedNetworkCodes, no examples were provided, for now we just know they will be a list of strings
	FixedNetworkCodes []string `json:"fixedNetworkCodes,omitempty"`

	// MobileNetworkCodes, notice these aren't a list, but its internal field `mnc` is
	MobileNetworkCodes MobileNetworkCodes `json:"mobileNetworkCodes,omitempty"`
}

type MobileNetworkCodes struct {
	// MCC, e.g. "329"
	MCC string `json:"mcc,omitempty"`
	// MNC, list of MNCs
	MNC []string `json:"mncs,omitempty"`
}

type Partner struct {
	// CallbackCredentials Authentication credentials for callbacks.
	// Callbacks use the same security scheme, flows, and scopes as the forward path.
	CallbackCredentials FederationCredentials `json:"callbackCredentials,omitempty"`
	StatusLink          string                `json:"statusLink"`
}

type FederationCredentials struct {
	// ClientId, e.g. "50290107-5a90-4d77-a1b8-ec9d311ab2fd"
	ClientId string `json:"clientId"`

	// TokenUrl, e.g. http://ip:5555/cb
	TokenUrl string `json:"tokenUrl,omitempty"`
}

type FederationState string

const (
	FederationStateFailed           FederationState = "Failed"
	FederationStateTemporaryFailure FederationState = "TemporaryFailure"
	FederationStateAvailable        FederationState = "Available"
	FederationStateLocked           FederationState = "Locked"
	FederationStateNotAvailable     FederationState = "NotAvailable"
)

type FederationPhase string

const (
	FederationPhaseReconciling FederationPhase = "Pending"
	FederationPhaseReady       FederationPhase = "Ready"
	FederationPhaseError       FederationPhase = "Error"
	FederationPhaseUnknown     FederationPhase = "Unknown"
)

// FederationStatus defines the observed state of Federation.
type FederationStatus struct {
	FederationContextId string `json:"federationContextId,omitempty"`

	State FederationState `json:"state,omitempty"`

	Phase FederationPhase `json:"phase,omitempty"`

	// OfferedAvailabilityZones, GuestOP offered AvailabilityZones
	// for this Federation
	OfferedAvailabilityZones []ZoneDetails `json:"offeredAvailabilityZones,omitempty"`
}

type ZoneDetails struct {
	// GeographyDetails Details about cities or state covered by the edge. Details about the type
	// of locality for eg rural, urban, industrial etc. This information is defined in human readable form.
	GeographyDetails string `json:"geographyDetails"`

	// Geolocation Latitude,Longitude as decimal fraction up to 4 digit precision
	Geolocation string `json:"geolocation"`

	// ZoneId Human readable name of the zone.
	ZoneId string `json:"zoneId"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Federation is the Schema for the federations API.
type Federation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederationSpec   `json:"spec,omitempty"`
	Status FederationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FederationList contains a list of Federation.
type FederationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Federation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Federation{}, &FederationList{})
}
