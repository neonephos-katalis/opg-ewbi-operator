# Local Development Instructions

This guide provides clear steps to run the OPG EWBI Federation Manager locally in a Kind cluster.

## Understanding the Components

There are **two components** that work together:

| Component | Repository | Purpose |
|-----------|------------|---------|
| **opg-ewbi-operator** | This repo | Kubernetes controller that watches CRDs and makes OPG API calls |
| **opg-ewbi-api** | `github.com/neonephos-katalis/opg-ewbi-api` | REST API implementing the OPG East/WestBound Interface |

**How they interact:**
- GUEST resources (labeled `opg.ewbi.nby.one/federation-relation: guest`) trigger HTTP calls to a HOST's API
- HOST resources are just marked ready (external orchestrators like Okto handle actual deployment)

---

## Prerequisites

```bash
# Required tools
go version      # 1.22+
docker version  # 17.03+
kubectl version # 1.11.3+
helm version    # 3.x+
kind version    # 0.20+
```

---

## Quick Start (Full Setup)

### Step 1: Create Kind Cluster

```bash
# Create a new Kind cluster for testing
kind create cluster --name federation-test

# Verify cluster is running
kubectl cluster-info
```

### Step 2: Create Empty ~/.netrc

The Dockerfile expects a `.netrc` file for the secret mount, but since both repos are public, it can be empty:

```bash
touch ~/.netrc
chmod 600 ~/.netrc
```

### Step 3: Clone Both Repositories

```bash
# Clone this repo (if not already)
git clone https://github.com/neonephos-katalis/opg-ewbi-operator.git
cd opg-ewbi-operator

# Clone the API repo alongside
cd ..
git clone https://github.com/neonephos-katalis/opg-ewbi-api.git
cd opg-ewbi-operator
```

### Step 4: Build Operator Image

```bash
make docker-build-controller
```

### Step 5: Fix and Build API Image

The `opg-ewbi-api` repo has an outdated dependency. Fix it:

```bash
cd ../opg-ewbi-api

# Add replace directive to use local operator
cat >> go.mod << 'EOF'

replace github.com/neonephos-katalis/opg-ewbi-operator => ../opg-ewbi-operator
EOF

# Vendor dependencies
go mod vendor

# Modify Dockerfile to use vendored deps
cat > Dockerfile << 'EOF'
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
EOF

# Build the image
docker build --platform=linux/arm64 -t ghcr.io/neonephos-katalis/opg-ewbi-api:neonephos .

cd ../opg-ewbi-operator
```

> **Note for AMD64:** Change `--platform=linux/arm64` to `--platform=linux/amd64`

### Step 6: Fix Helm Chart Issues

The Helm chart has bugs that need fixing:

```bash
# Fix hardcoded namespace in role.yaml
sed -i '' 's/namespace: foo/namespace: {{ .Release.Namespace }}/g' dist/chart/templates/rbac/role.yaml

# Fix securityContext placement in federation deployment
sed -i '' 's/^      securityContext:/          securityContext:/g' dist/chart/templates/federation/deployment.yaml
```

### Step 7: Fix Sample CR

The sample federation CR uses outdated types:

```bash
cat > config/samples/federationHostAuth.yaml << 'EOF'
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
EOF
```

### Step 8: Load Images into Kind

```bash
kind load docker-image ghcr.io/neonephos-katalis/opg-ewbi-operator:neonephos --name federation-test
kind load docker-image ghcr.io/neonephos-katalis/opg-ewbi-api:neonephos --name federation-test
```

### Step 9: Deploy Federation Host

```bash
# Create namespace
kubectl create namespace federation-host

# Create dummy registry secret (images already loaded, but Helm requires it)
kubectl -n federation-host create secret docker-registry opg-registry-secret \
  --docker-server=ghcr.io \
  --docker-username=dummy \
  --docker-password=dummy

# Install with Helm (includes CRDs)
helm install federation-host dist/chart -n federation-host \
  --set crd.enable=true \
  --set federation.services.federation.nodePort=30080 \
  --set federation.image.pullPolicy=IfNotPresent \
  --set controllerManager.container.opgInsecureSkipVerify=true

# Verify pods are running
kubectl get pods -n federation-host
```

### Step 10: Deploy Federation Guest

```bash
# Create namespace
kubectl create namespace federation-guest

# Create dummy registry secret
kubectl -n federation-guest create secret docker-registry opg-registry-secret \
  --docker-server=ghcr.io \
  --docker-username=dummy \
  --docker-password=dummy

# Install with Helm (CRDs already installed)
helm install federation-guest dist/chart -n federation-guest \
  --set crd.enable=false \
  --set federation.services.federation.nodePort=30081 \
  --set federation.image.pullPolicy=IfNotPresent \
  --set controllerManager.container.opgInsecureSkipVerify=true

# Verify pods are running
kubectl get pods -n federation-guest
```

### Step 11: Test Federation Flow

```bash
# 1. Create HOST federation (represents clientID registration)
kubectl apply -f config/samples/federationHostAuth.yaml

# 2. Create GUEST federation (calls HOST's API)
kubectl apply -f config/samples/federationGuest.yaml

# 3. Check federation status
kubectl get federations -A

# 4. Verify GUEST received federation context from HOST
kubectl get federation fed-2dae064c-28cc-456e-8b0a-dd67bab7d8f7 -n federation-guest -o yaml
```

**Expected output in GUEST federation status:**
```yaml
status:
  federationContextId: <uuid-from-host>
  state: Available
  offeredAvailabilityZones:
    - zoneId: "2a8fffaf-50de-4f93-8c6f-05f1c84b5a5f"
      geolocation: "40.4168,-3.7038"
      geographyDetails: "Madrid, Spain - Urban edge location"
```

---

## Cleanup

### Delete Federation Resources

```bash
kubectl delete -f config/samples/federationGuest.yaml
kubectl delete -f config/samples/federationHostAuth.yaml
```

### Uninstall Helm Releases

```bash
helm uninstall federation-guest -n federation-guest
helm uninstall federation-host -n federation-host
```

### Delete Kind Cluster

```bash
kind delete cluster --name federation-test
```

---

## Alternative: Run Operator Locally (Development Mode)

For quick iteration without Docker:

```bash
# Install CRDs
make install

# Create namespace
kubectl create namespace federation

# Run controller locally
NAMESPACE=federation make run
```

> **Note:** GUEST resources will fail in this mode (no HOST API to call).

---

## Troubleshooting

### Pods stuck in ImagePullBackOff

Images weren't loaded into Kind:
```bash
kind load docker-image IMAGE_NAME --name federation-test
```

### "namespace foo not found" error

Helm chart has hardcoded namespace. Fix with:
```bash
sed -i '' 's/namespace: foo/namespace: {{ .Release.Namespace }}/g' dist/chart/templates/rbac/role.yaml
```

### securityContext schema error

The federation deployment template has securityContext at wrong level:
```bash
sed -i '' 's/^      securityContext:/          securityContext:/g' dist/chart/templates/federation/deployment.yaml
```

### GUEST federation stuck / not getting federationContextId

Check controller logs:
```bash
kubectl logs -n federation-guest -l control-plane=controller-manager -f
```

Check HOST API logs:
```bash
kubectl logs -n federation-host -l control-plane=federation-api -f
```

### offeredAvailabilityZones type error

The sample CRs use old `[]string` format. Update to use `[]ZoneDetails`:
```yaml
offeredAvailabilityZones:
  - zoneId: "zone-uuid"
    geolocation: "lat,long"
    geographyDetails: "Description"
```

---

## Known Issues in Upstream Code

| Issue | Location | Fix |
|-------|----------|-----|
| Outdated operator dependency | `opg-ewbi-api/go.mod` | Add `replace` directive |
| Hardcoded `namespace: foo` | `dist/chart/templates/rbac/role.yaml` | Use `{{ .Release.Namespace }}` |
| securityContext at pod level | `dist/chart/templates/federation/deployment.yaml` | Move inside container spec |
| Old `offeredAvailabilityZones` type | `config/samples/federationHostAuth.yaml` | Use `ZoneDetails` objects |
| Missing Kind load step | README.md | Add `kind load docker-image` |

---

## Quick Reference

| Task | Command |
|------|---------|
| Create Kind cluster | `kind create cluster --name federation-test` |
| Delete Kind cluster | `kind delete cluster --name federation-test` |
| Build operator image | `make docker-build-controller` |
| Load image to Kind | `kind load docker-image IMAGE --name federation-test` |
| Install CRDs only | `make install` |
| Run controller locally | `NAMESPACE=ns make run` |
| View all federations | `kubectl get federations -A` |
| Controller logs | `kubectl logs -n NS -l control-plane=controller-manager -f` |
| API logs | `kubectl logs -n NS -l control-plane=federation-api -f` |

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        GUEST Cluster                             │
│  ┌──────────────┐     ┌──────────────┐                          │
│  │  Federation  │────▶│   Operator   │──────HTTP────┐           │
│  │  CR (guest)  │     │ (controller) │              │           │
│  └──────────────┘     └──────────────┘              │           │
└─────────────────────────────────────────────────────│───────────┘
                                                      │
                                                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                         HOST Cluster                             │
│                       ┌──────────────┐                          │
│                       │  opg-ewbi-   │                          │
│              ────────▶│     api      │                          │
│                       └──────┬───────┘                          │
│                              │ creates                          │
│                              ▼                                  │
│  ┌──────────────┐     ┌──────────────┐                          │
│  │  Federation  │◀────│   Operator   │                          │
│  │  CR (host)   │     │ (controller) │                          │
│  └──────────────┘     └──────────────┘                          │
└─────────────────────────────────────────────────────────────────┘
```
