# EWBI Federation Documentation

## Introduction

This documentation describes the deployment, operation, and usage of the EWBI Federation Platform.

The platform provides an implementation of the GSMA East-West Bound Interface (EWBI), enabling Operator Platforms to establish federation relationships and securely share edge computing capabilities, resources, and application workloads across organisations.

The documentation is intended for:

- Mobile Network Operators (MNOs)
- Telecommunications operators
- Edge platform providers
- Platform administrators
- Developers integrating with EWBI-enabled platforms

No prior federation experience is assumed.

---

## What This Documentation Covers

This documentation focuses on the EWBI Federation platform implementation, including federation establishment, resource onboarding, application lifecycle management and operational procedures.

It does not attempt to replace the GSMA EWBI specification, which should be consulted for interface definitions and protocol requirements.


## What is Federation?

Federation enables two independent Operator Platforms to cooperate while maintaining administrative independence.

Through federation, one operator can:

- Discover resources offered by another operator
- Deploy applications into partner infrastructure
- Access partner Availability Zones
- Exchange lifecycle information
- Share edge capabilities using standardised APIs

EWBI defines the interfaces used to establish and manage these relationships.

---

## What is EWBI?

EWBI (East-West Bound Interface) is a GSMA-defined set of APIs that enables interoperability between Operator Platforms.

EWBI standardises:

- Federation establishment
- Availability Zone synchronisation
- Application onboarding
- Application deployment
- Resource management
- Edge discovery
- Status notifications and callbacks

The APIs allow participating operators to exchange information using a common interface, regardless of the underlying platform implementation.

---

## Federation at a Glance

At a high level, federation enables one operator to deploy and manage applications on infrastructure owned by another operator.

A typical workflow looks like:

Guest Operator                    Host Operator
────────────────────────────────────────────────────

Create Federation      ────────►  Validate Federation

Discover Zones         ────────►  Offer Availability Zones

Select Zones           ────────►  Expose Zone Resources

Upload Resources       ────────►  Receive Resources

Onboard Application    ────────►  Validate Application

Deploy Application     ────────►  Host Application Instance

Receive Updates        ◄────────  Send Status Callbacks



## How the Platform Works

The platform consists of two major components:

### EWBI API

The API service implements the EWBI interfaces defined by the GSMA specification.
It exposes REST endpoints used to:

- Create federations
- Exchange Availability Zone information
- Onboard applications
- Manage application lifecycle operations
- Exchange status notifications

### EWBI Operator

The Kubernetes Operator provides a cloud-native interface for managing federation resources.
The operator reconciles Kubernetes Custom Resources and translates them into EWBI API operations.
This allows federation actions to be managed using standard Kubernetes workflows.

---

## Federation Roles

A federation relationship involves two logical roles.

### Guest Operator

Consumes capabilities exposed by a partner operator and initiates federation operations.

### Host Operator

Provides capabilities and resources to federation partners and hosts deployed workloads.

---


## Key Concepts

Before deploying the platform, operators should understand the following concepts.

### Federation

A trusted relationship between two Operator Platforms.

### Availability Zone

An Availability Zone is a location where a Host Operator makes edge resources available to federation partners.

During federation establishment, a Host Operator offers one or more Availability Zones and a Guest Operator subscribes to those it wishes to use.

Application Instances are deployed within Availability Zones.


### Federation Context

Each federation relationship is assigned a unique Federation Context Identifier.
This identifier is used in subsequent operations such as:

- Zone subscription
- File onboarding
- Application onboarding
- Application deployment

It's a unique identifier for a particular federation relationship

## Resource Model

The EWBI resource model represents the dependencies between federation resources required to share and deploy applications across federated operators.

Each resource introduces additional information required before an application can be deployed in a partner Availability Zone.

The platform uses a hierarchical resource model.
Resources generally cannot be created out of sequence because higher-level resources depend on the existence of lower-level resources. The onboarding examples included with the project follow this ordering.

```text
Federation
    └── File
          └── Artefact
                └── Application
                      └── ApplicationInstance
```
Each resource builds upon the previous resource.

- A Federation establishes trust between operators.
- A File represents an image that can be deployed.
- An Artefact describes how a workload should be deployed.
- An Application defines a logical service.
- An Application Instance represents a running deployment of an Application within a specific Availability Zone.

## State-Based Operations and Reconciliation

The platform uses Kubernetes Operators to manage federation resources.

When a Custom Resource (CR) is created, the operator:

1. Detects the change.
2. Validates the resource.
3. Calls the appropriate EWBI API.
4. Updates status information.
5. Processes asynchronous callbacks.

Federation operations are asynchronous.

Creating a resource does not necessarily mean the corresponding federation operation has completed successfully.

Operators should monitor resource status and reconciliation progress rather than assuming immediate completion.


### Why Kubernetes Custom Resources?

EWBI is defined as a REST API specification.

This platform adds a Kubernetes-native management layer by representing federation resources as Custom Resources (CRs).

Operators therefore manage federated resources using standard Kubernetes workflows, while the EWBI Operator handles communication with federation partners through the EWBI APIs.

This approach allows federation operations to be integrated into existing GitOps, automation and Kubernetes management processes.



## Callbacks

Callbacks allow long-running federation operations to complete asynchronously while keeping both operators informed of status changes.

Without callbacks, operators would need to continuously poll federation APIs to determine operation status.

Callbacks are used for:

- Federation status updates
- Availability Zone updates
- Application onboarding status
- Application Instance status

## EWBI API and Kubernetes Resources

The platform provides two ways to interact with federation functionality:

EWBI API
The standards-based REST API defined by the GSMA EWBI specification.

Kubernetes Custom Resources
The Kubernetes-native interface used by operators and automation systems.

The operator acts as the bridge between these two models.

## Documentation Structure

## Prerequisites

## Getting started