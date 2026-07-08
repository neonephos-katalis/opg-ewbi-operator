# Context
This documentation will help describe to current and new members:
- How the federation system works internally
- How to install, configure, and onboard into the platform

# Goal
Define documentation sets that cover two distinct areas:
1. Federation Architecture & Mechanism
2. Platform Onboarding & Installation

This issue is focused on agreeing the scope, structure, and required content.


# 1. Federation Architecture & Mechanism
This section documents the internal design and behaviour of the federation system.

## Topics to cover

### Quick start guide
[Quick Start Guide](Quick-start-introduction.md)
Gives new joiners a quick overview of the project components and some brief introductions to it. 
- Quick start guide
- Required knowledge

### High-Level Architecture
[Architecture](architecture.md)
Describes the overall architecture of the Katalis platform and how the major systems interact with each other.
- High-level architecture diagram
- Architectural overview (components and their relationship amongst each other)

### Core components
[Core components](components.md)
- Host and guest cluster
- Federation manager
    - Operator
    - API
- CRDs
- Orchestrator

### System Behaviour
[Federation model and workflows](federation.md)
- Federation model
- Flows: 
    - Registration
    - Application deployment
    - CR state synchronisation model
    - Reconciliation
- How resources flow through the system
- Event propagation model
- State synchronisation model
- How components communicate
- Protocols used (Kubernetes API, REST, etc)
- Event/callback mechanisms

### Security
[Security](security.md)
- How can we ensure trust between operators
- How to ensure trust of deployed application

### Diagrams
[Diagrams](diagrams.md)
There should be clear diagrams for: 
- Overall architecture
- Resource flow (end-to-end lifecycle)
- Component interaction model
- Failure / retry flows


# 2. Platform Onboarding & Installation
## Topics to cover
### Onboarding overview 
[Onboarding Overview](onboarding.md)
- Onboarding overview
- Different onboarding paths, when to choose each approach
    - Native Kubernetes setup
    - NearbyOne platform setup (or managed platform variant)

### Installation of Federation Components
[Deployment](deployment.md)
- Installing the Federation Manager
- Required dependencies
- Helm charts / manifests / deployment methods

### Cluster Connectivity Setup
[Connectivity](connectivity.md)
- Connecting host and guest clusters
- Authentication and credentials setup
- Network requirements:
	- IP addresses
	- Ports
	- Protocols
	- Firewall considerations

### Verification & Health Checks
[Verification, health checks and troubleshooting](troubleshooting.md)
- How to confirm successful onboarding
- Basic test workflows
- Debugging connection issues

# Open Questions
- NearbyOne separate product documentation? 