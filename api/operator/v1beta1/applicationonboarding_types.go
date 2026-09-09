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
	ApplicationOnboardingFinalizer = "katalis.com/applicationonboarding.opg.ewbi.finalizer"
)

type ApplicationOnboardingState string
type ApplicationOnboardingK8sErrMsg string
type ApplicationOnboardingK8sUnErrMsg string

const (
	ApplicationOnboardingStatePending    ApplicationOnboardingState = "PENDING"
	ApplicationOnboardingStateOnboarded  ApplicationOnboardingState = "ONBOARDED"
	ApplicationOnboardingStateDeboarding ApplicationOnboardingState = "DEBOARDING"
	ApplicationOnboardingStateFailed     ApplicationOnboardingState = "FAILED"
	ApplicationOnboardingStateRemoved    ApplicationOnboardingState = "REMOVED"

	PluralApplicationOnboarding = "applicationonboardings"
	KindApplicationOnboarding   = "ApplicationOnboarding"
)

// const (
// 	ErrorUpdatingApplicationStatusMsg ApplicationOnboardingK8sErrMsg   = ">>> [Application][K8s] Error Updating resource status"
// 	UnexpectedStatusApplicationMsg    ApplicationOnboardingK8sUnErrMsg = ">>> [Application][K8s] Unexpected Status Code"
// )

// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9-]*$`
// Human readable name of the zone.
type ZoneIdString string

type ZoneSetting struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	ZoneId []ZoneIdString `json:"zoneId,omitempty"`

	// +kubebuilder:validation:Optional
	// Value 'true' will forbid application instantiation on this zone.  No new instance of the application can be created on this zone.
	Forbid bool `json:"forbid,omitempty"`
}

type AppDeploymentZone struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9-]*$`
	// Human readable name of the zone.
	ZoneId string `json:"zoneId,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^[A-Z]{2}$`
	// ISO 3166-1 Alpha-2 code for the country of Partner operator
	CountryCode string `json:"countryCode,omitempty"`
}

type AppMetaData struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z][A-Za-z0-9_]{7,31}$`
	// Name of the application. Application provider define a human readable name for the application
	AppName string `json:"appName"`

	// +kubebuilder:validation:Required
	// Version info of the application
	Version string `json:"version"`

	// +kubebuilder:validation:Required
	// An application Access key, to be used with UNI interface to authorize UCs Access to a given application
	AccessToken string `json:"accessToken"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=16
	// +kubebuilder:validation:MaxLength=256
	// Brief application description provided by application provider
	AppDescription string `json:"appDescription,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Indicates if an application is sensitive to user mobility and can be relocated. Default is "FALSE"
	MobilitySupport bool `json:"mobilitySupport,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=IOT;HEALTH_CARE;GAMING;VIRTUAL_REALITY;SOCIALIZING;SURVEILLANCE;ENTERTAINMENT;CONNECTIVITY;PRODUCTIVITY;SECURITY;INDUSTRIAL;EDUCATION;OTHERS
	// Possible categorization of the application
	Category string `json:"category,omitempty"`
}

type AppQoSProfile struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=NONE;LOW;ULTRALOW
	// Latency requirements for the application.Allowed values (non-standardized) are none, low and ultra-low. Ultra-Low may corresponds to range 15 - 30 msec, Low correspond to range 30 - 50 msec. None means 51 and above
	LatencyConstraints string `json:"latencyConstraints"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// Data transfer bandwidth requirement (minimum limit) for the application. It should in Mbits/sec
	BandwidthRequired int32 `json:"bandwidthRequired,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=APP_TYPE_SINGLE_USER;APP_TYPE_MULTI_USER
	// Single user type application are designed to serve just one client. Multi user type application is designed to serve multiple clients
	MultiUserClients string `json:"multiUserClients,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// Maximum no of clients that can connect to an instance of this application. This parameter is relevant only for application of type multi user
	NoOfUsersPerAppInst int `json:"noOfUsersPerAppInst,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// Define if application can be instantiated or not
	AppProvisioning bool `json:"appProvisioning,omitempty"`
}

type AppComponentSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9])\.)*([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9])$`
	// Must be a valid RFC 1035 label name. This defines the DNS name via which the component can be accessed over NBI. Access via     serviceNameNB is restricted on specific ports. Platform shall expose component access externally via this DNS name
	ServiceNameNB string `json:"serviceNameNB,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9])\.)*([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9])$`
	// Must be a valid RFC 1035 label name. This defines the DNS name via which the component can be accessed via peer components. Access via serviceNameEW is open on all ports.   Platform shall not expose serviceNameEW externally outside edge.
	ServiceNameEW string `json:"serviceNameEW,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9_]{6,62}[A-Za-z0-9]$`
	// Must be a valid RFC 1035 label name.  Component name must be unique with an application
	ComponentName string `json:"componentName,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=uuid
	// A globally unique identifier associated with the artefact. Originating OP generates this identifier when artefact is submitted over NBI.
	ArtefactId string `json:"artefactId,omitempty"`
}

// Details about application compute resource requirements, associated artefacts, QoS profile and regions where application shall be made available etc.
type AppInfo struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z][A-Za-z0-9_]{7,63}$`
	// Identifier used to refer to an application.
	AppId string `json:"appId"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^[A-Za-z][A-Za-z0-9_]{7,63}$`
	// UserId of the app provider.  Identifier is relevant only in context of this federation.
	AppProviderId string `json:"appProviderId,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	// Details about partner OP zones where the application should be made available;  This field when specified will instruct the OP to restrict application instantiation only on the listed zones.
	AppDeploymentZones []AppDeploymentZone `json:"appDeploymentZones,omitempty"`

	// +kubebuilder:validation:Optional
	// Application metadata details
	AppMetaData *AppMetaData `json:"appMetaData,omitempty"`

	// +kubebuilder:validation:Optional
	// Parameters corresponding to the performance constraints, tenancy details etc.
	AppQoSProfile *AppQoSProfile `json:"appQoSProfile,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	AppComponentSpecs []AppComponentSpec `json:"appComponentSpecs,omitempty"`

	// +kubebuilder:validation:Optional
	AppStatusCallbackLink string `json:"appStatusCallbackLink,omitempty"`

	// +kubebuilder:validation:Optional
	// DNS FQDN assigned to application instances in an availability zone. User Clients can resolve the FQDN to communicate with the edge instances of the application
	EdgeAppFQDN string `json:"edgeAppFQDN,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=PENDING;ONBOARDED;DEBOARDING;REMOVED;FAILED
	// Defines change in application status. This change could be related to application itself or an application instance status
	OnboardStatusInfo string `json:"onboardStatusInfo,omitempty"`
}

// ApplicationOnboardingSpec defines the desired state of ApplicationOnboarding
type ApplicationOnboardingSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=HOST;GUEST
	RelationType string `json:"relationType"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9-]*$`
	// This identifier shall be provided by the partner OP on successful verification and validation of the federation create request and is used by partner op to identify this newly created federation context. Originating OP shall provide this identifier in any subsequent request towards the partner op.
	FederationContextId string `json:"federationContextId"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	ZoneSettings []ZoneSetting `json:"zoneSettings,omitempty"`

	// +kubebuilder:validation:Required
	AppInfo *AppInfo `json:"appInfo"`
}

// ApplicationOnboardingStatus defines the observed state of ApplicationOnboarding.
type ApplicationOnboardingStatus struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=PENDING;ONBOARDED;REMOVED;FAILED
	// Current state of the artefact upload
	State ApplicationOnboardingState `json:"state,omitempty"`

	// +kubebuilder:validation:Optional
	// Message indicating details about the current state
	Message string `json:"message,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=date-time
	// Timestamp of the last status update
	LastUpdated string `json:"lastUpdated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=apponb,scope=Namespaced
// +kubebuilder:printcolumn:name="relationType",type="string",JSONPath=".spec.relationType"
// +kubebuilder:printcolumn:name="federationContextId",type="string",JSONPath=".spec.federationContextId"
// +kubebuilder:printcolumn:name="appId",type="string",JSONPath=".spec.appInfo.appId"
// +kubebuilder:printcolumn:name="appProvider",type="string",JSONPath=".spec.appInfo.appProviderId"
// +kubebuilder:printcolumn:name="state",type="string",JSONPath=".status.state"

// ApplicationOnboarding is the Schema for the applicationonboardings API
type ApplicationOnboarding struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`
	// +required
	Spec ApplicationOnboardingSpec `json:"spec"`
	// +optional
	Status ApplicationOnboardingStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
// ApplicationOnboardingList contains a list of ApplicationOnboarding
type ApplicationOnboardingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ApplicationOnboarding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApplicationOnboarding{}, &ApplicationOnboardingList{})
}
