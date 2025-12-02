#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

CLUSTER_NAME="federation-test"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
API_DIR="$(dirname "$SCRIPT_DIR")/opg-ewbi-api"

echo -e "${GREEN}=== OPG EWBI Federation Manager Setup ===${NC}"
echo ""

# Step 1: Check prerequisites
echo -e "${YELLOW}[1/11] Checking prerequisites...${NC}"
command -v go >/dev/null 2>&1 || { echo -e "${RED}go is required but not installed.${NC}" >&2; exit 1; }
command -v docker >/dev/null 2>&1 || { echo -e "${RED}docker is required but not installed.${NC}" >&2; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo -e "${RED}kubectl is required but not installed.${NC}" >&2; exit 1; }
command -v helm >/dev/null 2>&1 || { echo -e "${RED}helm is required but not installed.${NC}" >&2; exit 1; }
command -v kind >/dev/null 2>&1 || { echo -e "${RED}kind is required but not installed.${NC}" >&2; exit 1; }
echo -e "${GREEN}All prerequisites found.${NC}"

# Step 2: Create Kind cluster
echo ""
echo -e "${YELLOW}[2/11] Creating Kind cluster '${CLUSTER_NAME}'...${NC}"
if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    echo -e "${YELLOW}Cluster '${CLUSTER_NAME}' already exists. Deleting...${NC}"
    kind delete cluster --name "$CLUSTER_NAME"
fi
kind create cluster --name "$CLUSTER_NAME"
kubectl cluster-info --context "kind-${CLUSTER_NAME}"
echo -e "${GREEN}Kind cluster created.${NC}"

# Step 3: Create empty ~/.netrc if not exists
echo ""
echo -e "${YELLOW}[3/11] Ensuring ~/.netrc exists...${NC}"
if [ ! -f ~/.netrc ]; then
    touch ~/.netrc
    chmod 600 ~/.netrc
    echo -e "${GREEN}Created empty ~/.netrc${NC}"
else
    echo -e "${GREEN}~/.netrc already exists.${NC}"
fi

# Step 4: Build operator image
echo ""
echo -e "${YELLOW}[4/11] Building opg-ewbi-operator Docker image...${NC}"
cd "$SCRIPT_DIR"
make docker-build-controller
echo -e "${GREEN}Operator image built.${NC}"

# Step 5: Clone and fix opg-ewbi-api
echo ""
echo -e "${YELLOW}[5/11] Setting up opg-ewbi-api...${NC}"
if [ ! -d "$API_DIR" ]; then
    echo "Cloning opg-ewbi-api..."
    cd "$(dirname "$SCRIPT_DIR")"
    git clone https://github.com/neonephos-katalis/opg-ewbi-api.git
fi
cd "$API_DIR"

# Add replace directive if not already present
if ! grep -q "replace github.com/neonephos-katalis/opg-ewbi-operator" go.mod; then
    echo "" >> go.mod
    echo "replace github.com/neonephos-katalis/opg-ewbi-operator => ../opg-ewbi-operator" >> go.mod
    echo "Added replace directive to go.mod"
fi

# Vendor dependencies
echo "Vendoring dependencies..."
go mod vendor

# Create fixed Dockerfile
cat > Dockerfile << 'DOCKERFILE'
# syntax = docker/dockerfile:1
FROM golang:1.24.6-alpine AS builder

RUN apk add -U --no-cache ca-certificates

WORKDIR /workspace

COPY go.mod go.sum ./
COPY vendor/ vendor/
COPY . ./

RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -mod=vendor -o ./app ./cmd/app/

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/app ./
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

USER 65532:65532

ENTRYPOINT ["./app"]
DOCKERFILE

# Detect platform
PLATFORM="linux/arm64"
if [ "$(uname -m)" = "x86_64" ]; then
    PLATFORM="linux/amd64"
fi

echo "Building opg-ewbi-api Docker image for ${PLATFORM}..."
docker build --platform="$PLATFORM" -t ghcr.io/neonephos-katalis/opg-ewbi-api:neonephos .
echo -e "${GREEN}API image built.${NC}"

# Step 6: Fix Helm chart issues
echo ""
echo -e "${YELLOW}[6/11] Fixing Helm chart issues...${NC}"
cd "$SCRIPT_DIR"

# Fix hardcoded namespace in role.yaml
if grep -q "namespace: foo" dist/chart/templates/rbac/role.yaml; then
    sed -i.bak 's/namespace: foo/namespace: {{ .Release.Namespace }}/g' dist/chart/templates/rbac/role.yaml
    rm -f dist/chart/templates/rbac/role.yaml.bak
    echo "Fixed namespace in role.yaml"
fi

# Fix securityContext placement in federation deployment
if grep -q "^      securityContext:" dist/chart/templates/federation/deployment.yaml; then
    sed -i.bak 's/^      securityContext:/          securityContext:/g' dist/chart/templates/federation/deployment.yaml
    rm -f dist/chart/templates/federation/deployment.yaml.bak
    echo "Fixed securityContext in federation deployment"
fi
echo -e "${GREEN}Helm chart fixes applied.${NC}"

# Step 7: Fix sample CR
echo ""
echo -e "${YELLOW}[7/11] Fixing sample CRs...${NC}"
cat > config/samples/federationHostAuth.yaml << 'YAML'
# Create this when the Origin Partner (Client)
# credentials are created in the host system
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Federation
metadata:
  namespace: federation-host
  annotations:
    opg.ewbi.nby.one/origin-mcc: "001"
    opg.ewbi.nby.one/origin-mncs: "001"
  labels:
    opg.ewbi.nby.one/federation-relation: host
    opg.ewbi.nby.one/origin-operator-name: guestName
    opg.ewbi.nby.one/origin-client-id: 3acde22c-d245-480d-b01e-24e38e01806d
    opg.ewbi.nby.one/origin-country-code: ES
  name: fed-e35f69d8-ae5a-456b-9f95-d950e4c03e8d
spec:
  offeredAvailabilityZones:
    - zoneId: "2a8fffaf-50de-4f93-8c6f-05f1c84b5a5f"
      geolocation: "40.4168,-3.7038"
      geographyDetails: "Madrid, Spain - Urban edge location"
  guestPartnerCredentials:
    clientId: 3acde22c-d245-480d-b01e-24e38e01806d
YAML
echo -e "${GREEN}Sample CRs fixed.${NC}"

# Step 8: Load images into Kind
echo ""
echo -e "${YELLOW}[8/11] Loading images into Kind cluster...${NC}"
kind load docker-image ghcr.io/neonephos-katalis/opg-ewbi-operator:neonephos --name "$CLUSTER_NAME"
kind load docker-image ghcr.io/neonephos-katalis/opg-ewbi-api:neonephos --name "$CLUSTER_NAME"
echo -e "${GREEN}Images loaded into Kind.${NC}"

# Step 9: Deploy federation-host
echo ""
echo -e "${YELLOW}[9/11] Deploying federation-host...${NC}"
kubectl create namespace federation-host || true
kubectl -n federation-host create secret docker-registry opg-registry-secret \
  --docker-server=ghcr.io \
  --docker-username=dummy \
  --docker-password=dummy 2>/dev/null || true

helm install federation-host dist/chart -n federation-host \
  --set crd.enable=true \
  --set federation.services.federation.nodePort=30080 \
  --set federation.image.pullPolicy=IfNotPresent \
  --set controllerManager.container.opgInsecureSkipVerify=true

echo "Waiting for federation-host pods..."
kubectl wait --for=condition=ready pod -l control-plane=controller-manager -n federation-host --timeout=120s
kubectl wait --for=condition=ready pod -l control-plane=federation-api -n federation-host --timeout=120s
echo -e "${GREEN}federation-host deployed.${NC}"

# Step 10: Deploy federation-guest
echo ""
echo -e "${YELLOW}[10/11] Deploying federation-guest...${NC}"
kubectl create namespace federation-guest || true
kubectl -n federation-guest create secret docker-registry opg-registry-secret \
  --docker-server=ghcr.io \
  --docker-username=dummy \
  --docker-password=dummy 2>/dev/null || true

helm install federation-guest dist/chart -n federation-guest \
  --set crd.enable=false \
  --set federation.services.federation.nodePort=30081 \
  --set federation.image.pullPolicy=IfNotPresent \
  --set controllerManager.container.opgInsecureSkipVerify=true

echo "Waiting for federation-guest pods..."
kubectl wait --for=condition=ready pod -l control-plane=controller-manager -n federation-guest --timeout=120s
kubectl wait --for=condition=ready pod -l control-plane=federation-api -n federation-guest --timeout=120s
echo -e "${GREEN}federation-guest deployed.${NC}"

# Step 11: Create federation resources
echo ""
echo -e "${YELLOW}[11/11] Creating federation resources...${NC}"
kubectl apply -f config/samples/federationHostAuth.yaml
sleep 2
kubectl apply -f config/samples/federationGuest.yaml
echo "Waiting for federation to establish..."
sleep 5
echo -e "${GREEN}Federation resources created.${NC}"

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Setup complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Run ./validate.sh to verify the setup."
echo ""
echo "To cleanup: ./cleanup.sh or 'kind delete cluster --name ${CLUSTER_NAME}'"
