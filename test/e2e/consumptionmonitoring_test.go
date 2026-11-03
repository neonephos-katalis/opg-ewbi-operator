package e2e

import (
	"context"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	opv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
)

func runConsumptionMonitoringStage(t *testing.T, tc *testContext, partneropClient, originatingopClient client.Client) {
	if tc.federationContextID == "" {
		t.Fatalf("federationContextId from stage 01 (Federation) is required")
	}
	if tc.zoneID == "" {
		t.Fatalf("zoneId from stage 02 (AvailabilityZone) is required")
	}
	if tc.appInstanceID == "" {
		t.Fatalf("appInstanceId from stage 06 (ApplicationDeployment) is required")
	}

	ctx := context.Background()

	cm := loadSample[opv1beta1.ConsumptionMonitoring](t, "consumptionmonitoring.yaml")
	cm.Namespace = tc.originatingopNamespace

	cm.Name = "cm-" + tc.appInstanceID
	cm.Spec.FederationContextId = tc.federationContextID
	cm.Spec.ZoneId = tc.zoneID
	cm.Spec.Application.AppInstanceId = tc.appInstanceID

	reused := createOrGet(t, ctx, originatingopClient, cm)
	tc.consumptionMonitoringName = cm.Name
	t.Logf("ConsumptionMonitoring %q %s (monType=%s, appInstanceId=%s).", tc.consumptionMonitoringName, reusedLabel(reused), cm.Spec.MonType, tc.appInstanceID)

	partneropList := &opv1beta1.ConsumptionMonitoringList{}
	found := findObject(t, ctx, partneropClient, partneropList, tc.partneropNamespace,
		"ConsumptionMonitoring (appInstanceId="+tc.appInstanceID+")",
		func(obj client.Object) bool {
			c := obj.(*opv1beta1.ConsumptionMonitoring)
			return c.Spec.Application != nil && c.Spec.Application.AppInstanceId == tc.appInstanceID &&
				c.Spec.ZoneId == tc.zoneID && c.Spec.MonType == cm.Spec.MonType &&
				c.Spec.FederationContextId == tc.federationContextID && c.Status.State != ""
		},
		2*time.Minute,
	)
	partneropCM := found.(*opv1beta1.ConsumptionMonitoring)
	t.Logf("Found partnerop ConsumptionMonitoring %q (state=%s).", partneropCM.Name, partneropCM.Status.State)

	if partneropCM.Status.State != opv1beta1.ConsumptionMonitoringStatePending {
		t.Fatalf("expected partnerop ConsumptionMonitoring initial state PENDING, got %s", partneropCM.Status.State)
	}

	waitBeforePartneropUpdate(t)

	partneropCM.Status.State = opv1beta1.ConsumptionMonitoringStateAvailable
	if err := partneropClient.Status().Update(ctx, partneropCM); err != nil {
		t.Fatalf("patching partnerop ConsumptionMonitoring %q status: %v", partneropCM.Name, err)
	}
	t.Logf("Patched partnerop ConsumptionMonitoring %q status to AVAILABLE.", partneropCM.Name)

	waitForCondition(t, ctx, originatingopClient, cm, "consumptionmonitoring/"+tc.consumptionMonitoringName, func() bool {
		return cm.Status.State == opv1beta1.ConsumptionMonitoringStateAvailable
	}, 2*time.Minute)
}
