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

	"github.com/go-logr/logr"
	opgmodels "github.com/nbycomp/neonephos-opg-ewbi-api/api/federation/models"
	"github.com/nbycomp/neonephos-opg-ewbi-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func GetFederationByContextId(
	ctx context.Context, c client.Client, fedCtxId string, filterLabels map[string]string,
) (*v1beta1.Federation, error) {
	log := log.FromContext(ctx)

	var federList v1beta1.FederationList

	labelSelector := labels.SelectorFromSet(filterLabels)

	listOpts := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(
			v1beta1.FederationStatusContextIDField,
			fedCtxId,
		),
		LabelSelector: labelSelector,
	}

	if err := c.List(ctx, &federList, listOpts); err != nil {
		log.Info("error listing federation objects")
		return nil, err
	}
	if len(federList.Items) != 1 {
		log.Info("unexpected number of federations, should be 1", "actual",
			len(federList.Items))
		return nil, errors.New(
			"unexpected number of federations for this resource, should be 1",
		)
	}
	return &federList.Items[0], nil
}

func handleProblemDetails(log logr.Logger, code int, p *opgmodels.ProblemDetails) {
	log.Info("response with error", "error", code, "details", p)
}

// returns true if LabelValue is v1beta1.FederationRelationGuest
// false otherwise (either label wasn't present or is RelationHost)
func IsGuestResource(labels map[string]string) bool {
	return labels[v1beta1.FederationRelationLabel] == string(v1beta1.FederationRelationGuest)
}
