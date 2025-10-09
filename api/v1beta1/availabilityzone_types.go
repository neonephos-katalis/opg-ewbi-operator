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

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AvailabilityZoneSpec defines the desired state of AvailabilityZone.
type AvailabilityZoneSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// GeographyDetails Details about cities or state covered by the edge.
	// Details about the type of locality for eg rural, urban, industrial etc.
	// This information is defined in human readable form.
	GeographyDetails string `json:"geographyDetails,omitempty"`

	// Geolocation Latitude,Longitude as decimal fraction up to 4 digit precision
	Geolocation GeoLocation `json:"geolocation,omitempty"`

	// ZoneId Human readable name of the zone.
	ZoneId ZoneIdentifier `json:"zoneId,omitempty"`
}

// GeoLocation Latitude,Longitude as decimal fraction up to 4 digit precision
type GeoLocation string

// ZoneIdentifier Human readable name of the zone.
type ZoneIdentifier string

type AvailabilityZonePhase string

const (
	AvailabilityZonePhaseReconciling AvailabilityZonePhase = "Pending"
	AvailabilityZonePhaseReady       AvailabilityZonePhase = "Ready"
	AvailabilityZonePhaseError       AvailabilityZonePhase = "Error"
	AvailabilityZonePhaseUnknown     AvailabilityZonePhase = "Unknown"
)

// AvailabilityZoneStatus defines the observed state of AvailabilityZone.
type AvailabilityZoneStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	Phase AvailabilityZonePhase `json:"phase,omitempty"`

	// To be considered, for now these are just placeholder samples
	FlavoursSupported []string `json:"flavoursSupported,omitempty"`

	// To be considered, for now these are just placeholder samples
	ReservedComputeResources string `json:"reservedComputeResources,omitempty"`

	// To be considered, for now these are just placeholder samples
	ComputeResourceQuotaLimits string `json:"computeResourceQuotaLimits,omitempty"`

	// To be considered, for now these are just placeholder samples
	Latency string `json:"latency,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=az

// AvailabilityZone is the Schema for the availabilityzones API.
type AvailabilityZone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AvailabilityZoneSpec   `json:"spec,omitempty"`
	Status AvailabilityZoneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AvailabilityZoneList contains a list of AvailabilityZone.
type AvailabilityZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AvailabilityZone `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AvailabilityZone{}, &AvailabilityZoneList{})
}
