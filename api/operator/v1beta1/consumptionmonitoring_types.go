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

type ConsumptionMonitoringState string

const (
	ConsumptionMonitoringFinalizer = "katalis.com/consumptionmonitoring.opg.ewbi.finalizer"

	ConsumptionMonitoringStatePending      ConsumptionMonitoringState = "PENDING"
	ConsumptionMonitoringStateAvailable    ConsumptionMonitoringState = "AVAILABLE"
	ConsumptionMonitoringStateNotAvailable ConsumptionMonitoringState = "NOT_AVAILABLE"
	ConsumptionMonitoringStateFailed       ConsumptionMonitoringState = "FAILED"

	PluralConsumptionMonitoring = "consumptionmonitorings"
	KindConsumptionMonitoring   = "ConsumptionMonitoring"
)

type PeriodicityInterval struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=23
	NumHours int32 `json:"numHours"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=59
	NumMins int32 `json:"numMins"`
}

type UtilizationValue struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=CPU;MEMORY;DISK;Network;FLAVOUR
	ResType string `json:"resType"`

	// +kubebuilder:validation:Required
	Value string `json:"value"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Percent;MBPS;GB;TB;CORES;SECONDS;MINUTES
	Unit string `json:"unit"`
}

type CpuUtilization struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=edge_resource;app_resource;alarm;all
	CpuType string `json:"cpuType"`

	// +kubebuilder:validation:Required
	NoOfSamples string `json:"noOfSamples"`

	// +kubebuilder:validation:Required
	AverageUtilization UtilizationValue `json:"averageUtilization"`

	// +kubebuilder:validation:Required
	MaxUtilization UtilizationValue `json:"maxUtilization"`

	// +kubebuilder:validation:Required
	MinUtilization UtilizationValue `json:"minUtilization"`

	// +kubebuilder:validation:Required
	EffectiveUtilization UtilizationValue `json:"effectiveUtilization"`
}

type MemUtilization struct {
	// +kubebuilder:validation:Required
	NoOfSamples string `json:"noOfSamples"`

	// +kubebuilder:validation:Required
	AverageUtilization UtilizationValue `json:"averageUtilization"`

	// +kubebuilder:validation:Required
	MaxUtilization UtilizationValue `json:"maxUtilization"`

	// +kubebuilder:validation:Required
	MinUtilization UtilizationValue `json:"minUtilization"`

	// +kubebuilder:validation:Optional
	EffectiveUtilization *UtilizationValue `json:"effectiveUtilization,omitempty"`
}

type DiskUtilization struct {
	// +kubebuilder:validation:Required
	NoOfSamples string `json:"noOfSamples"`

	// +kubebuilder:validation:Required
	AverageUtilization UtilizationValue `json:"averageUtilization"`

	// +kubebuilder:validation:Required
	MaxUtilization UtilizationValue `json:"maxUtilization"`

	// +kubebuilder:validation:Required
	MinUtilization UtilizationValue `json:"minUtilization"`

	// +kubebuilder:validation:Optional
	EffectiveUtilization *UtilizationValue `json:"effectiveUtilization,omitempty"`
}

type NetworkUtilization struct {
	// +kubebuilder:validation:Required
	NoOfSamples string `json:"noOfSamples"`

	// +kubebuilder:validation:Required
	IngressUsage UtilizationValue `json:"ingressUsage"`

	// +kubebuilder:validation:Required
	EgressUsage UtilizationValue `json:"egressUsage"`

	// +kubebuilder:validation:Required
	AverageThroughput UtilizationValue `json:"averageThroughput"`

	// +kubebuilder:validation:Required
	MaxThroughput UtilizationValue `json:"maxThroughput"`

	// +kubebuilder:validation:Required
	MinThroughput UtilizationValue `json:"minThroughput"`
}

type FlavourMetrics struct {
	// +kubebuilder:validation:Required
	NoOfSamples string `json:"noOfSamples"`

	// +kubebuilder:validation:Required
	FlavourId string `json:"flavourId"`

	// +kubebuilder:validation:Required
	AverageUtilization UtilizationValue `json:"averageUtilization"`

	// +kubebuilder:validation:Optional
	AverageThroughput *UtilizationValue `json:"averageThroughput,omitempty"`

	// +kubebuilder:validation:Required
	MaxUtilization UtilizationValue `json:"maxUtilization"`

	// +kubebuilder:validation:Required
	MinUtilization UtilizationValue `json:"minUtilization"`
}

type EdgeComputeMetrics struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9-]*$`
	ZoneId string `json:"zoneId"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=date-time
	StartTime metav1.Time `json:"startTime"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=date-time
	EndTime metav1.Time `json:"endTime"`

	// +kubebuilder:validation:Required
	CpuUtil CpuUtilization `json:"cpuUtil"`

	// +kubebuilder:validation:Required
	MemUtil MemUtilization `json:"memUtil"`

	// +kubebuilder:validation:Required
	DiskUtil DiskUtilization `json:"diskUtil"`

	// +kubebuilder:validation:Required
	NetworkUtil NetworkUtilization `json:"networkUtil"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	FlavourUtil []FlavourMetrics `json:"flavourUtil"`
}

type EdgeResUtilizeMetrics struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	EdgeMetrics []EdgeComputeMetrics `json:"edgeMetrics"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9-]*$`
	FederationContextId string `json:"federationContextId"`

	// +kubebuilder:validation:Required
	SequenceNum int64 `json:"sequenceNum"`
}

type AppAggrResUtil struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z][A-Za-z0-9_]{7,63}$`
	AppId string `json:"appId"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z][A-Za-z0-9_]{7,63}$`
	AppProvId string `json:"appProvId"`

	// +kubebuilder:validation:Required
	NoOfAppInstances int32 `json:"noOfAppInstances"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	AppInstances []string `json:"appInstances"`

	// +kubebuilder:validation:Required
	CpuUtil CpuUtilization `json:"cpuUtil"`

	// +kubebuilder:validation:Required
	MemUtil MemUtilization `json:"memUtil"`

	// +kubebuilder:validation:Required
	DiskUtil DiskUtilization `json:"diskUtil"`

	// +kubebuilder:validation:Required
	NetworkUtil NetworkUtilization `json:"networkUtil"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	FlavourUtil []FlavourMetrics `json:"flavourUtil"`
}

type AppsResUtilizeMetrics struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9-]*$`
	ZoneId string `json:"zoneId"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=date-time
	StartTime metav1.Time `json:"startTime"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=date-time
	EndTime metav1.Time `json:"endTime"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	AppZoneMetrics []AppAggrResUtil `json:"appZoneMetrics"`
}

type AppsResUtilizeInfo struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	AppMetrics []AppsResUtilizeMetrics `json:"appMetrics"`
}

type Application struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z][A-Za-z0-9_]{7,63}$`
	AppProviderId string `json:"appProviderId"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9_]{6,62}[A-Za-z0-9]$`
	AppInstanceId string `json:"appInstanceId"`
}

// +kubebuilder:validation:XValidation:rule="self.monType != 'app_resource' || has(self.application)",message="application is required when monType is app_resource"
type ConsumptionMonitoringSpec struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=HOST;GUEST
	RelationType string `json:"relationType"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9-]*$`
	FederationContextId string `json:"federationContextId"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9-]*$`
	ZoneId string `json:"zoneId,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=edge_resource;app_resource;alarm;all
	MonType string `json:"monType"`

	// +kubebuilder:validation:Optional
	Periodicity *PeriodicityInterval `json:"periodicity,omitempty"`

	// +kubebuilder:validation:Optional
	Application *Application `json:"application,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format=uri
	ResMonNotifLink string `json:"resMonNotifLink,omitempty"`
}

type ConsumptionMonitoringStatus struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=PENDING;AVAILABLE;NOT_AVAILABLE;FAILED
	// +kubebuilder:default=PENDING
	State ConsumptionMonitoringState `json:"state"`

	// +kubebuilder:validation:Optional
	EdgeMetrics *[]EdgeResUtilizeMetrics `json:"edgeMetrics,omitempty"`

	// +kubebuilder:validation:Optional
	AppMetrics *[]AppsResUtilizeMetrics `json:"appMetrics,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=0
	SequenceNum int64 `json:"sequenceNum"`

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
// +kubebuilder:resource:shortName=rcm,scope=Namespaced
// +kubebuilder:printcolumn:name="relationType",type="string",JSONPath=".spec.relationType"
// +kubebuilder:printcolumn:name="federationContextId",type="string",JSONPath=".spec.federationContextId"
// +kubebuilder:printcolumn:name="zoneId",type="string",JSONPath=".spec.zoneId"
// +kubebuilder:printcolumn:name="monType",type="string",JSONPath=".spec.monType"
// +kubebuilder:printcolumn:name="state",type="string",JSONPath=".status.state"
type ConsumptionMonitoring struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`
	// +required
	Spec ConsumptionMonitoringSpec `json:"spec"`

	// +optional
	Status ConsumptionMonitoringStatus `json:"status,omitzero"`
}

func (r *ConsumptionMonitoring) ResourceIdKey() string {
	key := r.Spec.FederationContextId + r.Spec.MonType + r.Spec.ZoneId
	if r.Spec.Application != nil {
		key += r.Spec.Application.AppInstanceId
	}
	return key
}

// +kubebuilder:object:root=true
type ConsumptionMonitoringList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ConsumptionMonitoring `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ConsumptionMonitoring{}, &ConsumptionMonitoringList{})
}
