#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

CLUSTER_NAME="federation-test"

echo -e "${YELLOW}=== OPG EWBI Federation Manager Cleanup ===${NC}"
echo ""

# Delete federation resources
echo -e "${YELLOW}[1/4] Deleting federation resources...${NC}"
kubectl delete -f config/samples/federationGuest.yaml 2>/dev/null || true
kubectl delete -f config/samples/federationHostAuth.yaml 2>/dev/null || true
echo -e "${GREEN}Done.${NC}"

# Uninstall Helm releases
echo ""
echo -e "${YELLOW}[2/4] Uninstalling Helm releases...${NC}"
helm uninstall federation-guest -n federation-guest 2>/dev/null || true
helm uninstall federation-host -n federation-host 2>/dev/null || true
echo -e "${GREEN}Done.${NC}"

# Delete namespaces
echo ""
echo -e "${YELLOW}[3/4] Deleting namespaces...${NC}"
kubectl delete namespace federation-guest 2>/dev/null || true
kubectl delete namespace federation-host 2>/dev/null || true
echo -e "${GREEN}Done.${NC}"

# Delete Kind cluster
echo ""
echo -e "${YELLOW}[4/4] Deleting Kind cluster '${CLUSTER_NAME}'...${NC}"
if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    kind delete cluster --name "$CLUSTER_NAME"
    echo -e "${GREEN}Kind cluster deleted.${NC}"
else
    echo -e "${YELLOW}Kind cluster '${CLUSTER_NAME}' not found.${NC}"
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Cleanup complete!${NC}"
echo -e "${GREEN}========================================${NC}"
