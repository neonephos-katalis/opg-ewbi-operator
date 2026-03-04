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
	AppFinalizer = "app.opg.ewbi.finalizer.nby.one"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ApplicationSpec defines the desired state of Application.
type ApplicationSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// AppProviderID identifies the provider of the application
	// e.g. "provider-id-67890"
	AppProviderId string `json:"appProviderId,omitempty"`

	// appId is stored as the .metadata.name

	ComponentSpecs []ComponentSpecRef `json:"componentSpecs,omitempty"`

	MetaData AppMetaData `json:"appMetaData,omitempty"`

	QoSProfile QoSProfile `json:"qoSProfile,omitempty"`

	StatusLink string `json:"statusLink,omitempty"`
}

type ComponentSpecRef struct {
	ArtefactId string `json:"artefactId"`
}

type AppMetaData struct {
	AccessToken     string `json:"accessToken,omitempty"`
	Name            string `json:"name,omitempty"`
	MobilitySupport bool   `json:"mobilitySupport,omitempty"`
	Version         string `json:"version,omitempty"`
}

type QoSProfile struct {
	Provisioning       bool   `json:"provisioning,omitempty"`
	LatencyConstraints string `json:"latencyConstraints,omitempty"`
	MultiUserClients   string `json:"multiUserClients,omitempty"`
	UsersPerAppInst    int64  `json:"usersPerAppInst,omitempty"`
}

type ApplicationState string

const (
	ApplicationStatePending    ApplicationState = "Pending"
	ApplicationStateOnboarded  ApplicationState = "Onboarded"
	ApplicationStateDeboarding ApplicationState = "Deboarding"
	ApplicationStateFailed     ApplicationState = "Failed"
	ApplicationStateRemoved    ApplicationState = "Removed"
)

type ApplicationPhase string

const (
	ApplicationPhaseReconciling ApplicationPhase = "Pending"
	ApplicationPhaseReady       ApplicationPhase = "Ready"
	ApplicationPhaseError       ApplicationPhase = "Error"
	ApplicationPhaseUnknown     ApplicationPhase = "Unknown"
)

// ApplicationStatus defines the observed state of Application.
type ApplicationStatus struct {
	Phase ApplicationPhase `json:"phase,omitempty"`
	State ApplicationState `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Application is the Schema for the applications API.
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationList contains a list of Application.
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
