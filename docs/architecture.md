# EWBI Federation Platform Architecture

# Purpose

This document provides a high-level architectural overview of the EWBI Federation Platform.

It explains:

- The overall system architecture
- The major platform components
- Component responsibilities
- How operators interact
- The relationship between Kubernetes resources and the EWBI APIs
- The architectural boundaries of the platform

This document intentionally focuses on architecture and component relationships.

Detailed federation workflows, onboarding procedures, resource lifecycles, state synchronisation and deployment processes are described in their respective documentation.

---

# System Context

The EWBI Federation Platform extends existing operator platforms with standards-based federation capabilities.

Each participating operator deploys an identical federation stack consisting of the EWBI API and EWBI Operator. These components enable operators to establish federation relationships and exchange federation resources using the GSMA EWBI interfaces. (Operators may deploy the federation manager components with slightly different applications to manage them, e.g. K8s native, OpenShift, NearbyOne)

The platform integrates with existing orchestration systems and does not replace local workload management platforms.

At a high level, the platform consists of:

- A Kubernetes cluster
- An EWBI API implementing the GSMA specification
- A Kubernetes Operator providing a Kubernetes-native interface
- Kubernetes Custom Resources representing federation objects
- An orchestration platform responsible for workload execution

The platform follows a distributed peer-to-peer model. There is no central federation controller or broker.

---

# Architectural Principles

The platform is designed around several key architectural principles.

## Federation Through Standard Interfaces

Operators communicate using GSMA-defined EWBI APIs while retaining complete administrative independence.

Federation does not require operators to expose internal platform implementations or relinquish control of local resources.

## Kubernetes-Native Management

Federation resources are represented as Kubernetes Custom Resources (CRs).

This enables operators to manage federation using familiar Kubernetes tooling while integrating naturally with automation systems and GitOps workflows.

## Declarative Reconciliation

Desired federation state is expressed through Kubernetes resources.

The EWBI Operator continuously reconciles the desired state with the actual federation state by communicating with partner operators through the EWBI APIs.

_(To be updated with Kubernetes communication model changes introduced in Phase 2.)_

## Asynchronous Operations

Many federation operations involve communication between independent platforms and therefore complete asynchronously.

Resource creation initiates federation workflows, while status updates and callbacks communicate progress and completion.

---

# High-Level Architecture

The platform is composed of several logical components deployed within each participating operator environment.

```text
                   Operator Platform A

+-------------------------------------------------------+
|                                                       |
|               Kubernetes Cluster                      |
|                                                       |
|  +-----------------------------------------------+    |
|  | Kubernetes API                                |    |
|  +----------------------+------------------------+    |
|                         |                             |
|                  EWBI Operator                       |
|                         |                             |
|                     EWBI API                         |
|                         |                             |
|              Local Orchestration Platform            |
|                                                       |
+-------------------------+-----------------------------+
                          |
                   GSMA EWBI REST APIs
                          |
+-------------------------+-----------------------------+
|                                                       |
|               Kubernetes Cluster                      |
|                                                       |
|  +-----------------------------------------------+    |
|  | Kubernetes API                                |    |
|  +----------------------+------------------------+    |
|                         |                             |
|                  EWBI Operator                       |
|                         |                             |
|                     EWBI API                         |
|                         |                             |
|              Local Orchestration Platform            |
|                                                       |
+-------------------------------------------------------+

                   Operator Platform B
```

Each operator deploys the same federation components.

Communication between operators occurs exclusively through the EWBI APIs, while each operator remains responsible for managing its own infrastructure and operational policies.

---

# Architectural Decisions

## Why Kubernetes Operators?

The GSMA EWBI specification defines federation using REST APIs.

The platform introduces a Kubernetes-native management model through the Kubernetes Operator pattern.

This allows federation resources to be managed using standard Kubernetes workflows while maintaining compatibility with the EWBI specification.

Benefits include:

- Declarative resource management
- Event-driven lifecycle handling
- Native Kubernetes integration
- GitOps compatibility
- Integration with automation systems

## Why Custom Resources?

Federation concepts are represented as Kubernetes Custom Resources.

This allows operators to manage federation resources using standard Kubernetes tools, APIs and operational procedures.

Resources become Kubernetes objects that can be managed through familiar workflows.

## Why Peer-to-Peer Federation?

The platform is intended to support federation between independent organisations.

There is no central federation controller, broker or shared operational platform.

Each operator deploys an equivalent federation stack and communicates directly with federation partners through the EWBI APIs.

This approach preserves administrative independence while providing interoperability.

---

# Architecture Components

This section provides a high-level overview of the primary architectural components.

Detailed implementation behaviour is described in the Components documentation.

## Local Orchestration Platform

The federation platform integrates with an operator's existing orchestration platform.

This platform remains responsible for infrastructure management and workload execution, while the federation platform coordinates resource sharing and cross-operator operations.

Typical responsibilities include:

- Resource scheduling
- Workload placement
- Infrastructure management
- Application lifecycle management
- Platform-specific operations

The federation platform does not replace these capabilities.

---

## Kubernetes

Kubernetes provides the management interface used to administer federation resources.

Operators interact with the platform by creating and managing Kubernetes Custom Resources that represent federation objects.

Using Kubernetes enables federation operations to integrate with existing automation systems, CI/CD pipelines and GitOps workflows.

---

## EWBI Operator

The EWBI Operator provides the Kubernetes-native interface to federation.

It continuously watches Custom Resources and reconciles the desired state with the remote federation platform.

Its responsibilities include:

- Watching Kubernetes Custom Resources
- Validating resource changes
- Executing reconciliation loops
- Translating Kubernetes resources into EWBI operations
- Updating resource status
- Processing asynchronous callbacks

The Operator acts as the bridge between Kubernetes and the EWBI APIs.

---

## EWBI API

The EWBI API implements the GSMA-defined federation interfaces.

It exposes REST endpoints used by partner operators to perform federation operations.

Its responsibilities include:

- Exposing EWBI endpoints
- Receiving federation requests
- Validating requests
- Processing federation operations
- Sending callbacks
- Returning operation status

All communication between operators occurs through this component.

---

## Kubernetes Custom Resources

Federation resources are represented as Kubernetes Custom Resources.

These resources provide a declarative representation of federation objects and allow operators to manage federation using standard Kubernetes tooling.

Examples include:

- Federation
- File
- Artefact
- Application
- ApplicationInstance

The Operator translates these resources into the corresponding EWBI operations.

---

# Deployment Model

The platform follows a distributed deployment model.

Each participating operator deploys:

- A Kubernetes cluster
- The EWBI API
- The EWBI Operator
- Federation Custom Resources

Operators communicate directly through the GSMA-defined EWBI APIs.

Within a federation relationship, an operator may assume:

- A Guest role
- A Host role

These are logical federation roles and do not affect the deployed architecture.

---

# Component Relationships

The following diagram illustrates the relationship between the major components.

```text
User / Automation
        │
        ▼

Kubernetes API
        │
        ▼

EWBI Operator
        │
        ▼

EWBI API
        │
        ▼

Partner EWBI API
        │
        ▼

Partner EWBI Operator
        │
        ▼

Partner Kubernetes Platform
```

Users and automation systems interact with Kubernetes through Custom Resources.

The EWBI Operator monitors these resources and determines when federation operations need to be performed.

The Operator communicates with the local EWBI API, which exchanges federation information with partner operators using the GSMA-defined EWBI interfaces.

Responses and asynchronous callbacks are reflected back into Kubernetes resources, ensuring resource status accurately represents the current federation state.

This separation of responsibilities allows Kubernetes to remain the primary management interface while the EWBI APIs provide standards-based interoperability between operators.

---

# Architecture Boundaries

The EWBI Federation Platform is responsible for:

- Federation establishment
- Resource exchange
- Cross-operator communication
- Federation lifecycle coordination
- Status management

The platform is not responsible for:

- Kubernetes cluster management
- Infrastructure provisioning
- Virtual machine lifecycle management
- Network provisioning
- Application scheduling
- Application runtime execution

These responsibilities remain within the local operator platform.

---

# Related Documentation

- [Quick Start Guide](Quick-start-introduction.md)
- [Core Components](components.md)
- [Federation Model and Workflows](federation.md)
- [Security](security.md)
- [Diagrams](diagrams.md)
- [Deployment Guide](deployment.md)
- [Connectivity Guide](connectivity.md)
- [Troubleshooting Guide](troubleshooting.md)