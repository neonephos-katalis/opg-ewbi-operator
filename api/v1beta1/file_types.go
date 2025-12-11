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
	FileFinalizer = "file.opg.ewbi.finalizer.nby.one"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FileSpec defines the desired state of File.
type FileSpec struct {
	// Important: Run "make" to regenerate code after modifying this file

	// AppProviderID identifies the provider of the application
	// e.g. "provider-id-67890"
	AppProviderId string `json:"appProviderId,omitempty"`

	// fileId is stored as the .metadata.name

	// FileName represents the file's human-friendly identifier
	// e.g. "example-image.qcow2"
	FileName string `json:"fileName,omitempty"`

	// FileVersion
	// e.g. "1.0.0"
	FileVersion string `json:"fileVersion,omitempty"`

	// FileType defines the file's type
	// e.g. "QCOW2"
	FileType string `json:"fileType,omitempty"`

	Repo Repo `json:"repoLocation,omitempty"`

	Image Image `json:"image,omitempty"`
}

type Repo struct {

	// Repo's type e.g. public,private
	Type string `json:"type,omitempty"`

	// Repo's URL
	URL string `json:"url,omitempty"`

	// Password field required to access private repos
	Password string `json:"password,omitempty"`

	// Repo's Access token, the spec doesn't clarify
	// if it should either Password or Token
	Token string `json:"token,omitempty"`

	UserName string `json:"username,omitempty"`
}

type Image struct {
	// the File's Image instructionSet Architecture
	InstructionSetArchitecture string `json:"instructionSetArchitecture,omitempty"`

	// OS the expected Host Operative System the Image requires
	OS OS `json:"os,omitempty"`
}

type OS struct {
	// The OS's required architecture
	// e.g. "x86_64"
	Architecture string `json:"architecture,omitempty"`

	// The OS's distribution
	// e.g. "Ubuntu"
	Distribution string `json:"distribution,omitempty"`

	// The OS's license
	// e.g. "OS_LICENSE_TYPE_FREE"
	License string `json:"license,omitempty"`

	// The OS's Version
	// e.g. "OS_VERSION_UBUNTU_2204_LTS"
	Version string `json:"version,omitempty"`
}

type FilePhase string
type FileState string

const (
	FilePhaseReconciling FilePhase = "Pending"
	FilePhaseReady       FilePhase = "Ready"
	FilePhaseError       FilePhase = "Error"
	FilePhaseUnknown     FilePhase = "Unknown"
)

const (
	FileStatePending FilePhase = "Pending"
	FileStateReady   FilePhase = "Ready"
	FileStateError   FilePhase = "Error"
	FileStateUnknown FilePhase = "Unknown"
)

// FileStatus defines the observed state of File.
type FileStatus struct {
	Phase FilePhase `json:"phase,omitempty"`
	State FileState `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// File is the Schema for the files API.
type File struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FileSpec   `json:"spec,omitempty"`
	Status FileStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FileList contains a list of File.
type FileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []File `json:"items"`
}

func init() {
	SchemeBuilder.Register(&File{}, &FileList{})
}
