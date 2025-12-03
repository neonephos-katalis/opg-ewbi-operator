# Federation Topology - Multi-Federation Support

This document explains the different federation topologies supported by the OPG EWBI operator.

## Key Finding: It's NOT Peer-to-Peer Only

The operator maintains **separate HTTP clients per federation ID**:

```go
// internal/opg/opg.go
type OPGClientsMap struct {
    opgClients map[string]opgc.ClientWithResponsesInterface  // keyed by fedId
}
```

This means **multiple federations are fully supported** in various topologies.

---

## Topology 1: One HOST, Multiple GUESTs

A single operator platform can receive connections from multiple GUEST operators.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    ONE HOST, MULTIPLE GUESTS                                 │
│                                                                              │
│                         ┌─────────────────────┐                             │
│                         │    Telefonica       │                             │
│                         │      (HOST)         │                             │
│                         │                     │                             │
│                         │ ┌─────────────────┐ │                             │
│                         │ │ Federation:     │ │                             │
│                         │ │ EdgeStream      │ │                             │
│                         │ │ (host)          │ │                             │
│                         │ ├─────────────────┤ │                             │
│                         │ │ Federation:     │ │                             │
│                         │ │ Netflix         │ │                             │
│                         │ │ (host)          │ │                             │
│                         │ ├─────────────────┤ │                             │
│                         │ │ Federation:     │ │                             │
│                         │ │ ACME IoT        │ │                             │
│                         │ │ (host)          │ │                             │
│                         │ └─────────────────┘ │                             │
│                         └──────────▲──────────┘                             │
│                                    │                                         │
│              ┌─────────────────────┼─────────────────────┐                  │
│              │                     │                     │                  │
│              │                     │                     │                  │
│    ┌─────────┴─────────┐ ┌─────────┴─────────┐ ┌─────────┴─────────┐       │
│    │   EdgeStream      │ │    Netflix        │ │    ACME IoT       │       │
│    │    (GUEST)        │ │    (GUEST)        │ │    (GUEST)        │       │
│    │                   │ │                   │ │                   │       │
│    │ Federation:       │ │ Federation:       │ │ Federation:       │       │
│    │ Telefonica (guest)│ │ Telefonica (guest)│ │ Telefonica (guest)│       │
│    └───────────────────┘ └───────────────────┘ └───────────────────┘       │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

**How it works:**
- Telefonica creates 3 separate Federation CRs, each with `relation: host`
- Each GUEST registers with their own OAuth2 credentials
- Each GUEST connects independently
- Resources are isolated by `federation-context-id` label

**HOST Side (Telefonica):**
```yaml
# Federation CR for EdgeStream
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Federation
metadata:
  name: fed-edgestream
  labels:
    opg.ewbi.nby.one/federation-relation: host
    opg.ewbi.nby.one/origin-client-id: edgestream-client-2024
spec:
  offeredAvailabilityZones:
    - zoneId: "zone-es-madrid"
  guestPartnerCredentials:
    clientId: edgestream-client-2024
---
# Federation CR for Netflix
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Federation
metadata:
  name: fed-netflix
  labels:
    opg.ewbi.nby.one/federation-relation: host
    opg.ewbi.nby.one/origin-client-id: netflix-client-2024
spec:
  offeredAvailabilityZones:
    - zoneId: "zone-es-madrid"
    - zoneId: "zone-es-barcelona"
  guestPartnerCredentials:
    clientId: netflix-client-2024
---
# Federation CR for ACME IoT
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Federation
metadata:
  name: fed-acme-iot
  labels:
    opg.ewbi.nby.one/federation-relation: host
    opg.ewbi.nby.one/origin-client-id: acme-iot-client-2024
spec:
  offeredAvailabilityZones:
    - zoneId: "zone-es-valencia"
  guestPartnerCredentials:
    clientId: acme-iot-client-2024
```

---

## Topology 2: One GUEST, Multiple HOSTs

A single app provider can deploy to multiple operator platforms simultaneously.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    ONE GUEST, MULTIPLE HOSTS                                 │
│                                                                              │
│                         ┌─────────────────────┐                             │
│                         │    EdgeStream       │                             │
│                         │     (GUEST)         │                             │
│                         │                     │                             │
│                         │ ┌─────────────────┐ │                             │
│                         │ │ Federation:     │ │                             │
│                         │ │ Telefonica      │ │                             │
│                         │ │ (guest)         │ │                             │
│                         │ ├─────────────────┤ │                             │
│                         │ │ Federation:     │ │                             │
│                         │ │ Deutsche Telekom│ │                             │
│                         │ │ (guest)         │ │                             │
│                         │ ├─────────────────┤ │                             │
│                         │ │ Federation:     │ │                             │
│                         │ │ Orange          │ │                             │
│                         │ │ (guest)         │ │                             │
│                         │ └─────────────────┘ │                             │
│                         └──────────┬──────────┘                             │
│                                    │                                         │
│              ┌─────────────────────┼─────────────────────┐                  │
│              │                     │                     │                  │
│              ▼                     ▼                     ▼                  │
│    ┌───────────────────┐ ┌───────────────────┐ ┌───────────────────┐       │
│    │   Telefonica      │ │ Deutsche Telekom  │ │    Orange         │       │
│    │    (HOST)         │ │    (HOST)         │ │    (HOST)         │       │
│    │                   │ │                   │ │                   │       │
│    │ Madrid, Barcelona │ │ Frankfurt, Berlin │ │ Paris, Lyon       │       │
│    └───────────────────┘ └───────────────────┘ └───────────────────┘       │
│                                                                              │
│    EdgeStream deploys CDN caches across all three operators' edge zones     │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

**How it works:**
- EdgeStream creates 3 separate Federation CRs, each with `relation: guest`
- Each federation points to a different HOST's API (`spec.partner.statusLink`)
- Resources use different `federation-context-id` labels for each HOST

**GUEST Side (EdgeStream):**
```yaml
# Federation with Telefonica
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Federation
metadata:
  name: fed-telefonica
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/id: "edgestream-telefonica-2024"
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
---
# Federation with Deutsche Telekom
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Federation
metadata:
  name: fed-deutsche-telekom
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/id: "edgestream-dt-2024"
spec:
  originOP:
    countryCode: "DE"
    mobileNetworkCodes:
      mcc: "262"
      mncs: ["01"]
  partner:
    statusLink: "https://opg-ewbi.telekom.de"
    callbackCredentials:
      clientId: "edgestream-dt-client"
      tokenUrl: "https://idp.telekom.de/token"
---
# Federation with Orange
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Federation
metadata:
  name: fed-orange
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/id: "edgestream-orange-2024"
spec:
  originOP:
    countryCode: "DE"
    mobileNetworkCodes:
      mcc: "262"
      mncs: ["01"]
  partner:
    statusLink: "https://opg-ewbi.orange.com"
    callbackCredentials:
      clientId: "edgestream-orange-client"
      tokenUrl: "https://idp.orange.com/token"
```

**Deploying to multiple HOSTs:**
```yaml
# Deploy to Telefonica (Madrid)
apiVersion: opg.ewbi.nby.one/v1beta1
kind: ApplicationInstance
metadata:
  name: cdn-madrid
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/federation-context-id: "ctx-telefonica-123"  # Telefonica's context
spec:
  appId: "edgestream-cdn"
  zoneInfo:
    zoneId: "zone-es-madrid"
---
# Deploy to Deutsche Telekom (Frankfurt)
apiVersion: opg.ewbi.nby.one/v1beta1
kind: ApplicationInstance
metadata:
  name: cdn-frankfurt
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/federation-context-id: "ctx-dt-456"  # DT's context
spec:
  appId: "edgestream-cdn"
  zoneInfo:
    zoneId: "zone-de-frankfurt"
---
# Deploy to Orange (Paris)
apiVersion: opg.ewbi.nby.one/v1beta1
kind: ApplicationInstance
metadata:
  name: cdn-paris
  labels:
    opg.ewbi.nby.one/federation-relation: guest
    opg.ewbi.nby.one/federation-context-id: "ctx-orange-789"  # Orange's context
spec:
  appId: "edgestream-cdn"
  zoneInfo:
    zoneId: "zone-fr-paris"
```

---

## Topology 3: Same Operator as Both GUEST and HOST

An operator platform can simultaneously act as HOST (receiving deployments) and GUEST (deploying to others).

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                 SAME OPERATOR AS BOTH GUEST AND HOST                         │
│                                                                              │
│                         ┌─────────────────────┐                             │
│                         │  Deutsche Telekom   │                             │
│                         │                     │                             │
│                         │ ┌─────────────────┐ │                             │
│                         │ │ AS HOST:        │ │                             │
│                         │ │ Federation:     │ │                             │
│                         │ │ EdgeStream      │ │◄─────── EdgeStream          │
│                         │ │ (host)          │ │         deploys here        │
│                         │ ├─────────────────┤ │                             │
│                         │ │ AS GUEST:       │ │                             │
│                         │ │ Federation:     │ │──────►  Deploy DT app       │
│                         │ │ Telefonica      │ │         on Telefonica       │
│                         │ │ (guest)         │ │                             │
│                         │ └─────────────────┘ │                             │
│                         └─────────────────────┘                             │
│                                                                              │
│    Deutsche Telekom:                                                         │
│    - Receives apps from EdgeStream (HOST role)                              │
│    - Deploys its own apps on Telefonica (GUEST role)                        │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

**How it works:**
- The `federation-relation` label is **per-Federation CR**, not per-namespace or per-cluster
- Same namespace can have both HOST and GUEST Federation CRs
- The operator handles both directions based on the label

**Deutsche Telekom's configuration:**
```yaml
# HOST role - receiving deployments from EdgeStream
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Federation
metadata:
  name: fed-edgestream
  namespace: federation
  labels:
    opg.ewbi.nby.one/federation-relation: host  # <-- HOST
    opg.ewbi.nby.one/origin-client-id: edgestream-client
spec:
  offeredAvailabilityZones:
    - zoneId: "zone-de-frankfurt"
    - zoneId: "zone-de-berlin"
  guestPartnerCredentials:
    clientId: edgestream-client
---
# GUEST role - deploying to Telefonica
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Federation
metadata:
  name: fed-telefonica
  namespace: federation
  labels:
    opg.ewbi.nby.one/federation-relation: guest  # <-- GUEST
    opg.ewbi.nby.one/id: "dt-telefonica-2024"
spec:
  originOP:
    countryCode: "DE"
    mobileNetworkCodes:
      mcc: "262"
      mncs: ["01"]
  partner:
    statusLink: "https://opg-ewbi.telefonica.com"
    callbackCredentials:
      clientId: "dt-client-2024"
      tokenUrl: "https://idp.telefonica.com/token"
```

---

## Topology 4: Chain Federation (A → B → C)

Federations can be chained, but resources do NOT automatically propagate.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         CHAIN FEDERATION                                     │
│                                                                              │
│    ┌───────────────┐     ┌───────────────┐     ┌───────────────┐           │
│    │   EdgeStream  │     │  Telefonica   │     │    Vodafone   │           │
│    │               │     │               │     │               │           │
│    │   Federation: │     │   Federation: │     │   (HOST only) │           │
│    │   Telefonica  │────►│   EdgeStream  │     │               │           │
│    │   (guest)     │     │   (host)      │     │               │           │
│    │               │     │               │     │               │           │
│    └───────────────┘     │   Federation: │     │   Federation: │           │
│                          │   Vodafone    │────►│   Telefonica  │           │
│                          │   (guest)     │     │   (host)      │           │
│                          │               │     │               │           │
│                          └───────────────┘     └───────────────┘           │
│                                                                              │
│    EdgeStream → Telefonica → Vodafone                                       │
│                                                                              │
│    IMPORTANT: This does NOT mean EdgeStream apps automatically propagate    │
│    to Vodafone. Each federation is independent.                             │
│                                                                              │
│    If Telefonica wants EdgeStream's apps on Vodafone, they must:            │
│    1. Create their own File, Artefact, Application, ApplicationInstance    │
│    2. Deploy to Vodafone as a separate operation                            │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Key Point:** Federation chains do NOT automatically propagate resources. Each hop is independent and requires explicit re-deployment.

---

## Topology 5: Mesh Federation

Multiple operators federated with each other in a mesh pattern.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         MESH FEDERATION                                      │
│                                                                              │
│              ┌───────────────────────────────────────────┐                  │
│              │                                           │                  │
│              │         ┌───────────────┐                 │                  │
│              │         │  Telefonica   │                 │                  │
│              │         │               │                 │                  │
│              │    ┌───►│  HOST for: DT │◄───┐            │                  │
│              │    │    │  GUEST to: DT │    │            │                  │
│              │    │    └───────────────┘    │            │                  │
│              │    │                         │            │                  │
│              │    │                         │            │                  │
│    ┌─────────┴────┴───┐             ┌───────┴────────────┴─┐                │
│    │ Deutsche Telekom │             │      Orange          │                │
│    │                  │◄───────────►│                      │                │
│    │  HOST for: TEF   │             │  HOST for: TEF, DT   │                │
│    │  GUEST to: TEF   │             │  GUEST to: TEF, DT   │                │
│    └──────────────────┘             └──────────────────────┘                │
│                                                                              │
│    Each operator can deploy to any other operator they're federated with    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Example: Orange federated with both Telefonica and DT:**
```yaml
# Orange's namespace

# HOST - receiving from Telefonica
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Federation
metadata:
  name: fed-telefonica-incoming
  labels:
    opg.ewbi.nby.one/federation-relation: host
spec:
  offeredAvailabilityZones:
    - zoneId: "zone-fr-paris"
---
# HOST - receiving from Deutsche Telekom
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Federation
metadata:
  name: fed-dt-incoming
  labels:
    opg.ewbi.nby.one/federation-relation: host
spec:
  offeredAvailabilityZones:
    - zoneId: "zone-fr-paris"
    - zoneId: "zone-fr-lyon"
---
# GUEST - deploying to Telefonica
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Federation
metadata:
  name: fed-telefonica-outgoing
  labels:
    opg.ewbi.nby.one/federation-relation: guest
spec:
  partner:
    statusLink: "https://opg-ewbi.telefonica.com"
---
# GUEST - deploying to Deutsche Telekom
apiVersion: opg.ewbi.nby.one/v1beta1
kind: Federation
metadata:
  name: fed-dt-outgoing
  labels:
    opg.ewbi.nby.one/federation-relation: guest
spec:
  partner:
    statusLink: "https://opg-ewbi.telekom.de"
```

---

## How Resources Are Isolated

Resources are linked to their federation via the `federation-context-id` label:

```yaml
# Resource for Telefonica federation
apiVersion: opg.ewbi.nby.one/v1beta1
kind: File
metadata:
  name: my-app-image
  labels:
    opg.ewbi.nby.one/federation-context-id: "ctx-telefonica-123"  # Links to Telefonica
---
# Resource for Deutsche Telekom federation
apiVersion: opg.ewbi.nby.one/v1beta1
kind: File
metadata:
  name: my-app-image-dt
  labels:
    opg.ewbi.nby.one/federation-context-id: "ctx-dt-456"  # Links to DT
```

The operator uses this label to:
1. Look up the correct Federation CR
2. Get the HOST's API endpoint from `spec.partner.statusLink`
3. Get credentials from `spec.guestPartnerCredentials`
4. Route the API call to the correct HOST

---

## Summary

| Question | Answer |
|----------|--------|
| **Peer-to-peer only?** | No - supports many-to-many |
| **One HOST, multiple GUESTs?** | Yes - each GUEST gets separate Federation CR on HOST |
| **One GUEST, multiple HOSTs?** | Yes - GUEST creates separate Federation CRs for each HOST |
| **Same operator as both GUEST and HOST?** | Yes - `relation` label is per-CR, not per-cluster |
| **Chain federation (A→B→C)?** | Technically yes, but requires manual re-deployment at each hop |
| **Mesh federation?** | Yes - each pair needs bi-directional Federation CRs |
| **Resources isolated between federations?** | Yes - via `federation-context-id` label |

---

## Implementation Notes

### How the Operator Determines Role

From `internal/controller/util.go`:
```go
// returns true if LabelValue is v1beta1.FederationRelationGuest
func IsGuestResource(labels map[string]string) bool {
    return labels[v1beta1.FederationRelationLabel] == string(v1beta1.FederationRelationGuest)
}
```

The decision is made **per-resource based on the label**, not globally.

### How Multiple Clients Are Managed

From `internal/opg/opg.go`:
```go
type OPGClientsMap struct {
    opgClients map[string]opgc.ClientWithResponsesInterface  // keyed by fedId
    mutex      *sync.Mutex
}

func (m *OPGClientsMap) GetOPGClient(fedId, url, client string) opgc.ClientWithResponsesInterface {
    // Creates a new client per federation ID if it doesn't exist
    // Each client has its own URL and credentials
}
```

This allows the operator to maintain separate HTTP clients for each federation relationship.

---

## Related Documentation

- [INTENT.md](./INTENT.md) - Resource types and examples
- [ARCHITECTURE.md](./ARCHITECTURE.md) - Technical architecture
- [INSTRUCTIONS.md](./INSTRUCTIONS.md) - Local development setup
