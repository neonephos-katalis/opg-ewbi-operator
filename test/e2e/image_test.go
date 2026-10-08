package e2e

import (
	"context"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	opv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
)

func runImageStage(t *testing.T, tc *testContext, partneropClient, originatingopClient client.Client) {
	if tc.federationContextID == "" {
		t.Fatalf("federationContextId from stage 01 (Federation) is required")
	}

	ctx := context.Background()

	image := loadSample[opv1beta1.Image](t, "image.yaml")
	image.Namespace = tc.originatingopNamespace

	tc.imageID = genUUID()
	image.Name = "image-" + tc.imageID
	image.Spec.FederationContextId = tc.federationContextID
	image.Spec.ImageId = tc.imageID

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
	t.Logf("Patched partnerop Image %q status to READY.", partneropImage.Name)

	waitForCondition(t, ctx, originatingopClient, image, "image/"+tc.imageName, func() bool {
		return image.Status.State == opv1beta1.ImageStateReady
	}, 2*time.Minute)
}
