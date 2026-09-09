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

package rest

import (
	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
)

func CompareSameAZs(s1, s2 []v1beta1.ZoneDetails) bool {
	if len(s1) != len(s2) {
		return false
	}
	set := make(map[string]bool)
	for _, v := range s1 {
		set[v.ZoneId] = true
	}
	for _, v := range s2 {
		if !set[v.ZoneId] {
			return false
		}
	}
	return true
}

func mapServiceEndpointToK8s(apiEndpoint *models.ServiceEndpoint) *v1beta1.ServiceEndpoint {
	if apiEndpoint == nil {
		return nil // Se l'API non ci dà nulla, restituiamo nil
	}
	k8sEndpoint := &v1beta1.ServiceEndpoint{
		Port: apiEndpoint.Port,
	}

	if apiEndpoint.Fqdn != nil {
		k8sEndpoint.Fqdn = *apiEndpoint.Fqdn
	}
	if apiEndpoint.Ipv4Addresses != nil {
		source := *apiEndpoint.Ipv4Addresses
		for i, ip := range source {
			k8sEndpoint.Ipv4Addresses[i] = ip
		}
	}

	if apiEndpoint.Ipv6Addresses != nil {
		source := *apiEndpoint.Ipv6Addresses
		k8sEndpoint.Ipv6Addresses = make([]string, len(source))
		for i, ip := range source {
			k8sEndpoint.Ipv6Addresses[i] = ip.(string)
		}
	}

	return k8sEndpoint
}
