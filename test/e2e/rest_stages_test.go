package e2e

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	opv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
)

// restDefaultZoneID is the zone used by ApplicationOnboarding / ApplicationDeployment on the
// REST path: zones are not registered over REST, so the zone id is only carried in the specs.
const restDefaultZoneID = "2a8fffaf-50de-4f93-8c6f-05f1c84b5a5f"

func runRestFederationStage(t *testing.T, tc *testContext, rc *restContext, partneropClient, originatingopClient client.Client) {
	ctx := context.Background()

	fed := loadRestSample[opv1beta1.Federation](t, "federation.yaml")
	fed.Namespace = tc.originatingopNamespace
	fed.Spec.FederationData.InitialDate = metav1.Now()
	fed.Spec.FederationData.K8sOptions = nil
	fed.Spec.FederationData.RestOptions = &opv1beta1.RestOptions{
		TokenUrl:          rc.partneropAPIURL,
		PartnerStatusLink: rc.originatingopAPIURL,
	}
	tc.origOPFederationID = fed.Spec.FederationData.OrigOPFederationId

	reused := createOrGet(t, ctx, originatingopClient, fed)
	tc.federationName = fed.Name
	t.Logf("Federation %q %s (technologyType=%s, tokenUrl=%s).",
		tc.federationName, reusedLabel(reused), fed.Spec.FederationData.TechnologyType, rc.partneropAPIURL)

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
	t.Logf("Found partnerop Federation %q created by the partnerop API (technologyType=%s).",
		partneropFed.Name, partneropFed.Spec.FederationData.TechnologyType)

	if partneropFed.Spec.FederationData.TechnologyType != string(opv1beta1.FederationTechnologyRest) {
		t.Fatalf("expected partnerop Federation technologyType REST, got %s", partneropFed.Spec.FederationData.TechnologyType)
	}

	// Push the expiry date forward so the federation does not get LOCKED during the run
	// (a LOCKED federation stops all REST callbacks).
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
	tc.zoneID = restDefaultZoneID
	t.Logf("federationContextId = %s, state = %s", tc.federationContextID, fed.Status.State)

	// The federationContextId validation policies are written asynchronously by the controllers:
	// wait for both federations to have them in place, otherwise the next resources are denied
	// by opg-ewbi-fedguest-validation-policy / opg-ewbi-fedhost-validation-policy.
	waitForCondition(t, ctx, originatingopClient, fed, "federation/"+tc.federationName+" (validation policy set)", func() bool {
		return fed.Annotations[opv1beta1.FederationPolicyAnnotation] == "set"
	}, 2*time.Minute)
	waitForCondition(t, ctx, partneropClient, partneropFed, "partnerop federation/"+partneropFed.Name+" (validation policy set)", func() bool {
		return partneropFed.Annotations[opv1beta1.FederationPolicyAnnotation] == "set"
	}, 2*time.Minute)
	t.Logf("federationContextId validation policies are in place on both sides.")
}

func runRestImageStage(t *testing.T, tc *testContext, rc *restContext, partneropClient, originatingopClient client.Client) {
	if tc.federationContextID == "" {
		t.Fatalf("federationContextId from stage 01 (Federation) is required")
	}

	ctx := context.Background()

	image := loadRestSample[opv1beta1.Image](t, "image.yaml")
	image.Namespace = tc.originatingopNamespace

	tc.imageID = genUUID()
	image.Name = "image-" + tc.imageID
	image.Spec.FederationContextId = tc.federationContextID
	image.Spec.ImageId = tc.imageID
	image.Spec.ImageNotifLink = rc.originatingopAPIURL

	reused := createOrGet(t, ctx, originatingopClient, image)
	tc.imageName = image.Name
	t.Logf("Image %q %s (imageId=%s).", tc.imageName, reusedLabel(reused), tc.imageID)

	partneropList := &opv1beta1.ImageList{}
	found := findObject(t, ctx, partneropClient, partneropList, tc.partneropNamespace,
		"Image (imageId="+tc.imageID+")",
		func(obj client.Object) bool {
			i := obj.(*opv1beta1.Image)
			return i.Spec.ImageId == tc.imageID && i.Spec.FederationContextId == tc.federationContextID && i.Status.State != ""
		},
		2*time.Minute,
	)
	partneropImage := found.(*opv1beta1.Image)
	t.Logf("Found partnerop Image %q (state=%s).", partneropImage.Name, partneropImage.Status.State)

	if partneropImage.Status.State != opv1beta1.ImageStatePending {
		t.Fatalf("expected partnerop Image initial state PENDING, got %s", partneropImage.Status.State)
	}

	waitBeforePartneropUpdate(t)

	partneropImage.Status.State = opv1beta1.ImageStateReady
	if err := partneropClient.Status().Update(ctx, partneropImage); err != nil {
		t.Fatalf("patching partnerop Image %q status: %v", partneropImage.Name, err)
	}
	t.Logf("Patched partnerop Image %q status to READY (callback to %s).", partneropImage.Name, rc.originatingopAPIURL)

	waitForCondition(t, ctx, originatingopClient, image, "image/"+tc.imageName, func() bool {
		return image.Status.State == opv1beta1.ImageStateReady
	}, 2*time.Minute)
}

func runRestArtefactStage(t *testing.T, tc *testContext, rc *restContext, partneropClient, originatingopClient client.Client) {
	if tc.federationContextID == "" {
		t.Fatalf("federationContextId from stage 01 (Federation) is required")
	}
	if tc.imageID == "" {
		t.Fatalf("imageId from stage 03 (Image) is required")
	}

	ctx := context.Background()

	artefact := loadRestSample[opv1beta1.Artefact](t, "artefact.yaml")
	artefact.Namespace = tc.originatingopNamespace

	tc.artefactID = genUUID()
	artefact.Name = "art-" + tc.artefactID
	artefact.Spec.FederationContextId = tc.federationContextID
	artefact.Spec.ArtefactId = tc.artefactID
	artefact.Spec.ArtefactNotifLink = rc.originatingopAPIURL
	if artefact.Spec.ArtefactBody != nil {
		for i := range artefact.Spec.ArtefactBody.ComponentSpec {
			artefact.Spec.ArtefactBody.ComponentSpec[i].Images = []string{tc.imageID}
		}
	}

	reused := createOrGet(t, ctx, originatingopClient, artefact)
	tc.artefactName = artefact.Name
	t.Logf("Artefact %q %s (artefactId=%s).", tc.artefactName, reusedLabel(reused), tc.artefactID)

	partneropList := &opv1beta1.ArtefactList{}
	found := findObject(t, ctx, partneropClient, partneropList, tc.partneropNamespace,
		"Artefact (artefactId="+tc.artefactID+")",
		func(obj client.Object) bool {
			a := obj.(*opv1beta1.Artefact)
			return a.Spec.ArtefactId == tc.artefactID && a.Spec.FederationContextId == tc.federationContextID && a.Status.State != ""
		},
		2*time.Minute,
	)
	partneropArtefact := found.(*opv1beta1.Artefact)
	t.Logf("Found partnerop Artefact %q (state=%s).", partneropArtefact.Name, partneropArtefact.Status.State)

	if partneropArtefact.Status.State != opv1beta1.ArtefactStatePending {
		t.Fatalf("expected partnerop Artefact initial state PENDING, got %s", partneropArtefact.Status.State)
	}

	waitBeforePartneropUpdate(t)

	partneropArtefact.Status.State = opv1beta1.ArtefactStateReady
	if err := partneropClient.Status().Update(ctx, partneropArtefact); err != nil {
		t.Fatalf("patching partnerop Artefact %q status: %v", partneropArtefact.Name, err)
	}
	t.Logf("Patched partnerop Artefact %q status to READY (callback to %s).", partneropArtefact.Name, rc.originatingopAPIURL)

	waitForCondition(t, ctx, originatingopClient, artefact, "artefact/"+tc.artefactName, func() bool {
		return artefact.Status.State == opv1beta1.ArtefactStateReady
	}, 2*time.Minute)
}

func runRestAppOnboardingStage(t *testing.T, tc *testContext, rc *restContext, partneropClient, originatingopClient client.Client) {
	if tc.federationContextID == "" {
		t.Fatalf("federationContextId from stage 01 (Federation) is required")
	}
	if tc.artefactID == "" {
		t.Fatalf("artefactId from stage 04 (Artefact) is required")
	}

	ctx := context.Background()

	appOnboard := loadRestSample[opv1beta1.ApplicationOnboarding](t, "appOnboard.yaml")
	appOnboard.Namespace = tc.originatingopNamespace

	tc.appID = genAppID()
	appOnboard.Name = "apponboard-" + tc.appID
	appOnboard.Spec.FederationContextId = tc.federationContextID
	appOnboard.Spec.AppInfo.AppId = tc.appID
	appOnboard.Spec.AppInfo.AppStatusCallbackLink = rc.originatingopAPIURL
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
	t.Logf("Patched partnerop ApplicationOnboarding %q status to ONBOARDED (callback to %s).", partneropAppOnboard.Name, rc.originatingopAPIURL)

	waitForCondition(t, ctx, originatingopClient, appOnboard, "applicationonboarding/"+tc.appOnboardName, func() bool {
		return appOnboard.Status.State == opv1beta1.ApplicationOnboardingStateOnboarded
	}, 2*time.Minute)
}

func runRestAppDeploymentStage(t *testing.T, tc *testContext, rc *restContext, partneropClient, originatingopClient client.Client) {
	if tc.federationContextID == "" {
		t.Fatalf("federationContextId from stage 01 (Federation) is required")
	}
	if tc.appID == "" {
		t.Fatalf("appId from stage 05 (ApplicationOnboarding) is required")
	}

	ctx := context.Background()

	appDeploy := loadRestSample[opv1beta1.ApplicationDeployment](t, "appDeploy.yaml")
	appDeploy.Namespace = tc.originatingopNamespace

	tc.appInstanceID = genAppInstanceID()
	appDeploy.Name = "appdeploy-" + tc.appInstanceID
	appDeploy.Spec.FederationContextId = tc.federationContextID
	appDeploy.Spec.AppId = tc.appID
	appDeploy.Spec.AppInstanceId = tc.appInstanceID
	appDeploy.Spec.ZoneId = tc.zoneID
	if appDeploy.Spec.AppDetails == nil {
		appDeploy.Spec.AppDetails = &opv1beta1.AppDetails{}
	}
	// The REST sample points this callback at the partnerop API: the partner must call back the originating OP.
	appDeploy.Spec.AppDetails.AppInstCallbackLink = rc.originatingopAPIURL

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
	t.Logf("Patched partnerop ApplicationDeployment %q status to READY (callback to %s).", partneropAppDeploy.Name, rc.originatingopAPIURL)

	waitForCondition(t, ctx, originatingopClient, appDeploy, "applicationdeployment/"+tc.appDeployName, func() bool {
		return appDeploy.Status.AppInstanceInfo.AppInstanceState == opv1beta1.ApplicationDeploymentStateReady
	}, 2*time.Minute)

	t.Logf("REST resource chain created: Federation=%s Image=%s Artefact=%s ApplicationOnboarding=%s ApplicationDeployment=%s",
		tc.federationName, tc.imageName, tc.artefactName, tc.appOnboardName, tc.appDeployName)
}
