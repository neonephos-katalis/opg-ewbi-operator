package e2e

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	opv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
)

func runFederationStage(t *testing.T, tc *testContext, partneropClient, originatingopClient client.Client) {
	ctx := context.Background()

	fed := loadSample[opv1beta1.Federation](t, "federation.yaml")
	fed.Namespace = tc.originatingopNamespace
	fed.Spec.FederationData.InitialDate = metav1.Now()
	tc.origOPFederationID = fed.Spec.FederationData.OrigOPFederationId

	fed.Spec.FederationData.K8sOptions = &opv1beta1.K8sOptions{
		Namespace:   tc.partneropNamespace,
		SecretName:  "kubeconfig-secret",
		ContextName: tc.partneropContextName,
	}

	reused := createOrGet(t, ctx, originatingopClient, fed)
	tc.federationName = fed.Name
	t.Logf("Federation %q %s.", tc.federationName, reusedLabel(reused))

	partneropList := &opv1beta1.FederationList{}
	found := findObject(t, ctx, partneropClient, partneropList, tc.partneropNamespace,
		"Federation (origOPFederationId="+tc.origOPFederationID+")",
		func(obj client.Object) bool {
			f := obj.(*opv1beta1.Federation)
			return f.Spec.FederationData != nil &&
				f.Spec.FederationData.RelationType == string(opv1beta1.FederationRelationHost) &&
				f.Spec.FederationData.OrigOPFederationId == tc.origOPFederationID &&
				f.Status.FederationContextId != ""
		},
		5*time.Minute,
	)
	partneropFed := found.(*opv1beta1.Federation)
	t.Logf("Found partnerop Federation %q.", partneropFed.Name)

	expiry := metav1.NewTime(time.Now().UTC().AddDate(0, 1, 0))

	waitBeforePartneropUpdate(t)

	partneropFed.Status.FederationExpiryDate = expiry
	partneropFed.Status.State = opv1beta1.FederationStateAvailable
	if err := partneropClient.Status().Update(ctx, partneropFed); err != nil {
		t.Fatalf("patching partnerop Federation %q status: %v", partneropFed.Name, err)
	}
	t.Logf("Patched partnerop Federation %q status: expiry=%s, state=AVAILABLE.", partneropFed.Name, expiry.Format(time.RFC3339))

	waitForCondition(t, ctx, originatingopClient, fed, "federation/"+tc.federationName, func() bool {
		return fed.Status.FederationContextId != "" && fed.Status.State == opv1beta1.FederationStateAvailable
	}, 5*time.Minute)

	tc.federationContextID = fed.Status.FederationContextId
	t.Logf("federationContextId = %s, state = %s", tc.federationContextID, fed.Status.State)

	tc.zoneDetails = []opv1beta1.ZoneDetails{
		{
			GeographyDetails: "Milano, Italia. Area urbana a forte densità di traffico e industriale.",
			Geolocation:      "45.4642,9.1900",
			ZoneId:           genUUID(),
		},
		{
			GeographyDetails: "Roma, Italia. Area periferica residenziale, bassa densità di popolazione.",
			Geolocation:      "41.9028,12.4964",
			ZoneId:           "3a8fffaf-50de-4f93-8c6f-05f1c84b5a5f",
		},
		{
			GeographyDetails: "Torino, Italia. Zona rurale vicino al confine con la Francia.",
			Geolocation:      "45.0703,7.6869",
			ZoneId:           "4a8fffaf-50de-4f93-8c6f-05f1c84b5a5f",
		},
	}

	if err := partneropClient.Get(ctx, client.ObjectKeyFromObject(partneropFed), partneropFed); err != nil {
		t.Fatalf("re-fetching partnerop Federation %q: %v", partneropFed.Name, err)
	}

	waitBeforePartneropUpdate(t)

	partneropFed.Status.ZoneDetails = tc.zoneDetails
	if err := partneropClient.Status().Update(ctx, partneropFed); err != nil {
		t.Fatalf("patching partnerop Federation %q status.zoneDetails: %v", partneropFed.Name, err)
	}
	t.Logf("Patched partnerop Federation %q status.zoneDetails.", partneropFed.Name)
}
