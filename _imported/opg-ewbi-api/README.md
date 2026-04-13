# Neonephos OPG EWBI API

A Go-based East-West Bound Interface API implementation.
Work in bundle together with Nearby Computing OPG EWBI Operator
You can find the helm chart to deploy both services in [OPG EWBI Operator](github.com/nbycomp/opg-ewbi-operator)

## ⚠️ Under development

**IMPORTANT**: This solution is a work in progress

## Overview

This repository contains:
- **Federation API**: A REST API server implementing the EWBI federation protocol
- **API Code Generation**: Tools for generating Go client/server code from OpenAPI specifications

## Prerequisites

- Docker and Docker Compose

## Build images

This file points to a registry image. Please modify according to your needs
Especially

```bash
image: registry.example.com/nearbyone/ewbi-opg-federation-api:neonephos
platform: linux/arm64
```

### Build standard image

Using docker-compose:
```bash
docker-compose build federation
```

Using Docker directly:
```bash
docker build --platform=linux/arm64 -t ghcr.io/neonephos-katalis/opg-ewbi-api:neonephos --secret id=netrc,src=$HOME/.netrc .
```

### Build debug image

**For debugging purposes**, you can build a debug image with bash and troubleshooting tools (curl, wget, tcpdump, dig, etc.):

Using docker-compose to only build:
```bash
docker-compose build federation-debug
```

Using Docker directly:
```bash
docker build --target debug --platform=linux/arm64 -t ghcr.io/neonephos-katalis/opg-ewbi-api:neonephos-debug --secret id=netrc,src=$HOME/.netrc .
```

## Regenerate API code

To regenerate API code after specification changes:

```bash
docker-compose build apigen
```

## Setting up .netrc (to build using private repositories)

Create a `.netrc` file in your home directory (`~/.netrc` on macOS/Linux) with the following format:

```
machine registry.example.com
login your-username
password your-token
```

Make sure the file has appropriate permissions:
```bash
chmod 600 ~/.netrc
```

## API Code Generator (apigen)

Generates Go client and server code from OpenAPI specifications using `oapi-codegen`.

```bash
# Run the API code generator
docker-compose up apigen
```

This service:
- Processes the OpenAPI specification in `api/federation/FederationApi_v1.3.0.yaml`
- Generates client code in `api/federation/client/`
- Generates server code in `api/federation/server/`
- Generates model definitions in `api/federation/models/`
- Applies necessary fixes for known oapi-codegen issues



## Project Structure

```
.
├── api/
│   ├── apigen.sh              # API code generation script
│   └── federation/
│       ├── FederationApi_v1.3.0.yaml  # OpenAPI specification
│       ├── client/            # Generated client code
│       ├── server/            # Generated server code
│       └── models/            # Generated model definitions
├── cmd/app/                   # Application entry point
├── pkg/                       # Package libraries
├── docker-compose.yaml        # Docker Compose configuration
├── Dockerfile                 # Federation service Docker image
└── Dockerfile.apigen          # API generator Docker image
```

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request. All PRs must follow conventional commits and the PR template.

## License

See [LICENSE](LICENSE) file for details.
