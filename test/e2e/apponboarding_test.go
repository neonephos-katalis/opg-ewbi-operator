package e2e

import (
	"context"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	opv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
)

func runAppOnboardingStage(t *testing.T, tc *testContext, partneropClient, originatingopClient client.Client) {
	if tc.federationContextID == "" {
		t.Fatalf("federationContextId from stage 01 (Federation) is required")
	}
	if tc.artefactID == "" {
		t.Fatalf("artefactId from stage 04 (Artefact) is required")
	}
	if tc.zoneID == "" {
		t.Fatalf("zoneId from stage 02 (AvailabilityZone) is required")
	}

	ctx := context.Background()

	appOnboard := loadSample[opv1beta1.ApplicationOnboarding](t, "appOnboard.yaml")
	appOnboard.Namespace = tc.originatingopNamespace

	tc.appID = genAppID()
	appOnboard.Name = "apponboard-" + tc.appID
	appOnboard.Spec.FederationContextId = tc.federationContextID
	appOnboard.Spec.AppInfo.AppId = tc.appID
	appOnboard.Spec.AppInfo.AppComponentSpecs = []opv1beta1.AppComponentSpec{{ArtefactId: tc.artefactID}}
	appOnboard.Spec.AppInfo.AppDeploymentZones = []opv1beta1.AppDeploymentZone{{ZoneId: tc.zoneID, CountryCode: "IT"}}

	reused := createOrGet(t, ctx, originatingopClient, appOnboard)
	tc.appOnboardName = appOnboard.Name
	t.Logf("ApplicationOnboarding %q %s (appId=%s).", tc.appOnboardName, reusedLabel(reused), tc.appID)

	partneropList := &opv1beta1.ApplicationOnboardingList{}
	found := findObject(t, ctx, partneropClient, partneropList, tc.partneropNamespace,
		"ApplicationOnboarding (appId="+tc.appID+")",
		func(obj client.Object) bool {
			a := obj.(*opv1beta1.ApplicationOnboarding)
			return a.Spec.AppInfo != nil && a.Spec.AppInfo.AppId == tc.appID &&
				a.Spec.FederationContextId == tc.federationContextID && a.Status.State != ""
		},
		2*time.Minute,
	)
	partneropAppOnboard := found.(*opv1beta1.ApplicationOnboarding)
	t.Logf("Found partnerop ApplicationOnboarding %q (state=%s).", partneropAppOnboard.Name, partneropAppOnboard.Status.State)

	if partneropAppOnboard.Status.State != opv1beta1.ApplicationOnboardingStatePending {
		t.Fatalf("expected partnerop ApplicationOnboarding initial state PENDING, got %s", partneropAppOnboard.Status.State)
	}

	waitBeforePartneropUpdate(t)

	partneropAppOnboard.Status.State = opv1beta1.ApplicationOnboardingStateOnboarded
	if err := partneropClient.Status().Update(ctx, partneropAppOnboard); err != nil {
		t.Fatalf("patching partnerop ApplicationOnboarding %q status: %v", partneropAppOnboard.Name, err)
	}
	t.Logf("Patched partnerop ApplicationOnboarding %q status to ONBOARDED.", partneropAppOnboard.Name)

	waitForCondition(t, ctx, originatingopClient, appOnboard, "applicationonboarding/"+tc.appOnboardName, func() bool {
		return appOnboard.Status.State == opv1beta1.ApplicationOnboardingStateOnboarded
	}, 2*time.Minute)
}
