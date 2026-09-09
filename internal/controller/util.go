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

package controller

import (
	"context"
	"errors"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetFederationByContextId(ctx context.Context, c client.Client, fedCtxId string, namespace string) (*v1beta1.Federation, error) {
	log := ctrl.Log

	var federList v1beta1.FederationList
	listOpts := &client.ListOptions{
		Namespace: namespace, // 🛡️ Limitiamo la ricerca a un singolo namespace!
	}
	if err := c.List(ctx, &federList, listOpts); err != nil {
		log.Error(err, "[CONTROLLER] Error listing federation objects.")
		return nil, err
	}
	var matchedFeds []v1beta1.Federation
	for _, fed := range federList.Items {
		if fed.Status.FederationContextId == fedCtxId {
			matchedFeds = append(matchedFeds, fed)
		}
	}
	if len(matchedFeds) == 0 {
		return nil, errors.New("federation not found for context ID")
	}
	if len(matchedFeds) > 1 {
		log.Info("[CONTROLLER] Unexpected number of federations, should be 1.", "actual", len(matchedFeds))
		return nil, errors.New("[CONTROLLER] Unexpected number of federations for this resource, should be 1.")
	}

	return &matchedFeds[0], nil
}

func GetFederation(ctx context.Context, isGuest bool, r client.Client, federationContextId string, namespace string) (*v1beta1.Federation, bool, error) {
	f, err := GetFederationByContextId(ctx, r, federationContextId, namespace)
	if err != nil {
		return nil, false, err
	}
	isRest := IsRestTechnology(f.Spec.FederationData.TechnologyType)
	return f, isRest, nil
}

// returns true if LabelValue is v1beta1.FederationRelationGuest
// false otherwise (either label wasn't present or is RelationHost)
func IsGuestResource(relation string) bool {
	return relation == string(v1beta1.FederationRelationGuest)
}

// returns true if LabelValue is v1beta1.FederationTechnologyRest
// false otherwise (either label wasn't present or is another technology)
func IsRestTechnology(technology string) bool {
	return technology == string(v1beta1.FederationTechnologyRest)
}

func CheckFederationState(fed *v1beta1.Federation, isRest bool, prefix string, name string, namespace string) bool {
	log := ctrl.Log
	switch fed.Status.State {
	case v1beta1.FederationStateLocked:
		switch isRest {
		case false:
			log.Info(">>> ["+prefix+"] Federation is LOCKED, stopping watcher. The Federation Expiry Date is reached", "name", name, "namespace", namespace, "expiryDate", fed.Status.FederationExpiryDate)
		case true:
			log.Info(">>> ["+prefix+"] Federation is LOCKED, stopping perform CALLBACKs via OPG EWBI API. The Federation Expiry Date is reached", "name", name, "namespace", namespace, "expiryDate", fed.Status.FederationExpiryDate)
		}
		return false
	case v1beta1.FederationStateAvailable:
		// If necessary, restart the watcher if it was previously stopped due to federation being locked
		log.Info(">>> ["+prefix+"] Federation is AVAILABLE.", "name", name, "namespace", namespace)
		return true
	default:
		log.Info(">>> ["+prefix+"] Federation is in an other state.", "name", name, "namespace", namespace, "state", fed.Status.State)
		return false
	}
}
