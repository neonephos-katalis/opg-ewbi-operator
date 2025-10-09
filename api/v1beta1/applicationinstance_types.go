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
	ApplicationInstanceFinalizer = "applicationinstance.opg.ewbi.finalizer.nby.one"
)

type ApplicationInstancePhase string

const (
	ApplicationInstancePhaseReconciling ApplicationInstancePhase = "Reconciling"
	ApplicationInstancePhaseReady       ApplicationInstancePhase = "Ready"
	ApplicationInstancePhaseError       ApplicationInstancePhase = "Error"
	ApplicationInstancePhaseUnknown     ApplicationInstancePhase = "Unknown"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ApplicationInstanceSpec defines the desired state of ApplicationInstance.
type ApplicationInstanceSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// AppProviderID identifies the provider of the application
	// e.g. "provider-id-67890"
	AppProviderId string `json:"appProviderId,omitempty"`

	// appInstanceId is stored as the .metadata.name

	// AppId references the AppId resource this applicationInstance instantiates
	AppId string `json:"appId,omitempty"`

	// AppVersion references the AppId's version this applicationInstance instantiates
	AppVersion string `json:"appVersion,omitempty"`

	ZoneInfo Zone `json:"zoneInfo,omitempty"`

	// If defined, Link where the guest orchestrator expects to receive the callBacks with
	// ApplicationInstance status updates
	CallbBackLink string `json:"callBackLink,omitempty"`
}

type Zone struct {
	ZoneId string `json:"zoneId,omitempty"`

	FlavourId string `json:"flavourId,omitempty"`

	ResourceConsumption string `json:"resourceConsumption,omitempty"`

	ResPool string `json:"resPool,omitempty"`
}

type ApplicationInstanceState string

const (
	ApplicationInstanceStatePending     ApplicationInstanceState = "Pending"
	ApplicationInstanceStateReady       ApplicationInstanceState = "Ready"
	ApplicationInstanceStateFailed      ApplicationInstanceState = "Failed"
	ApplicationInstanceStateTerminating ApplicationInstanceState = "Terminating"
)

// ApplicationInstanceStatus defines the observed state of ApplicationInstance.
type ApplicationInstanceStatus struct {
	State      ApplicationInstanceState `json:"state,omitempty"`
	Conditions []metav1.Condition       `json:"conditions,omitempty"`
	Phase      ApplicationInstancePhase `json:"phase,omitempty"`
	ErrorMsg   string                   `json:"errorMsg,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ApplicationInstance is the Schema for the applicationinstances API.
type ApplicationInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationInstanceSpec   `json:"spec,omitempty"`
	Status ApplicationInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationInstanceList contains a list of ApplicationInstance.
type ApplicationInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApplicationInstance `json:"items"`
}

// ApplicationInstance Reasons and Conditions

const (
	// ReadyCondition indicates whether the ApplicationInstance is ready for use.
	ApplicationInstanceConditionReady = "Ready"

	// ReconcilingCondition indicates whether the ApplicationInstance is being reconciled.
	ApplicationInstanceConditionReconciling = "Reconciling"

	// StalledCondition indicates whether the ApplicationInstance reconciliation is blocked due to errors.
	ApplicationInstanceConditionStalled = "Stalled"
)

const (
	// Reasons for ConditionReady
	ApplicationInstanceReasonResourcesAllocated = "ResourcesAllocated" // if its not ready, then is synching

	// Reasons for ConditionReconciling
	ApplicationInstanceReasonSyncInProgress = "SyncInProgress" // In progress
	ApplicationInstanceReasonSyncCompleted  = "SyncCompleted"  // completed

	// Reasons for ConditionStalled
	ApplicationInstanceReasonInvalidOperation   = "InvalidOperation"     // Invalid Data provided, such as invalid type
	ApplicationInstanceReasonResourcesExhausted = "ResourcesUnavailable" // Requested resources not available.

)

func init() {
	SchemeBuilder.Register(&ApplicationInstance{}, &ApplicationInstanceList{})
}
