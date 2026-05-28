# opg-ewbi-operator

OPG EWBI Operator and API — implements a subset of the OPG East/WestBound Interface. The operator translates requests to k8s CRD resources, while the API server provides the REST interface.

## ⚠️ Under development

**IMPORTANT**: This solution is a work in progress

## Description

This repository is a unified monorepo containing both:
- **OPG EWBI API** — A REST API server implementing the GSMA OPG Federation API (EWBI protocol)
- **OPG EWBI Operator** — A Kubernetes operator that reconciles CRDs and interacts with partner OPs

The system implements a subset of the GSMA OPG East/WestBound Interfaces, including a k8s Operator mapping the OPG objects to k8s CRD instances. It relies on other components (e.g. NearbyOne's Okto orchestrator) also monitoring the generated CRs to trigger the actions required to deploy the requested applications on the federated cluster.

## Funding and support

This open source project is part of activities carried out within the Important Project of Common European Interest on Next Generation Cloud Infrastructure and Services (IPCEI-CIS), an EU initiative to build a sovereign, interoperable and energy-efficient cloud‑to‑edge infrastructure in Europe.

<p align="center">
  <img src="assets/eu-funded-nextgenerationeu.png" alt="EU funding logo" width="300" />
</p>

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request. All PRs must follow conventional commits and the PR template.

## Getting Started

### Prerequisites

- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

For local end-to-end testing using Kind, see [docs/manual-testing-kind.md](docs/manual-testing-kind.md) (full Guest ↔ Host flow) and [docs/manual-callbacks-testing-kind.md](docs/manual-callbacks-testing-kind.md) (Host → Guest callback flow).

### Configuration: platform ARM64 or platform AMD64
This code is designed to run on both ARM64 and AMD64 platforms, but to enable this, some changes need to be made to the following files: values.yaml in (dist/chart) and the Makefile.

1. For the **value.yaml** in OPG-EWBI-OPERATOR (**dist/chart**), you need to change the repository parameter values (line 7 and line 49) as follows:

    **For Linux/ARM64**:
    - Line 7: ```ghcr.io/neonephos-katalis/opg-ewbi-operator```
    - Line 49: ```ghcr.io/neonephos-katalis/opg-ewbi-api```

    **For Linux/AMD64**:
    - Line 7: ```ghcr.io/neonephos-katalis/opg-ewbi-operator-amd64```
    - Line 49: ```ghcr.io/neonephos-katalis/opg-ewbi-api-amd64```

2. For the **Makefile** in OPG-EWBI-OPERATOR, you need to change lines 1-2 as follows:

    **For Linux/ARM64**:
    - Line 1: ```IMG ?= ghcr.io/neonephos-katalis/opg-ewbi-operator:neonephos```
    - Line 2: ```PLATFORM ?= linux/arm64```

    **For Linux/AMD64**:
    - Line 1: ```IMG ?= ghcr.io/neonephos-katalis/opg-ewbi-operator-amd:neonephos```
    - Line 2: ```PLATFORM ?= linux/amd64```

## Deploy the federation manager
Install operator in host namespace, set API nodeport and set CRD to true to also install CRDs NodePorts are exposed in case testing outside of the cluster is needed.

**⚠️ Currently, the files are set for Linux/arm64 platforms. If you need to build images for Linux/amd64 platforms, follow the previously described steps (Configuration: platform ARM64 or platform AMD64)**

1. Create a `.netrc` file in your home directory (`~/.netrc` on macOS/Linux) with the following format:

    ```
    machine ghcr.io
    login your-username
    password your-token
    ```

    Make sure the file has appropriate permissions:
    ```bash
    chmod 600 ~/.netrc
    ```
2. ```git clone https://github.com/neonephos-katalis/opg-ewbi-operator```
3. After the download, open this folder via terminal and exec the following command:

  Build the **operator** image:
  ```make docker-build-controller```
      **or**
  ```docker build . --no-cache -t ghcr.io/neonephos-katalis/opg-ewbi-operator:neonephos --secret id=netrc,src=$HOME/.netrc .```

  **For debugging purposes**, you can build a debug image with bash and troubleshooting tools (curl, wget, tcpdump, dig, etc.):
  ```make docker-build-debug```
      **or**
  ```docker build --target debug --no-cache -t ghcr.io/neonephos-katalis/opg-ewbi-operator:neonephos-debug --secret id=netrc,src=$HOME/.netrc .```

  To use the debug image, install the chart with `--set image.tag=neonephos-debug` and exec into the pod with `kubectl exec -it <pod-name> -- bash`

4. Build the **API** image:

  ```make docker-build-api```
      **or**
  ```docker build -f Dockerfile.api --no-cache -t ghcr.io/neonephos-katalis/opg-ewbi-api:neonephos --secret id=netrc,src=$HOME/.netrc .```

5. ```docker login ghcr.io```
6. In your cluster create a new namespace (e.g. ```kubectl create ns federation```) after this exec this command. (replace $username and $accessToken with your username and accessToken used for the docker login ghcr.io command)
  ```bash
      kubectl -n federation create secret docker-registry opg-registry-secret \
      --docker-server=ghcr.io \
      --docker-username= $username \
      --docker-password= $accessToken
  ```
7. In the end exec this command (in the **project root folder** via terminal)
  ```bash
  helm install federation-manager dist/chart -n federation \
  --set federation.services.federation.nodePort=30080 \
  --set crd.enable=true
  ```
After this command the OPG-EWBI-CONTROLLER and OPG-EWBI-API, if everything goes well, they will be available.
By running the command kubectl get pods -A, you should see two pods running in the federation namespace with names like nearbyone-federation-api-XXX and opg-ewbi-operator-controller-manager-XXX.

Use the following commands only if we you want to push the latest version of the controller and the api in github pacakge

**For Linux/ARM64:**
```bash
docker push ghcr.io/neonephos-katalis/opg-ewbi-operator:neonephos
docker push ghcr.io/neonephos-katalis/opg-ewbi-api:neonephos
```

**For Linux/AMD64:** **⚠️ In this moment the images are not push on GitHub please exec the following commands**
```bash
docker push ghcr.io/neonephos-katalis/opg-ewbi-operator-amd:neonephos
docker push ghcr.io/neonephos-katalis/opg-ewbi-api-amd:neonephos
```

**For Linux/ARM64:**
```bash
docker push ghcr.io/neonephos-katalis/opg-ewbi-operator-arm64:v1.0.0
docker push ghcr.io/neonephos-katalis/opg-ewbi-api-arm64:v1.0.0
```

**For Linux/AMD64:** **NOW IMAGES AVAILABLE**
```bash
docker push ghcr.io/neonephos-katalis/opg-ewbi-operator-amd64:v1.0.0
docker push ghcr.io/neonephos-katalis/opg-ewbi-api-amd64:v1.0.0
```

The Nearby code is written to work in both role (HOST and GUEST).
If you want test in local, you need two helm installation one for the host and one for the guest, use the following configuration of the helm command, but don't forget to follow the step 8 for both namespaces (katalis-dev-host and katalis-dev-guest)

```bash
helm install federationhost dist/chart -n katalis-dev-host \
  --set federation.services.federation.nodePort=30080 \
  --set crd.enable=true
```

We set crd to false since we have already installed them

```bash
helm install federationguest dist/chart -n katalis-dev-guest \
--set federation.services.federation.nodePort=30081 \
--set crd.enable=false
```
**Choose the names you prefer for namespaces and helm; those listed here are provided as examples.**


### Test:

Create Client ID in identity provider and store in a federation on the host. This is not yet the establishment of the federation: This is done to link the federation to the clientID.

It is named ..Auth because it represents identity provider integration and representation in the cluster.

```bash
kubectl apply -f config/samples/federationHostAuth.yaml
```

Federation establishment: ON GUEST
```bash
kubectl apply -f config/samples/federationGuest.yaml
```

In federationGuest, you will see we define the URL of the EWBI API of the host.

Example:
http://nearbyone-federation-api.katalis-dev-host.svc.cluster.local:8080
(and the callback url. Bear in mind the callback flow is not complete)

You should see in the federation status the FederationContextId. Copy it and use this for the next CRs.
(FederationContextId uuid is currently generated from ClientID, so if you don't change the examples, they should work without modifying)

```bash
kubectl apply -f config/samples/fileGuest.yaml
kubectl apply -f config/samples/artefactGuest.yaml
kubectl apply -f config/samples/appGuest.yaml
kubectl apply -f config/samples/appInstGuest.yaml
```

You will see that, for each specific CR created in the guest, a new one is mirrored in the host, all happenning through the EWBI API.
Same will happen if you remove them, even federation.

If you delete guest ones, mirrored host resources will get deleted.

```bash
kubectl delete -f config/samples/fileGuest.yaml
kubectl delete -f config/samples/artefactGuest.yaml
kubectl delete -f config/samples/appGuest.yaml
kubectl delete -f config/samples/appInstGuest.yaml
```
## Regenerate API Code

To regenerate Go client/server code after OpenAPI specification changes:

```bash
make apigen
```

This generates code from `api/ewbi/FederationApi_v1.3.0.yaml` into:
- `api/ewbi/client/` — Generated client code
- `api/ewbi/server/` — Generated server code
- `api/ewbi/models/` — Generated model definitions

## Project Structure

```
.
├── api/
│   ├── ewbi/                          # EWBI Federation API (generated + specs)
│   │   ├── FederationApi_v1.3.0.yaml  # OpenAPI specification
│   │   ├── apigen.sh                  # Code generation script
│   │   ├── client/                    # Generated client code
│   │   ├── server/                    # Generated server code
│   │   └── models/                    # Generated model definitions
│   └── operator/
│       └── v1beta1/                   # CRD type definitions
├── cmd/
│   ├── main.go                        # Operator entry point
│   └── api/
│       └── main.go                    # API server entry point
├── internal/                          # Operator internals
│   ├── config/                        # API server configuration
│   ├── controller/                    # Reconcilers
│   ├── indexer/                       # Field indexers
│   ├── multipart/                     # Multipart form helpers
│   ├── opg/                           # OPG client cache
│   └── options/                       # Namespace config
├── pkg/                               # API server packages
│   ├── handler/                       # HTTP handlers
│   ├── metastore/                     # K8s-backed persistence
│   ├── deployment/                    # Install/uninstall logic
│   └── ...
├── config/                            # Kustomize manifests
├── dist/chart/                        # Helm chart
├── Dockerfile                         # Operator image
├── Dockerfile.api                     # API server image
├── Dockerfile.apigen                  # Code generator image
└── Makefile                           # Build targets
```
