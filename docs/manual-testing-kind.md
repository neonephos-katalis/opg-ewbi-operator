# Manual Testing on Kind (Full Guest ↔ Host Flow)

End-to-end guide for testing the complete federation flow on a single Kind cluster with two namespaces simulating a Guest OP and a Host OP.

## Architecture

```
kind-federation cluster
├── namespace: katalis-dev-host    ← simulates the Host OP
│   ├── operator pod            ← reconciles host-mode CRs (marks them Ready)
│   ├── federation-api pod      ← receives inbound EWBI HTTP calls
│   ├── Federation CR           ← pre-provisioned (empty initialDate, origin-client-id label)
│
└── namespace: katalis-dev-guest   ← simulates the Guest OP
    ├── operator pod            ← reconciles guest-mode CRs (makes outbound calls to host API)
    └── [federation-api pod]    ← optional: only needed to receive host→guest callbacks
```

**Flow initiated by the guest operator:**
1. Admin creates a `Federation` CR in `katalis-dev-guest` with `federation-relation=guest`
2. Guest operator calls `POST /federation` on the host API → host CR updated → returns `federationContextId` + 1 AZ
3. Guest operator calls `ZoneSubscribe` automatically on next reconcile
4. Admin creates `File`, `Artefact`, `Application`, `ApplicationInstance` CRs in `katalis-dev-guest`
5. Guest operator calls the corresponding host API endpoints for each CR
6. Host operator reconciles each host-mode CR and marks it `Ready`

## Prerequisites

- Go 1.24+, Docker, kind, kubectl, helm, kubebuilder
- A Kind cluster named `federation`:
  ```sh
  kind create cluster --name federation
  kubectl config use-context kind-federation
  ```

## 1. Build Everything

```sh
# Regenerate CRDs, deepcopy, and OpenAPI code from source
make manifests generate apigen

# Build the Helm chart (includes CRDs)
make base-chart

# Build both Docker images (operator + API)
# The Makefile defaults to --platform=linux/amd64.
# On Apple Silicon (M1/M2/M3) the Kind node runs linux/arm64 — override with PLATFORM:
#   make docker-build docker-build-api PLATFORM=linux/arm64 IMG=opg-ewbi-operator:dev IMG_API=opg-ewbi-api:dev
# On Intel Macs or Linux amd64 (default):
make docker-build docker-build-api IMG=opg-ewbi-operator:dev IMG_API=opg-ewbi-api:dev

# Load images into the kind cluster
kind load docker-image opg-ewbi-operator:dev --name federation
kind load docker-image opg-ewbi-api:dev --name federation
```

## 2. Deploy Host Namespace

```sh
helm upgrade --install federation-host dist/chart \
  --namespace katalis-dev-host --create-namespace \
  --set controllerManager.container.image.repository=opg-ewbi-operator \
  --set controllerManager.container.image.tag=dev \
  --set controllerManager.container.image.pullPolicy=Never \
  --set federation.image.repository=opg-ewbi-api \
  --set federation.image.tag=dev \
  --set federation.image.pullPolicy=Never \
  --set crd.enable=true \
  --set controllerManager.container.opgInsecureSkipVerify=true

kubectl -n katalis-dev-host get pods
```

> **Note:** `pullPolicy=Never` is required for Kind because images are loaded directly into the node, not pulled from a registry.

## 3. Deploy Guest Namespace

The guest only needs the operator (it makes outbound calls to the host API). Deploy without API, or with API if you want to test host→guest callbacks.

```sh
helm upgrade --install federation-guest dist/chart \
  --namespace katalis-dev-guest --create-namespace \
  --set controllerManager.container.image.repository=opg-ewbi-operator \
  --set controllerManager.container.image.tag=dev \
  --set controllerManager.container.image.pullPolicy=Never \
  --set federation.image.repository=opg-ewbi-api \
  --set federation.image.tag=dev \
  --set federation.image.pullPolicy=Never \
  --set crd.enable=false \
  --set controllerManager.container.opgInsecureSkipVerify=true

kubectl -n katalis-dev-guest get pods
```

## 4. Pre-Provision Host Resources

The host API's `CreateFederation` handler **does not create** a new Federation CR — it **looks up an existing one** by matching label `opg.ewbi.nby.one/origin-client-id` against the incoming `X-Client-ID` header. A cluster admin must pre-provision it.

> **Important:** The host Federation CR must offer **at least 1** AvailabilityZone in `spec.offeredAvailabilityZones`. The guest operator's `handleAcceptExternalAZ` picks `offeredAvailabilityZones[0]` and calls `ZoneSubscribe` with it. With 0 offered zones the subscription is skipped.

Apply the pre-provisioned host Federation CR (the sample already sets `namespace: katalis-dev-host`):

```sh
kubectl apply -f config/samples/federationHostAuth.yaml
```

> **Note:** You do **not** need to create a matching `AvailabilityZone` CR. The value in `spec.offeredAvailabilityZones` is just a string ID — `CreateFederation` and `ZoneSubscribe` both read it directly from the Federation CR spec without doing any CR lookup. An `AvailabilityZone` CR is only needed if you later call `GET /{fcid}/zones/{zoneId}` (`GetZoneData`), which is not part of this flow.

The sample sets `opg.ewbi.nby.one/origin-client-id: 3acde22c-d245-480d-b01e-24e38e01806d`. The guest Federation CR's `spec.guestPartnerCredentials.clientId` must match this value exactly.

Note: the current implementation deterministically derives the `federationContextId` from the `clientId` (UUID V5). All sample YAML files already contain the resulting FederationContextId value `4d559f1b-f008-58c2-a2f8-0596892a0f7a` — if you use a different `clientId`, read the actual FCID from the guest Federation CR's `status.federationContextId` after establishment and update the sample files accordingly.

## 5. Create the Guest Federation CR

This CR triggers the guest operator to call `POST /federation` on the host API.

Key fields in `config/samples/federationGuest.yaml`:
- `spec.guestPartnerCredentials.tokenUrl` — the **base URL** of the host API (no path suffix). The generated client appends route paths automatically. The sample file already sets `http://nearbyone-federation-api.katalis-dev-host.svc.cluster.local:8080`, which is correct for this Kind setup.
- `spec.guestPartnerCredentials.clientId` — sent as `X-Client-ID` header to the host; **must match** the `origin-client-id` label on the host's pre-provisioned Federation CR (`3acde22c-d245-480d-b01e-24e38e01806d`).
- `metadata.labels["opg.ewbi.nby.one/id"]` — the `origOPFederationId` business ID sent in the request body.

```sh
kubectl apply -f config/samples/federationGuest.yaml
```

### Verify Federation Establishment

```sh
# Guest operator calls the host API and updates the guest CR status
kubectl -n katalis-dev-guest get federation fed-2dae064c-28cc-456e-8b0a-dd67bab7d8f7 -o yaml
# Look for: status.federationContextId, status.state: AVAILABLE, status.offeredAvailabilityZones

# On next reconcile, guest operator calls ZoneSubscribe and updates spec
# Look for: spec.acceptedAvailabilityZones

# Host-side: the Federation CR should have been updated with initialDate, partner info and have opg.ewbi.nby.one/federation-context-id label that matches the guest CR's status.federationContextId
kubectl -n katalis-dev-host get federation fed-e35f69d8-ae5a-456b-9f95-d950e4c03e8d -o yaml
```

Get the federation context ID for subsequent steps:
```sh
FCID=$(kubectl -n katalis-dev-guest get federation fed-2dae064c-28cc-456e-8b0a-dd67bab7d8f7 \
  -o jsonpath='{.status.federationContextId}')
echo "FederationContextId: $FCID"
```

## 6. Upload a File (Guest → Host)

The sample file already contains the `federation-context-id` `4d559f1b-f008-58c2-a2f8-0596892a0f7a`:

```sh
kubectl apply -f config/samples/fileGuest.yaml

# Guest operator calls POST /{fcid}/files on host API
kubectl -n katalis-dev-guest get file file-2dae064c-28cc-456e-8b0a-dd67bab7d8f7 -o yaml
# Verify: status.state = PENDING
kubectl -n katalis-dev-host get files  # File CR should exist in host namespace
```

## 7. Upload an Artefact (Guest → Host)

The artefact sample references `file-2dae064c-28cc-456e-8b0a-dd67bab7d8f7` (the file from step 6):

```sh
kubectl apply -f config/samples/artefactGuest.yaml

kubectl -n katalis-dev-guest get artefact artefact-2dae064c-28cc-456e-8b0a-dd67bab7d8f7 -o yaml
# Verify: status.state = PENDING
```

## 8. Onboard an Application (Guest → Host)

The application sample references `artefact-2dae064c-28cc-456e-8b0a-dd67bab7d8f7` (the artefact from step 7):

```sh
kubectl apply -f config/samples/appGuest.yaml

kubectl -n katalis-dev-guest get application app-2dae064c-28cc-456e-8b0a-dd67bab7d8f7 -o yaml
# Verify: status.state = PENDING
```

## 9. Deploy an Application Instance (Guest → Host)

The application instance sample references `app-2dae064c-28cc-456e-8b0a-dd67bab7d8f7` (the application from step 8) and zone `zone-es-madrid-001`:

```sh
kubectl apply -f config/samples/appInstGuest.yaml

kubectl -n katalis-dev-guest get applicationinstance app-inst-2dae064c-28cc-456e-8b0a-dd67bab7d8f7 -o yaml
# Verify: status.state = PENDING
```

## 10. Verify the Full State

```sh
# Guest-side CRs (guest operator reconciled these, made outbound calls)
kubectl -n katalis-dev-guest get federations,files,artefacts,applications,applicationinstances

# Host-side CRs (created by host API upon receiving guest operator's requests)
kubectl -n katalis-dev-host get federations,files,artefacts,applications,applicationinstances

# Host-side Federation should be AVAILABLE; child CRs should be PENDING
kubectl -n katalis-dev-host get applications -o yaml | grep state
```

## 11. Optional: Test the Host API Directly via curl

Port-forward to the host API to simulate raw API calls (bypassing the guest operator):

```sh
kubectl -n katalis-dev-host port-forward svc/nearbyone-federation-api 8080:8080 &
```

Then use the curl examples from the previous host-only flow. The API base URL is `http://localhost:8080/operatorplatform/federation/v1`.

> The `X-Client-ID` header must match the `origin-client-id` label of the pre-provisioned host Federation CR.
> Credentials validation is currently hardcoded to `skipCredentialsValidation=true` in `cmd/api/main.go`, so any value is accepted.

## 12. Cleanup

```sh
# Delete all guest CRs (triggers guest operator's finalizer + deletion calls to host API)
kubectl -n katalis-dev-guest delete applicationinstances,applications,artefacts,files,federations --all

# Wait for guest cleanup to complete, then uninstall
helm uninstall federation-guest -n katalis-dev-guest
helm uninstall federation-host -n katalis-dev-host

# Delete installed CRDs (not automatically removed by Helm because they are shared between guest and host)
kubectl delete crd federations.opg.ewbi.nby.one files.opg.ewbi.nby.one artefacts.opg.ewbi.nby.one applications.opg.ewbi.nby.one applicationinstances.opg.ewbi.nby.one availabilityzones.opg.ewbi.nby.one

# Delete kind cluster
kind delete cluster --name federation
```

## Next Steps

To test the Host → Guest callback flow (patching host-side CRs and verifying propagation to the guest), see [manual-callbacks-testing-kind.md](manual-callbacks-testing-kind.md).

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| Guest Federation stuck in `NOT_AVAILABLE`, no outbound call | `spec.guestPartnerCredentials.tokenUrl` wrong or missing | Must be the host API base URL **without** any path suffix (e.g. `http://nearbyone-federation-api.katalis-dev-host.svc.cluster.local:8080`) — the client appends route paths automatically |
| Host API returns 404 on `CreateFederation` | No pre-provisioned Federation CR with matching `origin-client-id` label | Create the host Federation CR (step 4) |
| Host API returns 409 on `CreateFederation` | Host Federation already has `spec.initialDate` set (already established) | Delete and recreate the host Federation CR |
| Guest `ZoneSubscribe` never called | Host offered 0 zones | Host's `spec.offeredAvailabilityZones` must have **at least 1** entry (the guest subscribes to the first) |
| `federation-context-id` label mismatch on child CRs | Used wrong/placeholder FCID | Set label from `Federation.Status.FederationContextId` |
| Child CRs return 500 `"application not found"` | Dependency not yet Ready on host | Check order: File → Artefact → Application → ApplicationInstance |
| Pods in `CrashLoopBackOff` | Missing env vars or RBAC | Check `kubectl -n <ns> logs <pod>` |
| Image pull errors in kind | Images not loaded | Re-run `kind load docker-image` and set `imagePullPolicy=Never` in helm values |
| `ErrImagePull` / `exec format error` on Kind | Image built for wrong CPU architecture | On Apple Silicon, build with `PLATFORM=linux/arm64` (Makefile defaults to amd64) |
| Host API returns 400 `"doesn't match schema"` | CR field values don't match swagger enum constraints | Check swagger.yaml for valid enum values (e.g., `multiUserClients`, `resourceConsumption`) |
| Stale CRDs from previous deployment | Old CRDs without Helm labels block `helm install` | Delete CRDs manually: `kubectl delete crd <name>`, patch stuck finalizers if needed |
| Guest operator caches wrong API URL | OPG client cached with old `tokenUrl` | Restart the guest operator pod to clear the OPG client cache |
