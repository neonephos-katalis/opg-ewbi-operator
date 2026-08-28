# EWBI Federation Documentation

## Introduction

This documentation describes the EWBI Federation Platform and provides an introduction to how it enables independent operator platforms to cooperate.

The platform provides an implementation of the **GSMA East-West Bound Interface (EWBI)**, enabling operators to establish federation relationships and exchange information about edge resources and application workloads.

The documentation is intended for:

* Mobile Network Operators (MNOs)
* Telecommunications operators
* Edge platform providers
* Platform administrators
* Developers integrating with EWBI-enabled platforms

No previous federation experience is assumed.

This document provides a high-level introduction to the main concepts. More detailed information about the architecture, implementation and operation of the platform is provided in the other documentation.

---

## What is Federation?

Federation allows two independent operator platforms to cooperate while maintaining control over their own infrastructure and operations.

Within a federation, one operator can make edge capabilities available to another operator. This can allow a partner to discover available resources, onboard applications and request workloads to be deployed on infrastructure operated by the other organisation.

A typical federation therefore involves two roles:

* A **Guest Operator**, which consumes capabilities from a partner.
* A **Host Operator**, which provides capabilities and hosts workloads for a partner.

The operators remain administratively independent while using agreed interfaces to exchange the information required to cooperate.

For the detailed federation model and operational workflows, see [Federation Model and Workflows](federation.md).

---

## What is EWBI?

**EWBI (East-West Bound Interface)** is a GSMA-defined set of interfaces for interoperability between Operator Platforms.

The EWBI APIs provide a standardised way for participating operators to exchange federation information and perform operations such as:

* Establishing federation relationships
* Sharing Availability Zone information
* Onboarding application resources
* Requesting application deployments
* Exchanging status information

The EWBI Federation Platform implements these interfaces so that operators can participate in federation without needing to expose the internal implementation of their own platform.

The detailed implementation of the EWBI interfaces is described in [Core Components](components.md).

---

## Federation at a Glance

At a conceptual level, federation can be viewed as a Guest requesting and using capabilities provided by a Host.

```text
Guest Operator                         Host Operator

Establish federation  ──────────────►  Accept / establish federation

Discover zones        ◄──────────────  Offer Availability Zones

Select zones          ──────────────►  Make selected zones available

Onboard resources     ──────────────►  Receive application resources

Request deployment    ──────────────►  Deploy application

Receive status        ◄──────────────  Return application status
```

The diagram shows the overall relationship rather than the internal implementation.

The detailed sequence of these operations, including resource creation, status changes and callbacks, is described in [Federation Model and Workflows](federation.md).

---

## Main Platform Components

The federation platform is built around several main components.

### Kubernetes

Kubernetes provides the platform environment in which federation resources are represented and managed.

It provides the management interface used by operators and automation systems.

### EWBI Operator

The EWBI Operator provides the Kubernetes-native interface to federation.

It manages federation resources within Kubernetes and connects those resources to federation operations.

The detailed implementation of the Operator is described in [Core Components](components.md).

### EWBI API

The EWBI API provides the external interface used for communication between federated operators.

It implements the EWBI interfaces and allows an operator to receive and process federation requests from a partner.

### Federation Custom Resources

Federation concepts are represented as Kubernetes Custom Resources.

These resources provide the Kubernetes representation of objects such as federations, applications and application instances.

The complete resource model is described in [Core Components](components.md) and the behaviour of those resources is described in [Federation Model and Workflows](federation.md).

### Local Orchestrator

The local orchestrator is responsible for executing workloads within an operator's infrastructure.

It is **operator-specific** and is not defined by the EWBI interface itself. Different operators may therefore use different orchestration platforms.

The federation platform provides the information required for workload management, while the local operator remains responsible for how workloads are actually executed.

---

## Federation Roles

A federation relationship has two logical roles.

### Guest Operator

The Guest Operator consumes capabilities provided by a federation partner.

The Guest can use the federation relationship to discover available zones, make resources available to its applications and request workloads to be deployed by the Host Operator.

### Host Operator

The Host Operator provides capabilities to its federation partner.

The Host makes Availability Zones available and provides the infrastructure on which partner workloads can be deployed.

These roles describe how an operator participates in a particular federation relationship. They do not necessarily imply a different set of federation components.

---

## Key Concepts

### Federation

A **Federation** represents a relationship between two independent Operator Platforms.

It provides the context in which resources and operations are exchanged between the operators.

### Availability Zone

An **Availability Zone** represents a location where the Host Operator makes edge resources available to a federation partner.

A Guest can select the zones it wants to use within a federation.

### Federation Context

A **Federation Context** identifies a specific federation relationship between two operators.

The Federation Context identifier is associated with subsequent federation operations so that resources and requests can be linked to the appropriate federation relationship.

### File

A **File** represents an application image or other deployable file that can be made available to a federation partner.

Files provide the deployable content required by an application.

### Artefact

An **Artefact** describes information required to deploy a workload.

It can contain information such as workload components, images, resource requirements and deployment-related parameters.

### Application

An **Application** represents a logical application that is onboarded into the federation.

An application can reference the artefacts required to deploy its workload and contain application metadata and relevant requirements.

### Application Instance

An **Application Instance** represents a requested instance of an application running within a particular Availability Zone.

It connects an application to a specific deployment location and contains information required for that deployment.

The relationships and lifecycle of these resources are described in more detail in [Federation Model and Workflows](federation.md).

---

## What Happens Next?

Once the basic concepts are understood, the remaining documentation can be read according to what you need to understand or do:

* [Architecture](architecture.md) — Understand the overall system architecture and the relationships between its major components.
* [Core Components](components.md) — Understand the implementation and responsibilities of the individual components.
* [Federation Model and Workflows](federation.md) — Understand how federation operations, resource flows and status updates work.
* [Deployment](deployment.md) — Learn how to install and configure the federation components.
* [Connectivity](connectivity.md) — Understand the network and connectivity requirements between operators.
* [Troubleshooting](troubleshooting.md) — Verify a deployment and diagnose common problems.
