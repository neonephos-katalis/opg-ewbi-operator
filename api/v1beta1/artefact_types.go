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
	ArtefactFinalizer = "artefact.opg.ewbi.finalizer.nby.one"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ArtefactSpec defines the desired state of Artefact.
type ArtefactSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// AppProviderID identifies the provider of the application
	// e.g. "provider-id-67890"
	AppProviderId string `json:"appProviderId,omitempty"`

	// artefactId is stored as the .metadata.name

	// Name represents the artefacts's human-friendly identifier
	// e.g. "artemis"
	ArtefactName string `json:"artefactName,omitempty"`

	// ArtefactVersion
	// e.g. "14"
	ArtefactVersion string `json:"artefactVersion,omitempty"`

	DescriptorType string `json:"descriptorType,omitempty"`
	VirtType       string `json:"virtType,omitempty"`

	ComponentSpec []ComponentSpec `json:"componentSpec,omitempty"`
}

type ComponentSpec struct {
	// The component's name
	Name string `json:"name,omitempty"`

	CommandLineParams CommandLine `json:"commandLineParams,omitempty"`

	Images []string `json:"images,omitempty"`

	NumOfInstances int64 `json:"numOfInstances,omitempty"`

	RestartPolicy string `json:"restartPolicy,omitempty"`

	ComputeResourceProfile ComputeResourceProfile `json:"computeResourceProfile,omitempty"`

	ExposedInterfaces []ExposedInterface `json:"exposedInterfaces,omitempty"`
}

type CommandLine struct {
	Command []string `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

type ComputeResourceProfile struct {
	CPUArchType    string `json:"cpuArchType,omitempty"`
	CPUExclusivity bool   `json:"cpuExclusivity,omitempty"`
	Memory         int64  `json:"memory,omitempty"`
	NumCPU         string `json:"numCPU,omitempty"`
}

type ExposedInterface struct {
	Port           int64  `json:"port,omitempty"`
	Protocol       string `json:"protocol,omitempty"`
	InterfaceId    string `json:"interfaceId,omitempty"`
	VisibilityType string `json:"visibilityType,omitempty"`
}

type ArtefactPhase string

const (
	ArtefactPhaseReconciling ArtefactPhase = "Pending"
	ArtefactPhaseReady       ArtefactPhase = "Ready"
	ArtefactPhaseError       ArtefactPhase = "Error"
	ArtefactPhaseUnknown     ArtefactPhase = "Unknown"
)

// ArtefactStatus defines the observed state of Artefact.
type ArtefactStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	Phase ArtefactPhase `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Artefact is the Schema for the artefacts API.
type Artefact struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArtefactSpec   `json:"spec,omitempty"`
	Status ArtefactStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ArtefactList contains a list of Artefact.
type ArtefactList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Artefact `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Artefact{}, &ArtefactList{})
}
