# OPG EWBI Operator Architecture

This document describes the architecture of the OPG EWBI (East/WestBound Interface) Operator and how it enables federation between operator platforms for edge computing.

## What This Operator Actually Does

```
┌────────────────────────────────────────────────────────────────────────────────┐
│                           WHAT THIS OPERATOR DOES                               │
│                                                                                 │
│   It's a "METADATA BROKER" - it syncs INTENT between operator platforms        │
│                                                                                 │
│   It does NOT:                                                                  │
│   ❌ Create VMs                                                                 │
│   ❌ Deploy containers                                                          │
│   ❌ Provision infrastructure                                                   │
│                                                                                 │
│   It DOES:                                                                      │
│   ✅ Sync resource definitions (CRs) from GUEST → HOST                         │
│   ✅ Express deployment INTENT via Kubernetes CRs                              │
│                                                                                 │
│   ANOTHER OPERATOR (like NearbyOne Okto) watches these CRs and does the        │
│   actual work of creating VMs/containers                                       │
└────────────────────────────────────────────────────────────────────────────────┘
```

## Overview

The OPG EWBI Operator implements a subset of the **GSMA OPG (Operator Platform Group) East/WestBound Interface** specification. It enables telecom operators to federate their edge computing resources, allowing applications to be deployed across multiple operator platforms seamlessly.

**The problem it solves:**

> "I'm Telecom Operator A. I want to deploy my app on Telecom Operator B's edge infrastructure in Madrid. How do we talk to each other?"

**Answer:** Both operators run OPG EWBI. The GUEST creates CRs expressing intent, they get mirrored to HOST, and HOST's orchestrator (Okto, Kubernetes, OpenStack, whatever) does the actual deployment.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           GSMA OPG ECOSYSTEM                                 │
│                                                                              │
│    Telefonica (Spain)                      Vodafone (Germany)               │
│    wants to deploy                         has edge servers                 │
│    streaming app                           in Frankfurt                     │
│         │                                        │                          │
│         │         OPG EWBI Protocol              │                          │
│         └────────────────────────────────────────┘                          │
│                       Federation!                                           │
│                                                                              │
│    Operator A                              Operator B                        │
│    (GUEST)                                 (HOST)                            │
│   ┌─────────────┐                        ┌─────────────┐                    │
│   │   Edge      │◄─── Federation ───────►│   Edge      │                    │
│   │   Platform  │     (OPG EWBI)         │   Platform  │                    │
│   └─────────────┘                        └─────────────┘                    │
│                                                                              │
│   Applications from Operator A can be deployed on Operator B's edge         │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Components

### 1. opg-ewbi-operator (Controller)

The Kubernetes operator that watches Custom Resources and manages the federation lifecycle.

**Responsibilities:**
- Watch for CR changes (create, update, delete)
- For GUEST resources: Make HTTP calls to HOST's API
- For HOST resources: Mark as ready for external orchestrators
- Manage finalizers for cleanup on deletion

**Location:** `cmd/main.go`, `internal/controller/`

### 2. opg-ewbi-api (REST API)

The HTTP API that implements the OPG East/WestBound Interface endpoints.

**Responsibilities:**
- Receive federation requests from GUEST operators
- Create corresponding CRs in the HOST cluster
- Return federation context IDs and offered resources

**Location:** Separate repository `github.com/neonephos-katalis/opg-ewbi-api`

## HOST vs GUEST Roles

The operator supports two roles, determined by the `opg.ewbi.nby.one/federation-relation` label:

### HOST Role

The HOST is the operator platform that **offers** edge computing resources.

```yaml
metadata:
  labels:
    opg.ewbi.nby.one/federation-relation: host
```

**Behavior:**
- Resources are created directly (no external API calls)
- Offers availability zones to federation partners
- Receives requests from GUEST operators via the API
- External orchestrators (e.g., NearbyOne Okto) handle actual deployment

### GUEST Role

The GUEST is the operator platform that **consumes** edge resources from a HOST.

```yaml
metadata:
  labels:
    opg.ewbi.nby.one/federation-relation: guest
```

**Behavior:**
- Resources trigger HTTP calls to the HOST's API
- Receives offered availability zones from HOST
- Resources are mirrored to HOST cluster
- Deletions are propagated to HOST

## Who Creates the VM/Container?

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                            THE FULL PICTURE                                      │
│                                                                                  │
│  GUEST Cluster              HOST Cluster                Edge Infrastructure     │
│  ─────────────              ────────────                ────────────────────     │
│                                                                                  │
│  ┌──────────────┐          ┌──────────────┐           ┌──────────────────┐      │
│  │ App Provider │          │   Telecom    │           │  Edge Datacenter │      │
│  │   creates    │          │   Operator   │           │    in Madrid     │      │
│  │    CRs       │          │              │           │                  │      │
│  └──────┬───────┘          └──────────────┘           └──────────────────┘      │
│         │                                                                        │
│         ▼                                                                        │
│  ┌──────────────┐                                                               │
│  │ opg-ewbi-    │──── HTTP ────┐                                                │
│  │ operator     │              │                                                │
│  └──────────────┘              ▼                                                │
│                         ┌──────────────┐                                        │
│                         │ opg-ewbi-api │                                        │
│                         └──────┬───────┘                                        │
│                                │                                                │
│                                ▼                                                │
│                         ┌──────────────┐                                        │
│                         │  Mirrored    │                                        │
│                         │    CRs       │                                        │
│                         └──────┬───────┘                                        │
│                                │                                                │
│                                ▼                                                │
│                         ┌──────────────┐         ┌──────────────────┐           │
│                         │  NearbyOne   │────────►│  Creates actual  │           │
│                         │    Okto      │         │  VM/Container    │           │
│                         │ (watches CRs)│         │  on edge infra   │           │
│                         └──────────────┘         └──────────────────┘           │
│                                                                                  │
│                         ▲                                                        │
│                         │                                                        │
│                    THIS IS THE PART                                             │
│                    THAT CREATES VMs!                                            │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## Custom Resource Definitions - Concrete Examples

### File = The Deployable Image

A **File** is a pointer to a deployable image. It can be a Docker image, VM disk (QCOW2), or any other deployable artifact.

```yaml
# EXAMPLE 1: Docker container image
apiVersion: opg.ewbi.nby.one/v1beta1
kind: File
metadata:
  name: nginx-file
  labels:
    opg.ewbi.nby.one/federation-context-id: "<federation-id>"
    opg.ewbi.nby.one/federation-relation: guest
spec:
  appProviderId: "my-company"
  fileName: "nginx:latest"
  fileType: DOCKER                    # ← It's a container image
  fileVersion: "1.25.0"
  repoLocation:
    url: "docker.io"
    type: public
  image:
    instructionSetArchitecture: ISA_X86_64
    os:
      distribution: UBUNTU
      version: OS_VERSION_UBUNTU_2204_LTS
```

```yaml
# EXAMPLE 2: VM disk image (QCOW2)
apiVersion: opg.ewbi.nby.one/v1beta1
kind: File
metadata:
  name: ubuntu-vm-image
spec:
  fileName: "ubuntu-22.04-server.qcow2"
  fileType: QCOW2                     # ← It's a VM image!
  fileVersion: "22.04"
  repoLocation:
    url: "https://cloud-images.ubuntu.com"
    type: public
  image:
    instructionSetArchitecture: ISA_X86_64
    os:
      architecture: "x86_64"
      distribution: UBUNTU
      license: OS_LICENSE_TYPE_FREE
      version: OS_VERSION_UBUNTU_2204_LTS
```

**What it represents:** A pointer to a deployable image (Docker, QCOW2 VM, OCI, etc.)

### Artefact = Deployment Package

An **Artefact** bundles one or more Files with resource requirements and exposed interfaces. Think of it like a Helm chart or docker-compose file.

```yaml
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Artefact
metadata:
  name: web-server-artefact
  labels:
    opg.ewbi.nby.one/federation-context-id: "<federation-id>"
    opg.ewbi.nby.one/federation-relation: guest
spec:
  appProviderId: "my-company"
  artefactName: "web-server-package"
  artefactVersion: "1.0.0"
  virtType: "CONTAINER"               # or "VM" for virtual machines
  descriptorType: "HELM"              # or "DOCKER_COMPOSE", "TOSCA", etc.
  componentSpec:
    - name: "nginx"
      images:
        - "file-uuid-here"            # ← References a File
      numOfInstances: 2               # ← How many replicas
      restartPolicy: "Always"
      computeResourceProfile:
        numCPU: "2"
        memory: 4096                  # MB
        cpuArchType: "x86_64"
        cpuExclusivity: false
      exposedInterfaces:
        - port: 80
          protocol: "TCP"
          interfaceId: "http"
          visibilityType: "EXTERNAL"
        - port: 443
          protocol: "TCP"
          interfaceId: "https"
          visibilityType: "EXTERNAL"
      commandLineParams:
        command: ["nginx"]
        args: ["-g", "daemon off;"]
```

**What it represents:** Like a Helm chart or docker-compose - bundles images with resource requirements and networking.

### Application = Business-Level Definition

An **Application** represents the business-level abstraction - the "app" from the provider's perspective, with QoS requirements.

```yaml
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Application
metadata:
  name: streaming-app
  labels:
    opg.ewbi.nby.one/federation-context-id: "<federation-id>"
    opg.ewbi.nby.one/federation-relation: guest
spec:
  appProviderId: "my-company"
  appMetaData:
    name: "my-streaming-app"
    version: "2.0.0"
    accessToken: "app-access-token"
    mobilitySupport: true             # App supports user mobility
  componentSpecs:
    - artefactId: "artefact-uuid"     # ← References Artefact
  qoSProfile:
    latencyConstraints: LOW           # ← Edge placement hint (LOW/MEDIUM/HIGH)
    multiUserClients: APP_TYPE_MULTI_USER
    usersPerAppInst: 1000             # ← Scaling hint
    provisioning: true
  statusLink: "https://myapp.com/callback"  # Callback URL for status updates
```

**What it represents:** "I have an app called X, it needs low latency, supports 1000 users per instance"

### ApplicationInstance = Deployed Instance

An **ApplicationInstance** is a request to deploy an Application in a specific edge zone.

```yaml
apiVersion: opg.ewbi.nby.one/v1beta1
kind: ApplicationInstance
metadata:
  name: streaming-app-madrid
  labels:
    opg.ewbi.nby.one/federation-context-id: "<federation-id>"
    opg.ewbi.nby.one/federation-relation: guest
spec:
  appProviderId: "my-company"
  appId: "application-uuid"           # ← Which Application
  appVersion: "2.0.0"
  zoneInfo:
    zoneId: "zone-es-madrid-001"      # ← WHERE to deploy (edge location)
    flavourId: "medium"               # ← Like t2.medium, resource size
    resourceConsumption: RESERVED_RES_AVOID  # or RESERVED_RES_USE
    resPool: "default-pool"
  callBackLink: "https://myapp.com/instance-callback"
```

**What it represents:** "Deploy app X in Madrid edge zone, use medium resources"

### Federation = Trust Relationship

A **Federation** establishes trust between two operator platforms.

```yaml
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Federation
metadata:
  name: fed-telefonica-vodafone
  labels:
    opg.ewbi.nby.one/federation-relation: guest  # or "host"
    opg.ewbi.nby.one/id: "<external-id>"
spec:
  originOP:
    countryCode: "ES"
    mobileNetworkCodes:
      mcc: "214"
      mncs: ["01", "02"]
    fixedNetworkCodes: ["none"]
  partner:
    callbackCredentials:
      clientId: "<client-id>"
      tokenUrl: "https://idp.example.com/token"
    statusLink: "http://guest-api:8080"
  guestPartnerCredentials:
    clientId: "<client-id>"
    tokenUrl: "http://host-api:8080"
  offeredAvailabilityZones:           # HOST fills this
    - zoneId: "zone-es-madrid-001"
      geolocation: "40.4168,-3.7038"
      geographyDetails: "Madrid, Spain - Urban edge"
  acceptedAvailabilityZones:          # GUEST fills this
    - "zone-es-madrid-001"
status:
  federationContextId: "<uuid>"       # Assigned by HOST
  state: Available
```

### AvailabilityZone = Edge Location

An **AvailabilityZone** represents a physical edge location offered by the HOST.

```yaml
apiVersion: opg.ewbi.nby.one/v1beta1
kind: AvailabilityZone
metadata:
  name: zone-es-madrid-001
spec:
  zoneId: "zone-es-madrid-001"
  geolocation: "40.4168,-3.7038"      # Lat/Long
  geographyDetails: "Madrid urban edge - Telefonica datacenter"
```

## Resource Hierarchy

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          RESOURCE HIERARCHY                                  │
│                                                                              │
│  Federation                                                                  │
│      │                                                                       │
│      ├── AvailabilityZone (offered by HOST)                                 │
│      │                                                                       │
│      └── File ──────────────────┐                                           │
│           │                     │                                           │
│           │   (references)      │                                           │
│           ▼                     │                                           │
│      Artefact ──────────────────┤                                           │
│           │                     │                                           │
│           │   (references)      │                                           │
│           ▼                     │                                           │
│      Application ───────────────┤                                           │
│           │                     │                                           │
│           │   (references)      │                                           │
│           ▼                     │                                           │
│      ApplicationInstance ───────┘                                           │
│           │                                                                  │
│           │   (deployed to)                                                  │
│           ▼                                                                  │
│      AvailabilityZone                                                       │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Federation Flow

### Step 1: Establish Federation

```
GUEST Cluster                                    HOST Cluster
─────────────────                                ─────────────────

1. Create Federation CR
   (relation: guest)
        │
        ▼
2. Operator detects CR ─────HTTP POST────────►  3. API receives request
                        /federation                     │
                                                        ▼
                                                4. Creates Federation CR
                                                   (relation: host)
                                                        │
        ◄───────────────────────────────────────────────┘
        │   Response: federationContextId,
        │             offeredAvailabilityZones
        ▼
5. Operator updates
   Federation status
```

### Step 2: Deploy Resources

Once federation is established, resources flow from GUEST to HOST:

```
GUEST Cluster                                    HOST Cluster
─────────────────                                ─────────────────

1. Create File CR ──────HTTP POST───────────►   2. API creates File CR
                   /artefact-management/files           │
                                                        ▼
                                                3. File available on HOST

4. Create Artefact CR ──HTTP POST───────────►   5. API creates Artefact CR
                     /artefact-management/artefacts

6. Create Application CR ─HTTP POST─────────►   7. API creates Application CR
                        /app-lcm/applications

8. Create AppInstance CR ─HTTP POST─────────►   9. API creates AppInstance CR
                        /app-lcm/app-instances        │
                                                      ▼
                                               10. External orchestrator
                                                   (Okto) watches CR
                                                      │
                                                      ▼
                                               11. Okto creates VM/Container
                                                   on edge infrastructure
```

### Step 3: Cleanup on Deletion

```
GUEST Cluster                                    HOST Cluster
─────────────────                                ─────────────────

1. Delete CR
   (finalizer present)
        │
        ▼
2. Operator detects ────HTTP DELETE──────────►  3. API deletes mirrored CR
   deletion                                            │
        │                                              ▼
        ◄──────────────────────────────────────────────┘
        │   Response: 200 OK
        ▼
4. Remove finalizer
5. CR deleted
```

## Resource Mirroring Summary

| Resource | GUEST → HOST | Created Via | What Gets Mirrored |
|----------|--------------|-------------|-------------------|
| Federation | ✅ | POST /federation | originOP, partner, credentials |
| File | ✅ | POST /artefact-management/files | Image pointer, repo location |
| Artefact | ✅ | POST /artefact-management/artefacts | Component specs, resources |
| Application | ✅ | POST /app-lcm/applications | QoS profile, artefact refs |
| ApplicationInstance | ✅ | POST /app-lcm/app-instances | Zone, flavour, app ref |
| AvailabilityZone | ❌ | HOST only | N/A |

## Key Labels

| Label | Purpose |
|-------|---------|
| `opg.ewbi.nby.one/federation-relation` | `guest` or `host` - determines behavior |
| `opg.ewbi.nby.one/federation-context-id` | Links resource to its federation |
| `opg.ewbi.nby.one/id` | External ID for cross-platform reference |

## Deployment Architecture

### Single Cluster (Development/Testing)

```
┌─────────────────────────────────────────────────────────────┐
│                      Kind/Minikube Cluster                   │
│                                                              │
│  ┌─────────────────────┐    ┌─────────────────────┐         │
│  │  federation-host    │    │  federation-guest   │         │
│  │  namespace          │    │  namespace          │         │
│  │                     │    │                     │         │
│  │  - Operator         │    │  - Operator         │         │
│  │  - API              │    │  - API              │         │
│  │  - HOST CRs         │◄───│  - GUEST CRs        │         │
│  │                     │    │                     │         │
│  └─────────────────────┘    └─────────────────────┘         │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Multi-Cluster (Production)

```
┌─────────────────────────────┐    ┌─────────────────────────────┐
│     Operator A Cluster      │    │     Operator B Cluster      │
│         (GUEST)             │    │         (HOST)              │
│                             │    │                             │
│  ┌───────────────────────┐  │    │  ┌───────────────────────┐  │
│  │  opg-ewbi-operator    │  │    │  │  opg-ewbi-operator    │  │
│  │  opg-ewbi-api         │  │    │  │  opg-ewbi-api         │  │
│  └───────────┬───────────┘  │    │  └───────────▲───────────┘  │
│              │              │    │              │              │
│              │   HTTP/HTTPS │    │              │              │
│              └──────────────┼────┼──────────────┘              │
│                             │    │                             │
│  ┌───────────────────────┐  │    │  ┌───────────────────────┐  │
│  │  GUEST CRs            │  │    │  │  HOST CRs (mirrored)  │  │
│  │  - Federation         │  │    │  │  - Federation         │  │
│  │  - File               │  │    │  │  - File               │  │
│  │  - Artefact           │  │    │  │  - Artefact           │  │
│  │  - Application        │  │    │  │  - Application        │  │
│  │  - AppInstance        │  │    │  │  - AppInstance        │  │
│  └───────────────────────┘  │    │  └───────────────────────┘  │
│                             │    │              │              │
└─────────────────────────────┘    │              ▼              │
                                   │  ┌───────────────────────┐  │
                                   │  │  External Orchestrator│  │
                                   │  │  (e.g., NearbyOne     │  │
                                   │  │   Okto)               │  │
                                   │  └───────────────────────┘  │
                                   │              │              │
                                   │              ▼              │
                                   │  ┌───────────────────────┐  │
                                   │  │  Edge Infrastructure  │  │
                                   │  │  (VMs / Containers)   │  │
                                   │  └───────────────────────┘  │
                                   └─────────────────────────────┘
```

## Real-World Example: Deploying a Streaming App

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  SCENARIO: Netflix-like app wants to deploy on Telefonica's Madrid edge    │
└─────────────────────────────────────────────────────────────────────────────┘

Step 1: App Provider (GUEST) creates resources:

    File: nginx:latest (CDN proxy)
    File: streaming-server:v2 (Video server)
         │
         ▼
    Artefact: streaming-package
    - nginx (2 replicas, 2CPU, 4GB)
    - streaming-server (4 replicas, 8CPU, 16GB)
    - Ports: 80, 443, 8080
         │
         ▼
    Application: streaming-app
    - latencyConstraints: LOW (needs edge!)
    - usersPerAppInst: 10000
         │
         ▼
    ApplicationInstance: streaming-app-madrid
    - zoneId: zone-es-madrid-001
    - flavourId: large

Step 2: OPG EWBI mirrors to HOST (Telefonica)

Step 3: Telefonica's Okto orchestrator sees ApplicationInstance CR:
    - Provisions 6 containers (2 nginx + 4 streaming)
    - Configures networking
    - Reports status back

Step 4: Users in Madrid get low-latency video streaming!
```

## Integration Points

### External Orchestrators

The HOST cluster typically has external orchestrators watching the CRs:

- **NearbyOne Okto**: Deploys workloads based on ApplicationInstance CRs
- **Kubernetes native**: Could use standard Deployments/StatefulSets
- **OpenStack**: For VM-based deployments
- **Custom controllers**: Can watch any CR type for custom logic

### Identity Providers

Federation credentials can integrate with:

- **Hydra**: OAuth2 server for token management
- **Keycloak**: Enterprise identity management
- **Custom IdP**: Via the callback credentials mechanism

## neonephos-katalis Context

From the GitHub organization and code:
- **NearbyOne** is the platform/product name
- **Okto** is their orchestrator that watches these CRs and deploys workloads
- **Katalis** is the project/initiative name
- This implements GSMA OPG specs for telecom edge federation

The operator is **one piece** of a larger edge computing platform - it handles the "federation protocol" while other components handle the actual infrastructure provisioning.

## Security Considerations

1. **TLS**: Use `--opg-insecure-skip-verify=false` in production
2. **Credentials**: Store in Kubernetes Secrets, not in CR specs
3. **RBAC**: Each namespace should have isolated permissions
4. **Network Policies**: Restrict API access to known partners

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NAMESPACE` | `default` | Namespace the operator watches |

## Related Specifications

- [GSMA OPG Specification](https://www.gsma.com/futurenetworks/operator-platform-group/)
- [ETSI MEC](https://www.etsi.org/technologies/multi-access-edge-computing)
- [CAMARA APIs](https://camaraproject.org/)
