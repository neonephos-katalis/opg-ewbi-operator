# Context
This documentation will help describe to current and new members how the federation system works internally.
This documentation only describes the federation manager component within the operator platform. 
The Federation Manager is not an end-to-end solution and the users of this software are expected to integrate the E/WBI Operator software in their environment with their edge resource management systems or resource orchestrators. 


# Goal
Federation Architecture & Mechanism

# Federation Architecture & Mechanism
This section documents the internal design and behaviour of the federation system.

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
- Event/callback mechanisms

### Security
[Security](security.md)
- How can we ensure trust between operators
- How to ensure trust of deployed application