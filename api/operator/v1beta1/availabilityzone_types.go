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
	AvailabilityZoneFinalizer = "katalis.com/availabilityzone.opg.ewbi.finalizer"
)

type ZoneState string

const (
	ZoneStateFailed           ZoneState = "FAILED"
	ZoneStateTemporaryFailure ZoneState = "TEMPORARY_FAILURE"
	ZoneStateAvailable        ZoneState = "AVAILABLE"
	ZoneStateLocked           ZoneState = "LOCKED"
	ZoneStateNotAvailable     ZoneState = "NOT_AVAILABLE"

	PluralAvailabilityZone = "availabilityzones"
	KindAvailabilityZone   = "AvailabilityZone"
)

type GPUResource struct {
	// GPU vendor name e.g. NVIDIA, AMD etc.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=GPU_PROVIDER_NVIDIA;GPU_PROVIDER_AMD
	// +kubebuilder:example=Nvidia
	GpuVendorType string `json:"gpuVendorType"`

	// Model name corresponding to vendorType may include info e.g. for NVIDIA, model name could be "Tesla M60", "Tesla V100" etc.
	// +kubebuilder:validation:Required
	GpuModeName string `json:"gpuModeName"`

	// GPU memory in Mbytes
	// +kubebuilder:validation:Required
	GpuMemory int `json:"gpuMemory"`

	// Number of GPUs
	// +kubebuilder:validation:Required
	NumGPU int `json:"numGPU"`
}

type Hugepage struct {
	// Size of hugepage
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum="2MB";"4MB";"1GB"
	PageSize string `json:"pageSize"`

	// Total number of huge pages
	// +kubebuilder:validation:Required
	Number int `json:"number"`
}

type ComputeResourceInfo struct {
	// CPU Instruction Set Architecture (ISA) E.g., Intel, Arm etc.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=ISA_X86_64;ISA_ARM_64
	CpuArchType string `json:"cpuArchType"`

	// Number of vcpus in whole, decimal up to millivcpu, or millivcpu format.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^\d+((\.\d{1,3})|(m))?$`
	// +kubebuilder:example={ "whole": { "value": "2" }, "decimal": { "value": "0.500" }, "millivcpu": { "value": "500m" } }
	NumCPU string `json:"numCPU"`

	// Amount of RAM in Mbytes
	// +kubebuilder:validation:Required
	Memory int64 `json:"memory"`

	// Amount of disk storage in Gbytes for a given ISA type
	// +kubebuilder:validation:Optional
	DiskStorage int32 `json:"diskStorage,omitempty"`

	// +kubebuilder:validation:Optional
	Gpu []GPUResource `json:"gpu,omitempty"`

	// Number of Intel VPUs available for a given ISA type
	// +kubebuilder:validation:Optional
	Vpu int `json:"vpu,omitempty"`

	// Number of FPGAs available for a given ISA type
	// +kubebuilder:validation:Optional
	Fpga int `json:"fpga,omitempty"`

	// +kubebuilder:validation:Optional
	Hugepages []Hugepage `json:"hugepages,omitempty"`

	// Support for exclusive CPUs
	// +kubebuilder:validation:Optional
	CpuExclusivity bool `json:"cpuExclusivity,omitempty"`
}

type SupportedOSType struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=x86_64;x86
	// +kubebuilder:example=x86_64
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

type FlavourSupported struct {
	// An identifier to refer to a specific combination of compute resources
	// +kubebuilder:validation:Required
	FlavourId string `json:"flavourId"`

	// CPU Instruction Set Architecture (ISA) E.g., Intel, Arm etc.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=ISA_X86;ISA_X86_64;ISA_ARM_64
	CpuArchType string `json:"cpuArchType"`

	// A list of operating systems which this flavour configuration can support e.g., RHEL Linux, Ubuntu 18.04 LTS, MS Windows 2012 R2.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	SupportedOSTypes []SupportedOSType `json:"supportedOSTypes"`

	// Number of available vCPUs
	// +kubebuilder:validation:Required
	NumCPU int32 `json:"numCPU"`

	// Amount of RAM in Mbytes
	// +kubebuilder:validation:Required
	MemorySize int32 `json:"memorySize"`

	// Amount of disk storage in Gbytes
	// +kubebuilder:validation:Required
	StorageSize int32 `json:"storageSize"`

	// +kubebuilder:validation:Optional
	Gpu []GPUResource `json:"gpu,omitempty"`

	// Number of FPGAs
	// +kubebuilder:validation:Optional
	Fpga int32 `json:"fpga,omitempty"`

	// Number of Intel VPUs available
	// +kubebuilder:validation:Optional
	Vpu int `json:"vpu,omitempty"`

	// +kubebuilder:validation:Optional
	Hugepages []Hugepage `json:"hugepages,omitempty"`

	// Support for exclusive CPUs
	// +kubebuilder:validation:Optional
	CpuExclusivity bool `json:"cpuExclusivity,omitempty"`
}

type NetworkResources struct {
	// Max dl throughput that this edge can offer. It is defined in Mbps.
	// +kubebuilder:validation:Required
	EgressBandWidth int32 `json:"egressBandWidth"`

	// Number of network interface cards which can be dedicatedly assigned to application pods on isolated networks. This includes virtual as well physical NICs
	// +kubebuilder:validation:Required
	DedicatedNIC int32 `json:"dedicatedNIC"`

	// If this zone support SRIOV networks or not
	// +kubebuilder:validation:Required
	SupportSriov bool `json:"supportSriov"`

	// If this zone supports DPDK based networking.
	// +kubebuilder:validation:Required
	SupportDPDK bool `json:"supportDPDK"`
}

type LatencyRanges struct {
	// The time for data/packet to reach from UC to edge application. It represent mínimum latency in milli seconds that may exist between UCs and edge apps in this zone but it can be higher in actual.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	MinLatency int32 `json:"minLatency,omitempty"`

	// The maximum limit of latency between UC and Edge App in milli seconds.
	// +kubebuilder:validation:Optional
	MaxLatency int32 `json:"maxLatency,omitempty"`
}

type JitterRanges struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	MinJitter int32 `json:"minJitter,omitempty"`

	// The maximum limit of network jitter between UC and Edge App in milli seconds.
	// +kubebuilder:validation:Optional
	MaxJitter int32 `json:"maxJitter,omitempty"`
}

type ThroughputRanges struct {
	// The minimum limit of network throughput between UC and Edge App in Mega bits per seconds (Mbps).
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	MinThroughput int32 `json:"minThroughput,omitempty"`

	// The maximum limit of network throughput between UC and Edge App in Mega bits per seconds (Mbps).
	// +kubebuilder:validation:Optional
	MaxThroughput int32 `json:"maxThroughput,omitempty"`
}

type ZoneServiceLevelObjsInfo struct {
	// +kubebuilder:validation:Required
	LatencyRanges *LatencyRanges `json:"latencyRanges"`

	// +kubebuilder:validation:Required
	JitterRanges *JitterRanges `json:"jitterRanges"`

	// +kubebuilder:validation:Required
	ThroughputRanges *ThroughputRanges `json:"throughputRanges"`
}

// AvailabilityZoneSpec defines the desired state of AvailabilityZone
type AvailabilityZoneSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=HOST;GUEST
	RelationType string `json:"relationType"`
	// This identifier shall be provided by the partner OP on successful verification and validation of the federation create request and is used by partner op to identify this newly created federation context. Originating OP shall provide this identifier in any subsequent request towards the partner op.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9-]*$`
	FederationContextId string `json:"federationContextId"`

	// Human readable name of the zone.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9-]*$`
	ZoneId string `json:"zoneId,omitempty"`

	// +kubebuilder:validation:Optional
	ZoneNotifLink string `json:"availZoneNotifLink,omitempty"`
}

// AvailabilityZoneStatus defines the observed state of AvailabilityZone.
type AvailabilityZoneStatus struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=FAILED;TEMPORARY_FAILURE;AVAILABLE;LOCKED;NOT_AVAILABLE
	State ZoneState `json:"state,omitempty"`
	// Resources exclusively reserved for the originator OP.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	ReservedComputeResources []ComputeResourceInfo `json:"reservedComputeResources,omitempty"`

	// Max quota on resources partner OP allows over reserved resources.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	ComputeResourceQuotaLimits []ComputeResourceInfo `json:"computeResourceQuotaLimits,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	FlavoursSupported []FlavourSupported `json:"flavoursSupported,omitempty"`

	// +kubebuilder:validation:Optional
	NetworkResources *NetworkResources `json:"networkResources,omitempty"`

	// It is a measure of the actual amount of data that is being sent over a network per unit of time and indicates máximum supported value for a zone
	// +kubebuilder:validation:Optional
	ZoneServiceLevelObjsInfo *ZoneServiceLevelObjsInfo `json:"zoneServiceLevelObjsInfo,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=avazone,scope=Namespaced
// +kubebuilder:printcolumn:name="relationType",type="string",JSONPath=".spec.relationType"
// +kubebuilder:printcolumn:name="federationContextId",type="string",JSONPath=".spec.federationContextId"
// +kubebuilder:printcolumn:name="zoneId",type="string",JSONPath=".spec.zoneId"
// +kubebuilder:printcolumn:name="state",type="string",JSONPath=".status.state"

// AvailabilityZone is the Schema for the availabilityzones API
type AvailabilityZone struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`
	// +required
	Spec AvailabilityZoneSpec `json:"spec"`
	// +optional
	Status AvailabilityZoneStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// AvailabilityZoneList contains a list of AvailabilityZone
type AvailabilityZoneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []AvailabilityZone `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AvailabilityZone{}, &AvailabilityZoneList{})
}
