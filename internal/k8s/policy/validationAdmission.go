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

package k8sPolicy

import (
	"context"
	"fmt"
	"strings"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingadmissionpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingadmissionpolicybindings,verbs=get;list;watch;create;update;patch;delete

// FederationReconciler reconciles a Federation object
type FederationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	opg.OPGClientsMapInterface
}

func (r *FederationReconciler) FederationContextIdPolicy(ctx context.Context, role v1beta1.FederationRelation, policyName string, action string, federationContextID string) error {
	log := ctrl.Log
	listFed := &v1beta1.FederationList{}

	// Retrive all the federations and filter by the specified role and federationContextID
	if err := r.Client.List(ctx, listFed); err != nil {
		log.Error(err, ">>> [Federation] Failed to list Federations")
		return err
	}

	idMap := make(map[string]bool)
	// List all the federations and filter by the specified role and federationContextID
	for _, fed := range listFed.Items {
		// Add the federationContextId to the map if it matches the role and has a non-empty federationContextId
		if fed.Spec.FederationData.RelationType == string(role) && fed.Status.FederationContextId != "" {
			idMap[fed.Status.FederationContextId] = true
		}
	}

	// Handle the action on the specific element
	if federationContextID != "" {
		switch action {
		case "remove":
			// Delete the element from the map (useful in case it has already appeared in the List cache)
			delete(idMap, federationContextID)
		case "add":
			// Add the element to the map (useful in case it has not yet appeared in the List cache)
			idMap[federationContextID] = true
		default:
			log.Error(fmt.Errorf("invalid action"), ">>> [Federation][Policy] Invalid action provided", "action", action)
			return fmt.Errorf("invalid action: %s", action)
		}
	}

	// Transform the map into a slice of strings to be used in the CEL expression
	var federationContextIds []string
	for id := range idMap {
		federationContextIds = append(federationContextIds, id)
	}

	// Build the CEL expression based on the federationContextIds and the role
	celList := "[]"
	if len(federationContextIds) > 0 {
		var celListItems []string
		for _, id := range federationContextIds {
			celListItems = append(celListItems, fmt.Sprintf(`"%s"`, id))
		}
		celList = fmt.Sprintf("[%s]", strings.Join(celListItems, ","))
	}
	celExpression := fmt.Sprintf(`
	(request.operation == 'DELETE' ? 
		(has(oldObject.spec) && has(oldObject.spec.relationType) &&
		(oldObject.spec.relationType == '%s' ? 
			(has(oldObject.spec.federationContextId) && oldObject.spec.federationContextId in %s) : true)) 
	: 
		(has(object.spec) && has(object.spec.relationType) &&
		(object.spec.relationType == '%s' ? 
			(has(object.spec.federationContextId) && object.spec.federationContextId in %s) : true))
	)
`, string(role), celList, string(role), celList)

	// Build and Patch the ValidatingAdmissionPolicy
	policy := &admissionregistrationv1.ValidatingAdmissionPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingAdmissionPolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: policyName,
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicySpec{
			FailurePolicy: ptr.To(admissionregistrationv1.Fail),
			MatchConstraints: &admissionregistrationv1.MatchResources{
				ResourceRules: []admissionregistrationv1.NamedRuleWithOperations{
					{
						RuleWithOperations: admissionregistrationv1.RuleWithOperations{
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
								admissionregistrationv1.Update,
								admissionregistrationv1.Delete,
							},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{"opg.ewbi.katalis.com"},
								APIVersions: []string{"v1beta1"},
								Resources:   []string{"images", "artefacts", "applicationonboardings", "applicationdeployments"},
							},
						},
					},
				},
			},
			Validations: []admissionregistrationv1.Validation{
				{
					Expression: celExpression,
					Message:    "No federationContextId found in the list of federationContextIds accepted",
				},
			},
		},
	}

	err := r.Client.Patch(ctx, policy, client.Apply, client.ForceOwnership, client.FieldOwner("federation-controller"))
	if err != nil {
		log.Error(err, ">>> [Federation][Policy] Impossible to patch ValidatingAdmissionPolicy", "policyName", policyName)
		return err
	}

	// Build and Patch the Binding (remains unchanged)
	bindingName := policyName + "-binding"
	binding := &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admissionregistration.k8s.io/v1",
			Kind:       "ValidatingAdmissionPolicyBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: bindingName,
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName: policyName,
			ValidationActions: []admissionregistrationv1.ValidationAction{
				admissionregistrationv1.Deny,
			},
			MatchResources: &admissionregistrationv1.MatchResources{},
		},
	}

	err = r.Client.Patch(ctx, binding, client.Apply, client.ForceOwnership, client.FieldOwner("federation-controller"))
	if err != nil {
		log.Error(err, ">>> [Federation][Policy] Impossible to patch ValidatingAdmissionPolicyBinding", "bindingName", bindingName)
		return err
	}

	log.Info(">>> [Federation][Policy] SUCCESSFULLY", "policyName", policyName, "validIDsCount", len(federationContextIds), "actionPerformed", action)
	return nil
}
