# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Kubernetes Operator implementing a subset of the GSMA OPG (Operator Platform Group) East/WestBound Interface. It translates OPG interface requests to Kubernetes CRD resources and vice versa, enabling federated orchestrator deployment across clusters.

The operator can function in both HOST and GUEST roles for federation relationships.

## Build Commands

```bash
# Build the manager binary
make build

# Run controller locally (connects to cluster in kubeconfig)
make run

# Run the host API server locally
make run-host

# Build Docker images (requires ~/.netrc with ghcr.io credentials)
make docker-build              # Both controller and host
make docker-build-controller   # Controller only
make docker-build-host         # Host API only

# Push images to registry
make docker-push
```

## Testing

```bash
# Run unit tests (uses envtest for k8s API)
make test

# Run e2e tests (requires Kind cluster named 'kind')
make test-e2e

# Run a single test file
KUBEBUILDER_ASSETS="$(bin/setup-envtest use 1.31.0 --bin-dir bin -p path)" go test ./internal/controller/federation_controller_test.go -v
```

## Linting

```bash
make lint        # Run golangci-lint
make lint-fix    # Run golangci-lint with auto-fix
```

## Code Generation

```bash
# Generate CRDs, RBAC, and webhook manifests
make manifests

# Generate DeepCopy methods for API types
make generate

# Build Helm charts
make helm
```

## Deployment

```bash
# Install CRDs only
make install

# Deploy controller to cluster
make deploy

# Helm install (federation namespace)
helm install federation-manager dist/chart -n federation \
  --set federation.services.federation.nodePort=30080 \
  --set crd.enable=true
```

## Architecture

### API Types (`api/v1beta1/`)

Six CRD types representing OPG resources:
- **Federation**: Establishes HOST/GUEST relationships between operator platforms. Contains partner credentials, offered/accepted availability zones
- **AvailabilityZone**: Geographic edge locations offered in federations
- **File**: File resources transferred between federated partners
- **Artefact**: Packaged application artifacts (references Files)
- **Application**: Application definitions
- **ApplicationInstance**: Running instances of applications

### Controllers (`internal/controller/`)

Each CRD has a reconciler following the pattern:
1. Add finalizer on create
2. For GUEST resources (labeled `opg.ewbi.nby.one/federation-relation: guest`): make OPG API calls to HOST
3. For HOST resources: mark as ready (external system handles actual deployment)
4. On delete: clean up external resources via OPG API before removing finalizer

### OPG Client (`internal/opg/`)

`OPGClientsMap` maintains HTTP clients per federation, keyed by federation ID. Clients are lazily created with X-Client-ID header injection. The API client is generated from OpenAPI specs in the `opg-ewbi-api` dependency.

### Host API Server (`cmd/host/`)

Simple HTTP server for testing that lists/creates ApplicationInstance CRs. Not the production API.

### Key Labels

- `opg.ewbi.nby.one/federation-context-id`: Links resources to their federation
- `opg.ewbi.nby.one/federation-relation`: `guest` or `host`
- `opg.ewbi.nby.one/id`: External ID for cross-platform reference

## Platform Configuration

The operator supports both ARM64 and AMD64. Modify these for AMD64:
- `Makefile` lines 1-3: Change IMG, HOSTIMG suffixes to `-amd`, PLATFORM to `linux/amd64`
- `dist/chart/values.yaml`: Update image repositories

## Dependencies

- `github.com/neonephos-katalis/opg-ewbi-api`: Generated OPG API client and models
- `sigs.k8s.io/controller-runtime`: Kubernetes controller framework
- `kubebuilder`: Scaffolding (v3 style layout)
