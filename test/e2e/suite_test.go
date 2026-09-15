package e2e

import (
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	opv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
)

type testContext struct {
	originatingopNamespace string
	partneropNamespace     string
	partneropContextName   string

	federationName      string
	federationContextID string
	origOPFederationID  string
	zoneDetails         []opv1beta1.ZoneDetails

	zoneName string
	zoneID   string

	imageName string
	imageID   string

	artefactName string
	artefactID   string

	appOnboardName string
	appID          string

	appDeployName string
	appInstanceID string
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(opv1beta1.AddToScheme(scheme))
	return scheme
}

func buildClient(t *testing.T, contextName string) client.Client {
	t.Helper()

	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		t.Fatalf("loading kubeconfig for context %q: %v", contextName, err)
	}

	c, err := client.New(cfg, client.Options{Scheme: newScheme()})
	if err != nil {
		t.Fatalf("building client for context %q: %v", contextName, err)
	}
	return c
}

// entry point
func TestFederationWalkthrough(t *testing.T) {
	partneropContext := getenv("E2E_PARTNEROP_CONTEXT", "kind-partnerop")
	originatingopContext := getenv("E2E_ORIGINATINGOP_CONTEXT", "kind-originatingop")

	tc := &testContext{
		originatingopNamespace: getenv("E2E_ORIGINATINGOP_NAMESPACE", "originatingop"),
		partneropNamespace:     getenv("E2E_PARTNEROP_NAMESPACE", "partnerop"),
		partneropContextName:   partneropContext,
	}

	partneropClient := buildClient(t, partneropContext)
	originatingopClient := buildClient(t, originatingopContext)

	t.Run("01_Federation", func(t *testing.T) { runFederationStage(t, tc, partneropClient, originatingopClient) })
	t.Run("02_AvailabilityZone", func(t *testing.T) { runZoneStage(t, tc, partneropClient, originatingopClient) })
	t.Run("03_Image", func(t *testing.T) { runImageStage(t, tc, partneropClient, originatingopClient) })
	t.Run("04_Artefact", func(t *testing.T) { runArtefactStage(t, tc, partneropClient, originatingopClient) })
	t.Run("05_ApplicationOnboarding", func(t *testing.T) { runAppOnboardingStage(t, tc, partneropClient, originatingopClient) })
	t.Run("06_ApplicationDeployment", func(t *testing.T) { runAppDeploymentStage(t, tc, partneropClient, originatingopClient) })
}
