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
	ImageFinalizer = "katalis.com/image.opg.ewbi.finalizer"
)

type ImageState string

const (
	ImageStatePending ImageState = "PENDING"
	ImageStateReady   ImageState = "READY"
	ImageStateError   ImageState = "ERROR"
	ImageStateUnknown ImageState = "UNKNOWN"

	PluralImage = "images"
	KindImage   = "Image"
)

type ImgOSType struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=x86_64;x86
	Architecture string `json:"architecture"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=RHEL;UBUNTU;COREOS;FEDORA;WINDOWS;OTHER
	Distribution string `json:"distribution"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=OS_VERSION_UBUNTU_2204_LTS;OS_VERSION_RHEL_8;OS_VERSION_RHEL_7;OS_VERSION_DEBIAN_11;OS_VERSION_COREOS_STABLE;OS_MS_WINDOWS_2012_R2;OTHER
	Version string `json:"version"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=OS_LICENSE_TYPE_FREE;OS_LICENSE_TYPE_ON_DEMAND;NOT_SPECIFIED
	License string `json:"license"`
}

type ImageRepoLocation struct {
	// +kubebuilder:validation:Optional
	RepoURL string `json:"repoURL,omitempty"`

	// Username to access the repository
	// +kubebuilder:validation:Optional
	UserName string `json:"userName,omitempty"`

	// Password to access the repository
	// +kubebuilder:validation:Optional
	Password string `json:"password,omitempty"`

	// Authorization token to access the repository
	// +kubebuilder:validation:Optional
	Token string `json:"token,omitempty"`
}

type ImageBody struct {
	// UserId of the app provider. Identifier is relevant only in context of this federation.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z][A-Za-z0-9_]{7,63}$`
	AppProviderId string `json:"appProviderId"`

	// Name of the image.   App provides specifies this name when image is uploaded on originating OP over NBI.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z][A-Za-z0-9_]{7,31}$`
	ImageName string `json:"imageName"`

	// Brief description about the image.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=8
	// +kubebuilder:validation:MaxLength=128
	ImageDescription string `json:"imageDescription,omitempty"`

	// Image version information.
	// +kubebuilder:validation:Required
	ImageVersionInfo string `json:"imageVersionInfo"`

	// Indicate if the image is Container image or VM image (QCOW2, OVA)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=QCOW2;DOCKER;OVA
	ImageType string `json:"imageType"`

	// MD5 checksum for VM and image-based images, sha256 digest for containers
	// +kubebuilder:validation:Optional
	Checksum string `json:"checksum,omitempty"`

	// +kubebuilder:validation:Required
	ImgOSType *ImgOSType `json:"imgOSType"`

	// CPU Instruction Set Architecture (ISA) E.g., Intel, Arm etc.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=ISA_X86_64;ISA_ARM_64
	ImgInsSetArch string `json:"imgInsSetArch"`

	// Image repository location. PUBLICREPO is used of public URLs like GitHub, Helm repo, docker registry etc., PRIVATEREPO is used for private repo managed by the application developer, UPLOAD is for the case when image is uploaded from MEC web portal.  OP should pull the image from ‘repoUrl' immediately after receiving the request and then send back the response. In case the repoURL corresponds to a docker registry, use docker v2 http api to do the pull.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=PRIVATEREPO;PUBLICREPO;UPLOAD
	RepoType string `json:"repoType,omitempty"`

	// +kubebuilder:validation:Optional
	ImageRepoLocation *ImageRepoLocation `json:"imageRepoLocation,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=byte
	Image []byte `json:"image,omitempty"`
}

// ImageSpec defines the desired state of Image
type ImageSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=HOST;GUEST
	RelationType string `json:"relationType"`

	// This identifier shall be provided by the partner OP on successful verification and validation of the federation create request and is used by partner op to identify this newly created federation context. Originating OP shall provide this identifier in any subsequent request towards the partner op.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9-]*$`
	FederationContextId string `json:"federationContextId"`

	// A globally unique identifier associated with the image. Originating OP generates this identifier when image is uploaded over NBI.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uuid
	ImageId string `json:"imageId"`

	// +kubebuilder:validation:Optional
	ImageBody *ImageBody `json:"imageBody,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=uri
	// Callback URL to which the partner OP will send the image status update. This is provided by the partner OP when the image is created and is used by originating OP to send the image status update.
	ImageNotifLink string `json:"imageNotifLink,omitempty"`
}

// ImageStatus defines the observed state of Image.
type ImageStatus struct {
	// Current state of the artefact upload
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=PENDING;READY;ERROR;UNKNOWN
	State ImageState `json:"state,omitempty"`

	// Message indicating details about the current state
	// +kubebuilder:validation:Optional
	Message string `json:"message,omitempty"`

	// Timestamp of the last status update
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=date-time
	LastUpdated string `json:"lastUpdated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=imgman,scope=Namespaced
// +kubebuilder:printcolumn:name="relationType",type="string",JSONPath=".spec.relationType"
// +kubebuilder:printcolumn:name="federationContextId",type="string",JSONPath=".spec.federationContextId"
// +kubebuilder:printcolumn:name="imageId",type="string",JSONPath=".spec.imageId"
// +kubebuilder:printcolumn:name="state",type="string",JSONPath=".status.state"

// Image is the Schema for the images API
type Image struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec ImageSpec `json:"spec"`

	// +optional
	Status ImageStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// ImageList contains a list of Image
type ImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Image `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Image{}, &ImageList{})
}
