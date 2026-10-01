package e2e

import (
	"context"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	opv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
)

func runArtefactStage(t *testing.T, tc *testContext, partneropClient, originatingopClient client.Client) {
	if tc.federationContextID == "" {
		t.Fatalf("federationContextId from stage 01 (Federation) is required")
	}
	if tc.imageID == "" {
		t.Fatalf("imageId from stage 03 (Image) is required")
	}

	ctx := context.Background()

	artefact := loadSample[opv1beta1.Artefact](t, "artefact.yaml")
	artefact.Namespace = tc.originatingopNamespace

	tc.artefactID = genUUID()
	artefact.Name = "art-" + tc.artefactID
	artefact.Spec.FederationContextId = tc.federationContextID
	artefact.Spec.ArtefactId = tc.artefactID

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
	t.Logf("Patched partnerop Artefact %q status to READY.", partneropArtefact.Name)

	waitForCondition(t, ctx, originatingopClient, artefact, "artefact/"+tc.artefactName, func() bool {
		return artefact.Status.State == opv1beta1.ArtefactStateReady
	}, 2*time.Minute)
}
