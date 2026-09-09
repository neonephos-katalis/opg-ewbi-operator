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
	ArtefactFinalizer = "katalis.com/artefact.opg.ewbi.finalizer"
)

type ArtefactState string
type ArtefactK8sErrMsg string
type ArtefactK8sUnErrMsg string

const (
	ArtefactStatePending ArtefactState = "PENDING"
	ArtefactStateReady   ArtefactState = "READY"
	ArtefactStateError   ArtefactState = "ERROR"
	ArtefactStateUnknown ArtefactState = "UNKNOWN"

	PluralArtefact = "artefacts"
	KindArtefact   = "Artefact"
)

// / +kubebuilder:validation:Format=uuid
// A globally unique identifier associated with the image file. Originating OP generates this identifier when file is uploaded over NBI.
type ImageUUID string

type CommandString string

type ArtefactRepoLocation struct {
	// +kubebuilder:validation:Optional
	RepoURL string `json:"repoURL,omitempty"`

	// +kubebuilder:validation:Optional
	// Username to access the repository
	UserName string `json:"userName,omitempty"`

	// +kubebuilder:validation:Optional
	// Password to access the repository
	Password string `json:"password,omitempty"`

	// Authorization token to access the repository
	// +kubebuilder:validation:Optional
	Token string `json:"token,omitempty"`
}

// List of commands and arguments that shall be invoked when the component instance is created. This is valid only for container based deployment.
type CommandLineParams struct {
	// +kubebuilder:validation:Required
	// List of commands that application should invoke when an instance is created.
	Command []string `json:"command"`

	// +kubebuilder:validation:Optional
	// List of arguments required by the command.
	CommandArgs []string `json:"commandArgs,omitempty"`
}

type ExposedInterface struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9_]{6,30}[A-Za-z0-9]$`
	// Each Port and corresponding traffic protocol exposed by the component is identified by a name. Application client on user device requires this to uniquely identify the interface.
	InterfaceId string `json:"interfaceId"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=TCP;UDP;HTTP_HTTPS
	// Defines the IP transport communication protocol i.e., TCP, UDP or HTTP
	CommProtocol string `json:"commProtocol"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// Port number exposed by the component. OP may generate a dynamic port towards the UCs corresponding to this internal port and forward the client traffic from dynamic port to container Port.
	CommPort int32 `json:"commPort"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=VISIBILITY_EXTERNAL;VISIBILITY_INTERNAL
	// Defines whether the interface is exposed to outer world or not i.e., external, or internal. If this is set to "external", then it is exposed to external applications otherwise it is exposed internally to edge application components within edge cloud. When exposed to external world, an external dynamic port is assigned for UC traffic and mapped to the internal container Port
	VisibilityType string `json:"visibilityType"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^[A-Za-z][A-Za-z0-9_]{6,30}[A-Za-z0-9]$`
	// Name of the network.  In case the application has to be associated with more than 1 network then app provider must define the name of the network on which this interface has to be exposed.  This parameter is required only if the port has to be exposed on a specific network other than default.
	Network string `json:"network,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9]{3}$`
	// Interface Name. Required only if application has to be attached to a network other than default.
	InterfaceName string `json:"InterfaceName,omitempty"`
}

// Environment variables are key value pairs that should be injected when component in instantiated
type CompEnvParam struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9_]{6,30}[A-Za-z0-9]$`
	// Name of environment variable
	EnvVarName string `json:"envVarName"` // Nota: Nello YAML era envVarNam in required e envVarName in properties. Uso la property.

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=USER_DEFINED;PLATFORM_DEFINED_DYNAMIC_PORT;PLATFORM_DEFINED_DNS;PLATFORM_DEFINED_IP
	EnvValueType string `json:"envValueType"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9_]{6,62}[A-Za-z0-9]$`
	// Value to be assigned to environment variable
	EnvVarValue string `json:"envVarValue,omitempty"`

	// +kubebuilder:validation:Optional
	// Full path of parameter from componentSpec that should be used to generate the environment value. Eg. networkResourceProfile. interfaceId.
	EnvVarSrc string `json:"envVarSrc,omitempty"`
}

// Configuration used when deploying a component. May override other ComponentSpec parameters related to deployment like restart policy, command line parameters, environment variables, etc.
type DeploymentConfig struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=DOCKER_COMPOSE;KUBERNETES_MANIFEST;CLOUD_INIT;HELM_VALUES
	// Config type.
	ConfigType string `json:"configType"`

	// +kubebuilder:validation:Required
	// Contents of the configuration.
	Contents string `json:"contents"`
}

type PersistentVolume struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum="10Gi";"20Gi";"50Gi";"100Gi"
	// Size of the volume given by user (10GB, 20GB, 50 GB or 100GB)
	VolumeSize string `json:"volumeSize"`

	// +kubebuilder:validation:Required
	// Defines the mount path of the volume
	VolumeMountPath string `json:"volumeMountPath"`

	// +kubebuilder:validation:Required
	// Human readable name for the volume
	VolumeName string `json:"volumeName"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// It indicates the ephemeral storage on the node and contents are not preserved if containers restarts
	EphemeralType bool `json:"ephemeralType,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=RW;RO
	// +kubebuilder:default=RW
	// Values are RW (read/write) and RO (read-only)
	AccessMode string `json:"accessMode,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=EXCLUSIVE;SHARED
	// +kubebuilder:default=EXCLUSIVE
	// Exclusive or Shared. If shared, then in case of multiple containers same volume will be shared across the containers.
	SharingPolicy string `json:"sharingPolicy,omitempty"`
}
type ComputeResourceProfile struct {
	// +kubebuilder:validation:Optional
	CPUArchType string `json:"cpuArchType"`
	// +kubebuilder:validation:Optional
	CPUExclusivity bool `json:"cpuExclusivity"`
	// +kubebuilder:validation:Optional
	Memory int64 `json:"memory"`
	// +kubebuilder:validation:Optional
	NumCPU string `json:"numCPU"`
}
type ExposedInterfaceInfo struct {
	// +kubebuilder:validation:Optional
	Port int32 `json:"port"`
	// +kubebuilder:validation:Optional
	Protocol string `json:"protocol"`
	// +kubebuilder:validation:Optional
	InterfaceId string `json:"interfaceId"`
	// +kubebuilder:validation:Optional
	VisibilityType string `json:"visibilityType"`
}
type ComponentSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9_]{6,62}[A-Za-z0-9]$`
	// Must be a valid RFC 1035 label name.  Component name must be unique with an application
	ComponentName string `json:"componentName"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// List of all images associated with the component. Images are specified using the file identifiers. Partner OP provides these images using file upload api.
	Images []string `json:"images"`

	// +kubebuilder:validation:Required
	// Number of component instances to be launched.
	NumOfInstances int32 `json:"numOfInstances"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=RESTART_POLICY_ALWAYS;RESTART_POLICY_NEVER
	// How the platform shall handle component failure
	RestartPolicy string `json:"restartPolicy"`

	// +kubebuilder:validation:Required
	ComputeResourceProfile *ComputeResourceProfile `json:"computeResourceProfile"`

	// +kubebuilder:validation:Optional
	CommandLineParams *CommandLineParams `json:"commandLineParams,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	// Each application component exposes some ports either for external users or for inter component communication. Application provider is required to specify which ports are to be exposed and the type of traffic that will flow through these ports.
	ExposedInterfaces []ExposedInterfaceInfo `json:"exposedInterfaces,omitempty"`

	// +kubebuilder:validation:Optional
	CompEnvParams []CompEnvParam `json:"compEnvParams,omitempty"`

	// +kubebuilder:validation:Optional
	DeploymentConfig *DeploymentConfig `json:"deploymentConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	// The ephemeral volume a container process may need to temporary store internal data
	PersistentVolumes []PersistentVolume `json:"persistentVolumes,omitempty"`
}

type ArtefactBody struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z][A-Za-z0-9_]{7,63}$`
	// UserId of the app provider.  Identifier is relevant only in context of this federation.
	AppProviderId string `json:"appProviderId"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z][A-Za-z0-9_]{7,31}$`
	// Name of the artefact.
	ArtefactName string `json:"artefactName"`

	// +kubebuilder:validation:Required
	// Artefact version information
	ArtefactVersionInfo string `json:"artefactVersionInfo"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MaxLength=256
	// Brief description of the artefact by the application provider
	ArtefactDescription string `json:"artefactDescription,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=VM_TYPE;CONTAINER_TYPE
	ArtefactVirtType string `json:"artefactVirtType"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=8
	// +kubebuilder:validation:MaxLength=32
	// Name of the file.
	ArtefactFileName string `json:"artefactFileName,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=ZIP;TAR;TEXT;TARGZ
	// Artefacts like Helm charts or Terraform scripts may need compressed format.
	ArtefactFileFormat string `json:"artefactFileFormat,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=HELM;TERRAFORM;ANSIBLE;SHELL;COMPONENTSPEC
	// Type of descriptor present in the artefact.  App provider can either define either a Helm chart or a Terraform script or container spec.
	ArtefactDescriptorType string `json:"artefactDescriptorType"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=PRIVATEREPO;PUBLICREPO;UPLOAD
	// Artefact or file repository location. PUBLICREPO is used of public URLs like GitHub, Helm repo, docker registry etc., PRIVATEREPO is used for private repo managed by the application developer, UPLOAD is for the case when artefact/file is uploaded from MEC web portal.  OP should pull the image from ‘repoUrl' immediately after receiving the request and then send back the response. In case the repoURL corresponds to a docker registry, use docker v2 http api to do the pull.
	RepoType string `json:"repoType,omitempty"`

	// +kubebuilder:validation:Optional
	ArtefactRepoLocation *ArtefactRepoLocation `json:"artefactRepoLocation,omitempty"`

	// +kubebuilder:validation:Required
	// Details about compute, networking and storage requirements for each component of the application. App provider should define all information needed to instantiate the component. If artefact is being defined at component level this section should have information just about the component. In case the artefact is being defined at application level the section should provide details about all the components.
	ComponentSpec []ComponentSpec `json:"componentSpec"`
}

// ArtefactSpec defines the desired state of Artefact
type ArtefactSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=HOST;GUEST
	RelationType string `json:"relationType"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9-]*$`
	// This identifier shall be provided by the partner OP on successful verification and validation of the federation create request and is used by partner op to identify this newly created federation context. Originating OP shall provide this identifier in any subsequent request towards the partner op.
	FederationContextId string `json:"federationContextId"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=uuid
	// A globally unique identifier associated with the artefact. Originating OP generates this identifier when artefact is submitted over NBI.
	ArtefactId string `json:"artefactId"`

	// +kubebuilder:validation:Optional
	ArtefactBody *ArtefactBody `json:"artefactBody,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=uri
	// Callback URL to which the partner OP will send the artefact status update. This is provided by the partner OP when the artefact is created and is used by originating OP to send the artefact status update.
	ArtefactNotifLink string `json:"artefactNotifLink,omitempty"`
}

// ArtefactStatus defines the observed state of Artefact.
type ArtefactStatus struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=PENDING;READY;ERROR;UNKNOWN
	// Current state of the artefact upload
	State ArtefactState `json:"state,omitempty"`
	// +kubebuilder:validation:Optional
	// Message indicating details about the current state
	Message string `json:"message,omitempty"`
	// Timestamp of the last status update
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=art,scope=Namespaced
// +kubebuilder:printcolumn:name="relationType",type="string",JSONPath=".spec.relationType"
// +kubebuilder:printcolumn:name="federationContextId",type="string",JSONPath=".spec.federationContextId"
// +kubebuilder:printcolumn:name="artefactId",type="string",JSONPath=".spec.artefactId"
// +kubebuilder:printcolumn:name="state",type="string",JSONPath=".status.state"

// Artefact is the Schema for the artefacts API
type Artefact struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`
	// +required
	Spec ArtefactSpec `json:"spec"`

	// +optional
	Status ArtefactStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ArtefactList contains a list of Artefact
type ArtefactList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Artefact `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Artefact{}, &ArtefactList{})
}
