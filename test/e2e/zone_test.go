package e2e

import (
	"context"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	opv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
)

func runZoneStage(t *testing.T, tc *testContext, partneropClient, originatingopClient client.Client) {
	if tc.federationContextID == "" {
		t.Fatalf("federationContextId from stage 01 (Federation) is required")
	}
	if len(tc.zoneDetails) == 0 {
		t.Fatalf("zoneDetails from stage 01 (Federation) is required")
	}

	ctx := context.Background()

	zone := loadSample[opv1beta1.AvailabilityZone](t, "zone.yaml")
	zone.Namespace = tc.originatingopNamespace

	tc.zoneID = tc.zoneDetails[0].ZoneId
	zone.Name = "zone-" + tc.zoneID
	zone.Spec.FederationContextId = tc.federationContextID
	zone.Spec.ZoneId = tc.zoneID

	reused := createOrGet(t, ctx, originatingopClient, zone)
	tc.zoneName = zone.Name
	t.Logf("AvailabilityZone %q %s (zoneId=%s).", tc.zoneName, reusedLabel(reused), tc.zoneID)

	partneropList := &opv1beta1.AvailabilityZoneList{}
	found := findObject(t, ctx, partneropClient, partneropList, tc.partneropNamespace,
		"AvailabilityZone (zoneId="+tc.zoneID+")",
		func(obj client.Object) bool {
			z := obj.(*opv1beta1.AvailabilityZone)
			return z.Spec.ZoneId == tc.zoneID && z.Spec.FederationContextId == tc.federationContextID && z.Status.State != ""
		},
		2*time.Minute,
	)
	partneropZone := found.(*opv1beta1.AvailabilityZone)
	t.Logf("Found partnerop AvailabilityZone %q (state=%s).", partneropZone.Name, partneropZone.Status.State)

	if partneropZone.Status.State != opv1beta1.ZoneStateNotAvailable {
		t.Fatalf("expected partnerop AvailabilityZone initial state NOT_AVAILABLE, got %s", partneropZone.Status.State)
	}

	waitBeforePartneropUpdate(t)

	partneropZone.Status.State = opv1beta1.ZoneStateAvailable
	if err := partneropClient.Status().Update(ctx, partneropZone); err != nil {
		t.Fatalf("patching partnerop AvailabilityZone %q status: %v", partneropZone.Name, err)
	}
	t.Logf("Patched partnerop AvailabilityZone %q status to AVAILABLE.", partneropZone.Name)

	waitForCondition(t, ctx, originatingopClient, zone, "availabilityzone/"+tc.zoneName, func() bool {
		return zone.Status.State == opv1beta1.ZoneStateAvailable
	}, 2*time.Minute)
}
