# OPG EWBI - Understanding the Federation Manager

This document explains what the OPG EWBI Federation Manager does, what each resource type means, and provides realistic usage examples.

## Table of Contents

1. [What is OPG EWBI?](#what-is-opg-ewbi)
2. [Why Kubernetes? (And Why It's Optional)](#why-kubernetes-and-why-its-optional)
3. [Resource Types Explained](#resource-types-explained)
4. [Real-World Example: EdgeStream CDN](#real-world-example-edgestream-cdn)
5. [Simpler Example: IoT Gateway](#simpler-example-iot-gateway)

---

## What is OPG EWBI?

**OPG** = Operator Platform Group (a GSMA initiative)
**EWBI** = East/WestBound Interface (the API specification)

The [GSMA OPG East/WestBound Interface](https://www.gsma.com/solutions-and-impact/technologies/networks/gsma_resources/gsma-operator-platform-group-east-westbound-interface-apis-version-5-0/) is a **REST API specification** that defines how telecom operators federate their edge computing resources.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     WHAT THE GSMA OPG SPEC DEFINES                           │
│                                                                              │
│   REST APIs for edge computing federation:                                   │
│                                                                              │
│   POST /federation                    → Establish trust between operators   │
│   POST /artefact-management/files     → Register deployable images          │
│   POST /artefact-management/artefacts → Create deployment packages          │
│   POST /app-lcm/applications          → Register applications               │
│   POST /app-lcm/app-instances         → Request deployment to edge zone     │
│   DELETE /...                         → Remove resources                    │
│                                                                              │
│   That's it. The spec defines HTTP APIs, not Kubernetes resources.          │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### What This Operator Does

This operator is a **Kubernetes implementation** of the OPG EWBI spec. It translates Kubernetes Custom Resources (CRs) into OPG API calls.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    WHAT THE OPERATOR DOES                                    │
│                                                                              │
│   It's a translation layer:                                                  │
│                                                                              │
│   ┌──────────────────┐                      ┌──────────────────┐            │
│   │  Kubernetes CR   │                      │   HTTP API Call  │            │
│   │                  │    operator          │                  │            │
│   │  kind: File      │ ──────────────────►  │  POST /files     │            │
│   │  spec:           │    translates        │  {               │            │
│   │    fileName: x   │                      │    "fileName": x │            │
│   │    fileType: y   │                      │    "fileType": y │            │
│   └──────────────────┘                      │  }               │            │
│                                             └──────────────────┘            │
│                                                                              │
│   Plus it handles:                                                           │
│   • Retry on failure (reconciliation loop)                                  │
│   • Update CR status with API response                                      │
│   • Cleanup on deletion (finalizers → DELETE API call)                      │
│   • OAuth token management                                                  │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### What This Operator Does NOT Do

```
┌────────────────────────────────────────────────────────────────────────────────┐
│                           WHAT THIS OPERATOR DOES NOT DO                        │
│                                                                                 │
│   ❌ Create VMs                                                                 │
│   ❌ Deploy containers                                                          │
│   ❌ Provision infrastructure                                                   │
│   ❌ Manage networking                                                          │
│                                                                                 │
│   It only syncs METADATA (deployment intent) between operator platforms.       │
│                                                                                 │
│   ANOTHER SYSTEM (like NearbyOne Okto, OpenStack, or any orchestrator)         │
│   watches the synced resources and does the actual deployment work.            │
└────────────────────────────────────────────────────────────────────────────────┘
```

---

## Why Kubernetes? (And Why It's Optional)

### The Short Answer

**Kubernetes is NOT required by the GSMA OPG spec.** This operator is one implementation choice.

### You Can Just Call the APIs Directly

```bash
# No Kubernetes needed - just HTTP calls

# Establish federation
curl -X POST https://opg-ewbi.telefonica.com/federation \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"originOPId": {"countryCode": "DE", "mcc": "262"}}'

# Upload file reference
curl -X POST https://opg-ewbi.telefonica.com/artefact-management/files \
  -H "X-Federation-Context-Id: ctx-123..." \
  -d '{"fileName": "my-app", "fileType": "DOCKER"}'

# Create application instance
curl -X POST https://opg-ewbi.telefonica.com/app-lcm/app-instances \
  -d '{"appId": "app-123", "zoneInfo": {"zoneId": "madrid-edge"}}'
```

This works without any Kubernetes involvement.

### Why NearbyOne Uses Kubernetes

NearbyOne chose Kubernetes because of their existing ecosystem:

| Reason | Explanation |
|--------|-------------|
| **Okto Integration** | Their orchestrator (Okto) already watches Kubernetes CRs to trigger deployments |
| **GitOps** | Teams can manage federation resources via ArgoCD/Flux |
| **Reconciliation** | Operator pattern automatically retries failed API calls |
| **RBAC** | Kubernetes native access control for federation resources |
| **Familiarity** | Their teams already work with Kubernetes |

### When You Would NOT Use Kubernetes

| Scenario | Alternative Implementation |
|----------|---------------------------|
| Legacy telecom (VMware/OpenStack) | Python/Java OPG API client → PostgreSQL → OpenStack Heat |
| Simple app provider | Shell scripts calling OPG APIs |
| Cloud provider integration | Lambda/Azure Functions → DynamoDB → native cloud services |
| Existing orchestrator | Direct API integration with your existing tooling |

---

## Resource Types Explained

The OPG EWBI spec defines six resource types. Here's what each one is and does:

### Federation

**What it is:** A trust relationship between two operator platforms.

**What it does:**
- Establishes mutual authentication (OAuth2 credentials)
- Defines which edge zones (AvailabilityZones) the HOST offers to the GUEST
- Provides a `federationContextId` that links all subsequent resources

**GSMA Spec Context:** The federation is the foundational relationship. Before any applications can be deployed across platforms, operators must first establish federation. This involves exchanging credentials and the HOST advertising which edge locations are available.

**Key Fields:**

| Field | Purpose |
|-------|---------|
| `spec.originOP` | Identity of the originating operator (country code, MCC/MNC) |
| `spec.partner.statusLink` | URL of the partner's OPG API endpoint |
| `spec.partner.callbackCredentials` | OAuth2 credentials for authenticating to the partner |
| `spec.offeredAvailabilityZones` | (HOST) Edge zones offered to the partner |
| `spec.acceptedAvailabilityZones` | (GUEST) Which offered zones the guest accepts |
| `status.federationContextId` | Unique ID assigned by HOST, used in all subsequent API calls |
| `status.state` | Current state: `Available`, `Locked`, `Failed`, etc. |

**Labels:**

| Label | Purpose |
|-------|---------|
| `opg.ewbi.nby.one/federation-relation` | `guest` or `host` - determines operator behavior |
| `opg.ewbi.nby.one/federation-context-id` | Links resources to their federation |

```yaml
# Example: GUEST establishing federation with a HOST
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Federation
metadata:
  name: my-federation
  labels:
    opg.ewbi.nby.one/federation-relation: guest  # This triggers API calls to HOST
spec:
  originOP:
    countryCode: "DE"
    mobileNetworkCodes:
      mcc: "262"
      mncs: ["01"]
  partner:
    statusLink: "https://opg-ewbi.telefonica.com"  # HOST's API endpoint
    callbackCredentials:
      clientId: "my-client-id"
      tokenUrl: "https://idp.telefonica.com/token"
  acceptedAvailabilityZones:
    - "zone-es-madrid-central"
```

---

### AvailabilityZone

**What it is:** A physical edge location where applications can be deployed.

**What it does:**
- Represents a geographic edge site (datacenter, cell tower, etc.)
- Advertises location coordinates and characteristics
- HOST creates these; GUEST receives them via federation

**GSMA Spec Context:** Availability Zones represent the physical edge infrastructure. They have geographic coordinates, descriptions of the locality type (urban, rural, industrial), and resource characteristics. The HOST operator creates these to represent their edge sites, and they're offered to GUEST operators through the federation relationship.

**Key Fields:**

| Field | Purpose |
|-------|---------|
| `spec.zoneId` | Human-readable zone identifier |
| `spec.geolocation` | Latitude,Longitude (e.g., "40.4168,-3.7038") |
| `spec.geographyDetails` | Human-readable description of the location |
| `status.flavoursSupported` | Resource tiers available (like EC2 instance types) |
| `status.latency` | Expected network latency characteristics |

```yaml
# Example: HOST defines an edge zone
apiVersion: opg.ewbi.nby.one/v1beta1
kind: AvailabilityZone
metadata:
  name: zone-es-madrid-central
spec:
  zoneId: "zone-es-madrid-central"
  geolocation: "40.4168,-3.7038"
  geographyDetails: "Madrid Central - Telefonica Tower, 5G connected, urban location"
status:
  flavoursSupported: ["edge-small", "edge-medium", "edge-large"]
  latency: "5ms"
```

---

### File

**What it is:** A reference to a deployable image (container or VM).

**What it does:**
- Points to where the image can be pulled from (registry URL)
- Specifies the image type (Docker, QCOW2, OCI, etc.)
- Defines OS and architecture requirements
- Does NOT contain the actual image bytes - just metadata

**GSMA Spec Context:** A File represents any deployable artifact - container images, VM disk images (QCOW2), or other formats. The File resource tells the HOST where to find the image and what its requirements are. Multiple Artefacts can reference the same File.

**Supported File Types:**
- `DOCKER` - Container image (most common)
- `QCOW2` - VM disk image
- `OCI` - OCI container image
- `RAW` - Raw disk image
- `VMDK` - VMware disk

**Key Fields:**

| Field | Purpose |
|-------|---------|
| `spec.appProviderId` | Who owns this file |
| `spec.fileName` | Human-readable name |
| `spec.fileType` | `DOCKER`, `QCOW2`, `OCI`, etc. |
| `spec.fileVersion` | Version string |
| `spec.repoLocation.url` | Where to pull the image from |
| `spec.repoLocation.type` | `public` or `private` |
| `spec.image.instructionSetArchitecture` | `ISA_X86_64`, `ISA_ARM64`, etc. |
| `spec.image.os` | Operating system requirements |

```yaml
# Example: Container image reference
apiVersion: opg.ewbi.nby.one/v1beta1
kind: File
metadata:
  name: my-app-image
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/federation-context-id: "ctx-123..."
spec:
  appProviderId: "my-company"
  fileName: "my-app"
  fileType: DOCKER
  fileVersion: "2.0.0"
  repoLocation:
    url: "ghcr.io/my-company/my-app"
    type: private
  image:
    instructionSetArchitecture: ISA_X86_64
    os:
      distribution: UBUNTU
      version: OS_VERSION_UBUNTU_2204_LTS
```

```yaml
# Example: VM image reference
apiVersion: opg.ewbi.nby.one/v1beta1
kind: File
metadata:
  name: my-vm-image
spec:
  fileName: "ubuntu-server"
  fileType: QCOW2
  fileVersion: "22.04"
  repoLocation:
    url: "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
    type: public
```

---

### Artefact

**What it is:** A deployment package that bundles Files with resource requirements.

**What it does:**
- References one or more Files (images)
- Specifies compute resources (CPU, memory, disk)
- Defines network interfaces (ports, protocols)
- Sets scaling parameters (replicas, restart policy)
- Like a Helm chart or docker-compose file - describes HOW to deploy

**GSMA Spec Context:** An Artefact is a deployment descriptor. It can be thought of as similar to a Helm chart, Kubernetes Deployment, or docker-compose file. It bundles the image references (Files) with all the information needed to deploy them: resource requirements, networking, scaling, and startup commands. Multiple Applications can reference the same Artefact.

**Key Fields:**

| Field | Purpose |
|-------|---------|
| `spec.appProviderId` | Who owns this artefact |
| `spec.artefactName` | Human-readable name |
| `spec.artefactVersion` | Version string |
| `spec.virtType` | `CONTAINER` or `VM` |
| `spec.descriptorType` | `HELM`, `DOCKER_COMPOSE`, `TOSCA`, etc. |
| `spec.componentSpec` | List of components to deploy |
| `spec.componentSpec[].images` | References to File resources |
| `spec.componentSpec[].numOfInstances` | Number of replicas |
| `spec.componentSpec[].computeResourceProfile` | CPU, memory, disk requirements |
| `spec.componentSpec[].exposedInterfaces` | Ports and protocols |
| `spec.componentSpec[].commandLineParams` | Entrypoint and arguments |

```yaml
# Example: Deployment package for a web application
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Artefact
metadata:
  name: my-app-bundle
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/federation-context-id: "ctx-123..."
spec:
  appProviderId: "my-company"
  artefactName: "my-app-bundle"
  artefactVersion: "2.0.0"
  virtType: "CONTAINER"
  descriptorType: "HELM"
  componentSpec:
    - name: "web-server"
      images:
        - "my-app-image"          # References the File resource
      numOfInstances: 3           # 3 replicas
      restartPolicy: "Always"
      computeResourceProfile:
        numCPU: "2"
        memory: 4096              # 4GB RAM
        cpuArchType: "x86_64"
        cpuExclusivity: false     # Can share CPU with other workloads
      exposedInterfaces:
        - interfaceId: "http"
          port: 80
          protocol: "TCP"
          visibilityType: "EXTERNAL"
        - interfaceId: "https"
          port: 443
          protocol: "TCP"
          visibilityType: "EXTERNAL"
        - interfaceId: "metrics"
          port: 9090
          protocol: "TCP"
          visibilityType: "INTERNAL"
      commandLineParams:
        command: ["/app/server"]
        args: ["--port=80", "--log-level=info"]
```

---

### Application

**What it is:** A business-level application definition with QoS requirements.

**What it does:**
- References one or more Artefacts (deployment packages)
- Defines Quality of Service profile (latency, scaling hints)
- Specifies application metadata (name, version, category)
- Represents the "what" at a business level, not the "how"

**GSMA Spec Context:** An Application is the business-level abstraction. It represents what the application provider wants to deploy, with quality of service requirements that guide placement decisions. The QoS profile tells the HOST what kind of edge placement is needed (low latency for gaming vs higher latency acceptable for batch processing). The Application references Artefacts, which contain the actual deployment details.

**Application States:**
- `Pending` - Being processed
- `Onboarded` - Ready for deployment
- `Deboarding` - Being removed
- `Failed` - Onboarding failed
- `Removed` - Successfully removed

**Key Fields:**

| Field | Purpose |
|-------|---------|
| `spec.appProviderId` | Who owns this application |
| `spec.appMetaData.name` | Application name |
| `spec.appMetaData.version` | Application version |
| `spec.appMetaData.mobilitySupport` | Whether app supports user mobility |
| `spec.componentSpecs` | References to Artefact resources |
| `spec.qoSProfile.latencyConstraints` | `LOW`, `MEDIUM`, `HIGH` - placement hint |
| `spec.qoSProfile.multiUserClients` | Single or multi-user application |
| `spec.qoSProfile.usersPerAppInst` | Expected users per instance (scaling hint) |
| `spec.statusLink` | Callback URL for status updates |

```yaml
# Example: CDN application with low latency requirements
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Application
metadata:
  name: my-cdn-app
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/federation-context-id: "ctx-123..."
spec:
  appProviderId: "my-company"
  appMetaData:
    name: "My CDN Application"
    version: "2.0.0"
    mobilitySupport: false        # CDN caches don't move with users
  componentSpecs:
    - artefactId: "my-app-bundle" # References the Artefact resource
  qoSProfile:
    latencyConstraints: LOW       # Needs edge placement (not central DC)
    multiUserClients: APP_TYPE_MULTI_USER
    usersPerAppInst: 10000        # Each instance serves 10k users
    provisioning: true
  statusLink: "https://api.my-company.com/webhooks/opg-status"
```

---

### ApplicationInstance

**What it is:** A request to deploy an Application to a specific edge zone.

**What it does:**
- References an Application (what to deploy)
- Specifies the target AvailabilityZone (where to deploy)
- Selects a flavour/resource tier (how much resources)
- This is the resource that triggers actual deployment

**GSMA Spec Context:** The ApplicationInstance is the deployment request. It says "deploy Application X to Zone Y with resource tier Z". When the HOST receives this, their orchestrator (Okto, OpenStack, etc.) watches for ApplicationInstance resources and creates the actual infrastructure. The ApplicationInstance status is updated as the deployment progresses.

**Application Instance States:**
- `Pending` - Deployment requested, not yet started
- `Ready` - Successfully deployed and running
- `Failed` - Deployment failed
- `Terminating` - Being removed

**Key Fields:**

| Field | Purpose |
|-------|---------|
| `spec.appProviderId` | Who owns this instance |
| `spec.appId` | Which Application to deploy |
| `spec.appVersion` | Which version of the Application |
| `spec.zoneInfo.zoneId` | Target AvailabilityZone |
| `spec.zoneInfo.flavourId` | Resource tier (like EC2 instance type) |
| `spec.zoneInfo.resourceConsumption` | `RESERVED_RES_USE` or `RESERVED_RES_AVOID` |
| `spec.callBackLink` | Callback URL for instance-specific updates |
| `status.state` | Current deployment state |

```yaml
# Example: Deploy CDN app to Madrid edge zone
apiVersion: opg.ewbi.nby.one/v1beta1
kind: ApplicationInstance
metadata:
  name: my-cdn-madrid
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/federation-context-id: "ctx-123..."
spec:
  appProviderId: "my-company"
  appId: "my-cdn-app"              # Which Application
  appVersion: "2.0.0"
  zoneInfo:
    zoneId: "zone-es-madrid-central"  # WHERE to deploy
    flavourId: "edge-large"           # Resource tier
    resourceConsumption: RESERVED_RES_USE
    resPool: "default-pool"
  callBackLink: "https://api.my-company.com/webhooks/instance/madrid"
```

---

## Resource Hierarchy and Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          RESOURCE HIERARCHY                                  │
│                                                                              │
│  Federation ─────────────────────────────────────────────────────────────── │
│      │                                                                       │
│      │  "I (GUEST) trust Telefonica (HOST). They offer Madrid, Barcelona."  │
│      │                                                                       │
│      ├── AvailabilityZone ───────────────────────────────────────────────── │
│      │       │                                                               │
│      │       │  "Madrid: 40.4168,-3.7038, urban, 5G connected"              │
│      │       │                                                               │
│      └── File ───────────────────────────────────────────────────────────── │
│           │                                                                  │
│           │  "Here's my container image at ghcr.io/my-company/my-app"       │
│           │                                                                  │
│           └── Artefact ──────────────────────────────────────────────────── │
│                 │                                                            │
│                 │  "Deploy image with 2 CPU, 4GB RAM, ports 80/443"         │
│                 │                                                            │
│                 └── Application ─────────────────────────────────────────── │
│                       │                                                      │
│                       │  "My CDN app, needs LOW latency, 10k users/instance"│
│                       │                                                      │
│                       └── ApplicationInstance ───────────────────────────── │
│                             │                                                │
│                             │  "Deploy my CDN app to Madrid zone, large tier"│
│                             │                                                │
│                             └── [Okto/Orchestrator deploys actual workload] │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Real-World Example: EdgeStream CDN

### Scenario

EdgeStream is a video streaming company in Germany. They want to deploy edge caches in Spain (Telefonica) to reduce latency for Spanish users.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│   Current: 120ms latency (Frankfurt → Spain)                                │
│   With edge: 15ms latency (Madrid edge → Spanish users)                     │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Step 1: HOST Creates Federation (Telefonica)

Telefonica registers EdgeStream as a client and creates a HOST federation:

```yaml
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Federation
metadata:
  name: fed-edgestream-telefonica
  namespace: federation-host
  labels:
    opg.ewbi.nby.one/federation-relation: host
    opg.ewbi.nby.one/origin-operator-name: EdgeStream
    opg.ewbi.nby.one/origin-client-id: edgestream-client-2024
spec:
  offeredAvailabilityZones:
    - zoneId: "zone-es-madrid-central"
      geolocation: "40.4168,-3.7038"
      geographyDetails: "Madrid Central - Telefonica Tower, 5G connected"
    - zoneId: "zone-es-barcelona-nord"
      geolocation: "41.3851,2.1734"
      geographyDetails: "Barcelona Nord - Port Olympic edge site"
  guestPartnerCredentials:
    clientId: edgestream-client-2024
```

### Step 2: GUEST Establishes Federation (EdgeStream)

EdgeStream creates a GUEST federation to connect:

```yaml
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Federation
metadata:
  name: fed-edgestream-telefonica
  namespace: federation-guest
  labels:
    opg.ewbi.nby.one/federation-relation: guest
spec:
  originOP:
    countryCode: "DE"
    mobileNetworkCodes:
      mcc: "262"
      mncs: ["01"]
  partner:
    statusLink: "https://opg-ewbi.telefonica.com"
    callbackCredentials:
      clientId: "edgestream-client-2024"
      tokenUrl: "https://idp.telefonica.com/token"
  acceptedAvailabilityZones:
    - "zone-es-madrid-central"
```

**Result:** GUEST receives `federationContextId` and `offeredAvailabilityZones` in status.

### Step 3: Upload Container Image (File)

```yaml
apiVersion: opg.ewbi.nby.one/v1beta1
kind: File
metadata:
  name: edgestream-cache-v2
  namespace: federation-guest
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/federation-context-id: "ctx-a1b2c3d4..."
spec:
  appProviderId: "edgestream-gmbh"
  fileName: "edge-cache"
  fileType: DOCKER
  fileVersion: "2.4.1"
  repoLocation:
    url: "ghcr.io/edgestream/edge-cache"
    type: private
  image:
    instructionSetArchitecture: ISA_X86_64
    os:
      distribution: UBUNTU
      version: OS_VERSION_UBUNTU_2204_LTS
```

### Step 4: Create Deployment Package (Artefact)

```yaml
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Artefact
metadata:
  name: edgestream-cache-bundle
  namespace: federation-guest
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/federation-context-id: "ctx-a1b2c3d4..."
spec:
  appProviderId: "edgestream-gmbh"
  artefactName: "edge-cache-bundle"
  artefactVersion: "2.4.1"
  virtType: "CONTAINER"
  componentSpec:
    - name: "edge-cache"
      images: ["edgestream-cache-v2"]
      numOfInstances: 3
      computeResourceProfile:
        numCPU: "4"
        memory: 8192
      exposedInterfaces:
        - interfaceId: "https"
          port: 443
          protocol: "TCP"
          visibilityType: "EXTERNAL"
```

### Step 5: Register Application

```yaml
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Application
metadata:
  name: edgestream-cdn
  namespace: federation-guest
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/federation-context-id: "ctx-a1b2c3d4..."
spec:
  appProviderId: "edgestream-gmbh"
  appMetaData:
    name: "EdgeStream CDN Cache"
    version: "2.4.1"
    mobilitySupport: false
  componentSpecs:
    - artefactId: "edgestream-cache-bundle"
  qoSProfile:
    latencyConstraints: LOW
    multiUserClients: APP_TYPE_MULTI_USER
    usersPerAppInst: 50000
```

### Step 6: Deploy to Madrid (ApplicationInstance)

```yaml
apiVersion: opg.ewbi.nby.one/v1beta1
kind: ApplicationInstance
metadata:
  name: edgestream-cdn-madrid
  namespace: federation-guest
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/federation-context-id: "ctx-a1b2c3d4..."
spec:
  appProviderId: "edgestream-gmbh"
  appId: "edgestream-cdn"
  appVersion: "2.4.1"
  zoneInfo:
    zoneId: "zone-es-madrid-central"
    flavourId: "edge-large"
    resourceConsumption: RESERVED_RES_USE
```

### What Happens

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              FLOW                                            │
│                                                                              │
│  EdgeStream (GUEST)                    Telefonica (HOST)                    │
│                                                                              │
│  1. Federation CR ──── POST /federation ────► Federation CR                 │
│  2. File CR ────────── POST /files ─────────► File CR (mirrored)            │
│  3. Artefact CR ────── POST /artefacts ─────► Artefact CR (mirrored)        │
│  4. Application CR ─── POST /applications ──► Application CR (mirrored)     │
│  5. AppInstance CR ─── POST /app-instances ─► AppInstance CR (mirrored)     │
│                                                     │                        │
│                                                     ▼                        │
│                                              ┌─────────────┐                │
│                                              │    Okto     │                │
│                                              │  (watches)  │                │
│                                              └──────┬──────┘                │
│                                                     │                        │
│                                                     ▼                        │
│                                              ┌─────────────┐                │
│                                              │ 3x edge-    │                │
│                                              │ cache pods  │                │
│                                              │ in Madrid   │                │
│                                              └─────────────┘                │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Simpler Example: IoT Gateway

A manufacturing company wants to deploy an IoT gateway at a factory edge location:

```yaml
# All resources in one file for simplicity
---
apiVersion: opg.ewbi.nby.one/v1beta1
kind: File
metadata:
  name: iot-gateway-image
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/federation-context-id: "ctx-iot-123"
spec:
  appProviderId: "acme-iot"
  fileName: "iot-gateway"
  fileType: DOCKER
  fileVersion: "1.0.0"
  repoLocation:
    url: "docker.io/acmeiot/gateway"
    type: public
---
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Artefact
metadata:
  name: iot-gateway-bundle
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/federation-context-id: "ctx-iot-123"
spec:
  appProviderId: "acme-iot"
  artefactName: "iot-gateway"
  artefactVersion: "1.0.0"
  virtType: "CONTAINER"
  componentSpec:
    - name: "gateway"
      images: ["iot-gateway-image"]
      numOfInstances: 1
      computeResourceProfile:
        numCPU: "1"
        memory: 512
      exposedInterfaces:
        - interfaceId: "mqtt"
          port: 1883
          protocol: "TCP"
          visibilityType: "EXTERNAL"
---
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Application
metadata:
  name: iot-gateway-app
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/federation-context-id: "ctx-iot-123"
spec:
  appProviderId: "acme-iot"
  appMetaData:
    name: "ACME IoT Gateway"
    version: "1.0.0"
  componentSpecs:
    - artefactId: "iot-gateway-bundle"
  qoSProfile:
    latencyConstraints: LOW
    usersPerAppInst: 1000
---
apiVersion: opg.ewbi.nby.one/v1beta1
kind: ApplicationInstance
metadata:
  name: iot-gateway-factory-munich
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/federation-context-id: "ctx-iot-123"
spec:
  appProviderId: "acme-iot"
  appId: "iot-gateway-app"
  zoneInfo:
    zoneId: "zone-de-munich-industrial"
    flavourId: "edge-small"
```

---

## Summary

| Resource | What It Is | Key Purpose |
|----------|------------|-------------|
| **Federation** | Trust relationship | Establish credentials, exchange available zones |
| **AvailabilityZone** | Edge location | Advertise physical sites with coordinates |
| **File** | Image reference | Point to container/VM image in a registry |
| **Artefact** | Deployment package | Bundle images with resources, ports, scaling |
| **Application** | Business definition | Define QoS requirements, reference artefacts |
| **ApplicationInstance** | Deployment request | Request deployment to specific zone |

---

## Related Documentation

- [ARCHITECTURE.md](./ARCHITECTURE.md) - Technical architecture details
- [INSTRUCTIONS.md](./INSTRUCTIONS.md) - Local development setup
- [GSMA OPG E/WBI Spec v5.0](https://www.gsma.com/solutions-and-impact/technologies/networks/gsma_resources/gsma-operator-platform-group-east-westbound-interface-apis-version-5-0/)
- [GSMA OPG E/WBI Spec v6.0 PDF](https://www.gsma.com/solutions-and-impact/technologies/networks/wp-content/uploads/2025/03/OPG.04-v6.0-EWBI-APIs.pdf)
