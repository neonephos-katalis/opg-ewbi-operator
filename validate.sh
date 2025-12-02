#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

CLUSTER_NAME="federation-test"
PASSED=0
FAILED=0

echo -e "${GREEN}=== OPG EWBI Federation Manager Validation ===${NC}"
echo ""

# Helper function
check() {
    local name="$1"
    local cmd="$2"
    local expected="$3"

    echo -n "Checking $name... "
    result=$(eval "$cmd" 2>/dev/null)
    if echo "$result" | grep -q "$expected"; then
        echo -e "${GREEN}PASS${NC}"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        echo "  Expected: $expected"
        echo "  Got: $result"
        ((FAILED++))
        return 1
    fi
}

# Check Kind cluster exists
echo -e "${YELLOW}[1/8] Checking Kind cluster...${NC}"
check "Kind cluster exists" "kind get clusters" "$CLUSTER_NAME"

# Check kubectl context
echo ""
echo -e "${YELLOW}[2/8] Checking kubectl context...${NC}"
check "kubectl context" "kubectl config current-context" "kind-${CLUSTER_NAME}"

# Check CRDs installed
echo ""
echo -e "${YELLOW}[3/8] Checking CRDs...${NC}"
check "Federation CRD" "kubectl get crd federations.opg.ewbi.nby.one" "federations.opg.ewbi.nby.one"
check "Application CRD" "kubectl get crd applications.opg.ewbi.nby.one" "applications.opg.ewbi.nby.one"
check "ApplicationInstance CRD" "kubectl get crd applicationinstances.opg.ewbi.nby.one" "applicationinstances.opg.ewbi.nby.one"
check "Artefact CRD" "kubectl get crd artefacts.opg.ewbi.nby.one" "artefacts.opg.ewbi.nby.one"
check "File CRD" "kubectl get crd files.opg.ewbi.nby.one" "files.opg.ewbi.nby.one"
check "AvailabilityZone CRD" "kubectl get crd availabilityzones.opg.ewbi.nby.one" "availabilityzones.opg.ewbi.nby.one"

# Check federation-host pods
echo ""
echo -e "${YELLOW}[4/8] Checking federation-host pods...${NC}"
check "Controller manager pod (host)" "kubectl get pods -n federation-host -l control-plane=controller-manager -o jsonpath='{.items[0].status.phase}'" "Running"
check "Federation API pod (host)" "kubectl get pods -n federation-host -l control-plane=federation-api -o jsonpath='{.items[0].status.phase}'" "Running"

# Check federation-guest pods
echo ""
echo -e "${YELLOW}[5/8] Checking federation-guest pods...${NC}"
check "Controller manager pod (guest)" "kubectl get pods -n federation-guest -l control-plane=controller-manager -o jsonpath='{.items[0].status.phase}'" "Running"
check "Federation API pod (guest)" "kubectl get pods -n federation-guest -l control-plane=federation-api -o jsonpath='{.items[0].status.phase}'" "Running"

# Check HOST federation exists
echo ""
echo -e "${YELLOW}[6/8] Checking HOST federation...${NC}"
check "HOST federation exists" "kubectl get federation fed-e35f69d8-ae5a-456b-9f95-d950e4c03e8d -n federation-host -o jsonpath='{.metadata.name}'" "fed-e35f69d8-ae5a-456b-9f95-d950e4c03e8d"

# Check GUEST federation
echo ""
echo -e "${YELLOW}[7/8] Checking GUEST federation...${NC}"
check "GUEST federation exists" "kubectl get federation fed-2dae064c-28cc-456e-8b0a-dd67bab7d8f7 -n federation-guest -o jsonpath='{.metadata.name}'" "fed-2dae064c-28cc-456e-8b0a-dd67bab7d8f7"
check "GUEST federation state" "kubectl get federation fed-2dae064c-28cc-456e-8b0a-dd67bab7d8f7 -n federation-guest -o jsonpath='{.status.state}'" "Available"
check "GUEST has federationContextId" "kubectl get federation fed-2dae064c-28cc-456e-8b0a-dd67bab7d8f7 -n federation-guest -o jsonpath='{.status.federationContextId}'" "-"

# Check GUEST received offered zones from HOST
echo ""
echo -e "${YELLOW}[8/8] Checking federation data flow...${NC}"
check "GUEST received offeredAvailabilityZones" "kubectl get federation fed-2dae064c-28cc-456e-8b0a-dd67bab7d8f7 -n federation-guest -o jsonpath='{.status.offeredAvailabilityZones[0].zoneId}'" "2a8fffaf-50de-4f93-8c6f-05f1c84b5a5f"
check "GUEST accepted availability zone" "kubectl get federation fed-2dae064c-28cc-456e-8b0a-dd67bab7d8f7 -n federation-guest -o jsonpath='{.spec.acceptedAvailabilityZones[0]}'" "2a8fffaf-50de-4f93-8c6f-05f1c84b5a5f"

# Summary
echo ""
echo "=========================================="
echo -e "Validation Summary: ${GREEN}${PASSED} passed${NC}, ${RED}${FAILED} failed${NC}"
echo "=========================================="
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All validations passed! Federation is working correctly.${NC}"
    echo ""
    echo "Federation flow verified:"
    echo "  1. HOST federation created with offered availability zones"
    echo "  2. GUEST federation called HOST API"
    echo "  3. GUEST received federationContextId from HOST"
    echo "  4. GUEST received offeredAvailabilityZones from HOST"
    echo "  5. GUEST automatically accepted the availability zone"
    echo ""

    # Show federation details
    echo "GUEST Federation Status:"
    kubectl get federation fed-2dae064c-28cc-456e-8b0a-dd67bab7d8f7 -n federation-guest -o jsonpath='
  federationContextId: {.status.federationContextId}
  state: {.status.state}
  offeredZone: {.status.offeredAvailabilityZones[0].zoneId}
  acceptedZone: {.spec.acceptedAvailabilityZones[0]}
'
    echo ""
    exit 0
else
    echo -e "${RED}Some validations failed. Check the output above for details.${NC}"
    echo ""
    echo "Debugging commands:"
    echo "  kubectl get pods -A"
    echo "  kubectl logs -n federation-guest -l control-plane=controller-manager"
    echo "  kubectl logs -n federation-host -l control-plane=federation-api"
    exit 1
fi
