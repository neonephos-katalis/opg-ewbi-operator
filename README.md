# opg-ewbi-operator

OPG EWBI Operator implements a subset of the OPG East/WestBound Interface and translates these requests to k8s CRD resources.

## ⚠️ Under development 

**IMPORTANT**: This solution is a work in progress

## Description

The repository implements a subset of the GSMA OPG East/WestBound Interfaces, including a k8s Operator mapping the OPG objects to k8s CRD instances, and viceversa. It relies on other components (e.g. NearbyOne's Okto orchestrator) also monitoring the generated CRs to trigger the actions required to deploy the requested applications on the federated cluster. And also the other way around, other components can create the CRs that would trigger the OPG East/Westbound Interface to interact with federated orchestrator to deploy applications externally.

## Getting Started

### Prerequisites

- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

## Quickstart: Deploy on development cluster, federating two namespaces

Install operator in host namespace, set API nodeport and set CRD to true to also install CRDs
NodePorts are exposed in case testing outside of the cluster is needed.

⚠️ These helm charts pull images from a registry, left as an example
Please modify the [values](dist/chart/values.yaml) for the images in order to deploy them.

```bash
helm install federationhost dist/chart -n federation-host \
  --set federation.services.federation.nodePort=30080 \
  --set crd.enable=true
```

We set crd to false since we have already installed them

```bash
helm install federationguest dist/chart -n federation-guest \
  --set federation.services.federation.nodePort=30081 \
  --set crd.enable=false
```

Prerequirement:

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
http://nearbyone-federation-api.federation-host.svc.cluster.local:8080
(and the callback url. Bear in mind the callback flow is not complete)

You should see in the federation status the FederationContextId. Copy it and use this for the next CRs.
(FederationContextId uuid is currently generated from ClientID, so if you don't change the examples, they should work without modifying)

```bash
kubectl apply -f config/samples/fileGuest.yaml
kubectl apply -f config/samples/artifactGuest.yaml
kubectl apply -f config/samples/appGuest.yaml
kubectl apply -f config/samples/appInstGuest.yaml
```

You will see that, for each specific CR created in the guest, a new one is mirrored in the host, all happenning through the EWBI API.
Same will happen if you remove them, even federation.

If you delete guest ones, mirrored host resources will get deleted.

```bash
kubectl delete -f config/samples/fileGuest.yaml
kubectl delete -f config/samples/artifactGuest.yaml
kubectl delete -f config/samples/appGuest.yaml
kubectl delete -f config/samples/appInstGuest.yaml
```

## Quickstart: build and push

Useful commands to build and push the binaries in this repo and to create the helm charts with CRDs.

Please check the Makefile and configure according to your needs, especially these:

```bash
# Image URL to use all building/pushing image targets
IMG ?= registry.example.com/operators/opg-ewbi-operator:neonephos
PLATFORM ?= linux/arm64
```

```bash
make docker-build-controller 
```

```bash
make docker-push-controller 
```

```bash
make helm 
```

This operator is built using kubebuilder, you can refer to official instructions on how to develop with it.

