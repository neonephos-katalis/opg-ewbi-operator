package e2e

import (
	"context"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	opv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
)

func runAppDeploymentStage(t *testing.T, tc *testContext, partneropClient, originatingopClient client.Client) {
	if tc.federationContextID == "" {
		t.Fatalf("federationContextId from stage 01 (Federation) is required")
	}
	if tc.appID == "" {
		t.Fatalf("appId from stage 05 (ApplicationOnboarding) is required")
	}
	if tc.zoneID == "" {
		t.Fatalf("zoneId from stage 02 (AvailabilityZone) is required")
	}

	ctx := context.Background()

	appDeploy := loadSample[opv1beta1.ApplicationDeployment](t, "appDeploy.yaml")
	appDeploy.Namespace = tc.originatingopNamespace

	tc.appInstanceID = genAppInstanceID()
	appDeploy.Name = "appdeploy-" + tc.appInstanceID
	appDeploy.Spec.FederationContextId = tc.federationContextID
	appDeploy.Spec.AppId = tc.appID
	appDeploy.Spec.AppInstanceId = tc.appInstanceID
	appDeploy.Spec.ZoneId = tc.zoneID

	reused := createOrGet(t, ctx, originatingopClient, appDeploy)
	tc.appDeployName = appDeploy.Name
	t.Logf("ApplicationDeployment %q %s (appInstanceId=%s).", tc.appDeployName, reusedLabel(reused), tc.appInstanceID)

	partneropList := &opv1beta1.ApplicationDeploymentList{}
	found := findObject(t, ctx, partneropClient, partneropList, tc.partneropNamespace,
		"ApplicationDeployment (appInstanceId="+tc.appInstanceID+")",
		func(obj client.Object) bool {
			a := obj.(*opv1beta1.ApplicationDeployment)
			return a.Spec.AppId == tc.appID && a.Spec.AppInstanceId == tc.appInstanceID &&
				a.Spec.FederationContextId == tc.federationContextID && a.Status.AppInstanceInfo.AppInstanceState != ""
		},
		2*time.Minute,
	)
	partneropAppDeploy := found.(*opv1beta1.ApplicationDeployment)
	t.Logf("Found partnerop ApplicationDeployment %q (state=%s).", partneropAppDeploy.Name, partneropAppDeploy.Status.AppInstanceInfo.AppInstanceState)

	if partneropAppDeploy.Status.AppInstanceInfo.AppInstanceState != opv1beta1.ApplicationDeploymentStatePending {
		t.Fatalf("expected partnerop ApplicationDeployment initial state PENDING, got %s", partneropAppDeploy.Status.AppInstanceInfo.AppInstanceState)
	}

	waitBeforePartneropUpdate(t)

	partneropAppDeploy.Status.AppInstanceInfo.AppInstanceState = opv1beta1.ApplicationDeploymentStateReady
	if err := partneropClient.Status().Update(ctx, partneropAppDeploy); err != nil {
		t.Fatalf("patching partnerop ApplicationDeployment %q status: %v", partneropAppDeploy.Name, err)
	}
	t.Logf("Patched partnerop ApplicationDeployment %q status to READY.", partneropAppDeploy.Name)

	waitForCondition(t, ctx, originatingopClient, appDeploy, "applicationdeployment/"+tc.appDeployName, func() bool {
		return appDeploy.Status.AppInstanceInfo.AppInstanceState == opv1beta1.ApplicationDeploymentStateReady
	}, 2*time.Minute)

	t.Logf("Resource chain created: Federation=%s AvailabilityZone=%s Image=%s Artefact=%s ApplicationOnboarding=%s ApplicationDeployment=%s",
		tc.federationName, tc.zoneName, tc.imageName, tc.artefactName, tc.appOnboardName, tc.appDeployName)
}
