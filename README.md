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
Install operator in host namespace, set API nodeport and set CRD to true to also install CRDs NodePorts are exposed in case testing outside of the cluster is needed. 

**No changes to values.yaml (dist/chart/values.yaml) are required.**

1. Download the OPG-EWBI-OPERATOR folder: https://github.com/neonephos-katalis/opg-ewbi-operator
2. After the download, open this folder via terminal and exec the following command: 
  ```make docker-build-controller ```
      **or**
  ```docker build . --no-cache -t ghcr.io/neonephos-katalis/opg-ewbi-operator:neonephos ```
3. Download the OPG-EWBI-API folder: https://github.com/neonephos-katalis/opg-ewbi-api
4. After the download, open this folder via terminale and exec the following command:
  ```docker-compose build federation --no-cache ```
   **or**
  ```docker compose build federation --no-cache ```
6. In your cluster create a new namesapce (e.g. ```bash kubectl create ns federation ```) after this exec this command. (replace $username and $accessToken with your username and accessToken used for the docker login ghcr.io command)   
  ```bash
      kubectl -n federation create secret docker-registry opg-registry-secret \
      --docker-server=ghcr.io \
      --docker-username= $username \
      --docker-password= $accessToken
  ```
6. Becouse the images alle pull in a registry (ghcr.io): ghcr.io/ipcei-egate-federation/opg-ewbi-operator:neonephos for the controller and ghcr.io/ipcei-egate-federation/ewbi-opg-federation-api:neonephos for api we need to exec this command for the login: ```docker login ghcr.io```, during this command, you will asked to provide your your GITHUB USERNAME and GITHUB PERSONAL ACCESS TOKEN.
7. In the end exec this command (in **OPG-EWBI-OPERATOR folder** via terminal)
  ```bash
  helm install federation-manager dist/chart -n federation \
  --set federation.services.federation.nodePort=30080 \
  --set crd.enable=true
  ```
After this command the OPG-EWBI-CONTROLLER and OPG-EWBI-API, if everything goes well, they will be available.
By running the command kubectl get pods -A, you should see two pods running in the federation namespace with names like nearbyone-federation-api-XXX and opg-ewbi-operator-controller-manager-XXX.

Use the following commands only if we you want to push the latest version of the controller and the api in github pacakge
  ```bash
  docker push ghcr.io/neonephos-katalis/opg-ewbi-operator:neonephos
  docker push ghcr.io/neonephos-katalis/opg-ewbi-api:neonephos
  ```

The Nearby code is written to work in both role (HOST and GUEST).
If you want test in local, you need two helm installation one for the host and one for the guest, use the following configuration of the helm command, but don't forget to follow the step 5-6 in both namespace (federation-host and federation-guest)

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
**Choose the names you prefer for namespaces and helm; those listed here are provided as examples.**


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

