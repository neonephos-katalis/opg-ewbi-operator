package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// REST walkthrough.
//
// Same scenario as TestFederationWalkthrough, but the originating OP federates with the
// partner OP through the EWBI REST API instead of the Kubernetes API:
//
//   - originatingop controller -> partnerop API   (tokenUrl)
//   - partnerop controller     -> originatingop API (partnerStatusLink and the *NotifLink / *CallbackLink of each resource)
//
// Required environment variables (set by .github/workflows/e2e-federation-rest.yml):
//
//	E2E_PARTNEROP_API_URL      base URL of the partnerop EWBI API, reachable from originatingop pods
//	E2E_ORIGINATINGOP_API_URL  base URL of the originatingop EWBI API, reachable from partnerop pods
//
// AvailabilityZone and ConsumptionMonitoring are not implemented on the REST path
// (internal/rest/availabilityzones_rest.go, internal/rest/consumptionmonitoring_rest.go),
// so those stages are reported as SKIP.
func TestRestFederationWalkthrough(t *testing.T) {
	partneropContext := getenv("E2E_PARTNEROP_CONTEXT", "kind-partnerop")
	originatingopContext := getenv("E2E_ORIGINATINGOP_CONTEXT", "kind-originatingop")

	partneropAPIURL := strings.TrimRight(os.Getenv("E2E_PARTNEROP_API_URL"), "/")
	originatingopAPIURL := strings.TrimRight(os.Getenv("E2E_ORIGINATINGOP_API_URL"), "/")
	if partneropAPIURL == "" || originatingopAPIURL == "" {
		t.Fatalf("E2E_PARTNEROP_API_URL and E2E_ORIGINATINGOP_API_URL must be set (got %q, %q)", partneropAPIURL, originatingopAPIURL)
	}

	tc := &testContext{
		originatingopNamespace: getenv("E2E_ORIGINATINGOP_NAMESPACE", "originatingop"),
		partneropNamespace:     getenv("E2E_PARTNEROP_NAMESPACE", "partnerop"),
		partneropContextName:   partneropContext,
	}
	rc := &restContext{
		partneropAPIURL:     partneropAPIURL,
		originatingopAPIURL: originatingopAPIURL,
	}

	t.Logf("partnerop API URL:     %s", rc.partneropAPIURL)
	t.Logf("originatingop API URL: %s", rc.originatingopAPIURL)

	partneropClient := buildClient(t, partneropContext)
	originatingopClient := buildClient(t, originatingopContext)

	t.Run("01_Federation", func(t *testing.T) { runRestFederationStage(t, tc, rc, partneropClient, originatingopClient) })
	t.Run("02_AvailabilityZone", func(t *testing.T) {
		t.Skip("AvailabilityZone is not implemented on the REST path (AcceptZone is a no-op)")
	})
	t.Run("03_Image", func(t *testing.T) { runRestImageStage(t, tc, rc, partneropClient, originatingopClient) })
	t.Run("04_Artefact", func(t *testing.T) { runRestArtefactStage(t, tc, rc, partneropClient, originatingopClient) })
	t.Run("05_ApplicationOnboarding", func(t *testing.T) { runRestAppOnboardingStage(t, tc, rc, partneropClient, originatingopClient) })
	t.Run("06_ApplicationDeployment", func(t *testing.T) { runRestAppDeploymentStage(t, tc, rc, partneropClient, originatingopClient) })
	t.Run("07_ConsumptionMonitoring", func(t *testing.T) {
		t.Skip("ConsumptionMonitoring is not implemented on the REST path (CreateConsumptionMonitoring is a no-op)")
	})
}

// restContext holds the cross-cluster API endpoints used by the REST walkthrough.
type restContext struct {
	partneropAPIURL     string
	originatingopAPIURL string
}

// loadRestSample loads a sample from config/samples/rest.
func loadRestSample[T any](t *testing.T, fileName string) *T {
	t.Helper()

	path := filepath.Join("..", "..", "config", "samples", "rest", fileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading sample %s: %v", path, err)
	}

	var obj T
	if err := yaml.Unmarshal(raw, &obj); err != nil {
		t.Fatalf("unmarshalling sample %s: %v", path, err)
	}
	return &obj
}
