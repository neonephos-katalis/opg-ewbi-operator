#!/usr/bin/env bash
# Checks shared by the e2e workflows (.github/workflows/e2e-federation-*.yml) to verify that the chart
# runs in the expected install mode. CLUSTER_WIDE (true|false) is the expected mode.
#
#   check-install-mode.sh install   <cluster> <release-ns>
#       CONTROLLER_CLUSTER_WIDE / API_CLUSTER_WIDE match CLUSTER_WIDE, and the controller-manager
#       service account can list Federations in all namespaces only in cluster-wide mode.
#
#   check-install-mode.sh placement <cluster> <namespace-regex>
#       every opg.ewbi.katalis.com resource on the cluster is in a namespace matching the regex
#       (and there is at least one Federation).
set -euo pipefail

CRS=federations,availabilityzones,images,artefacts,applicationonboardings,applicationdeployments,consumptionmonitorings

: "${CLUSTER_WIDE:?CLUSTER_WIDE must be set to true or false}"

err() { echo "::error title=Install mode::$*" >&2; exit 1; }

env_of() {  # env_of <cluster> <ns> <control-plane label> <env name>
  kubectl --context "kind-$1" -n "$2" get deploy -l control-plane="$3" \
    -o jsonpath="{..env[?(@.name==\"$4\")].value}"
}

install() {
  local cluster=$1 ns=$2 v sa can expected
  v=$(env_of "$cluster" "$ns" controller-manager CONTROLLER_CLUSTER_WIDE)
  echo "[$cluster] CONTROLLER_CLUSTER_WIDE=$v"
  [ "$v" = "$CLUSTER_WIDE" ] || err "[$cluster] CONTROLLER_CLUSTER_WIDE is '$v', expected '$CLUSTER_WIDE'"

  if [ -n "$(kubectl --context "kind-$cluster" -n "$ns" get deploy -l control-plane=katalis-fm-api -o name)" ]; then
    v=$(env_of "$cluster" "$ns" katalis-fm-api API_CLUSTER_WIDE)
    echo "[$cluster] API_CLUSTER_WIDE=$v"
    [ "$v" = "$CLUSTER_WIDE" ] || err "[$cluster] API_CLUSTER_WIDE is '$v', expected '$CLUSTER_WIDE'"
  fi

  sa=$(kubectl --context "kind-$cluster" -n "$ns" get deploy -l control-plane=controller-manager \
    -o jsonpath='{.items[0].spec.template.spec.serviceAccountName}')
  can=$(kubectl --context "kind-$cluster" auth can-i list federations.opg.ewbi.katalis.com -A \
    --as="system:serviceaccount:$ns:$sa" 2>/dev/null || true)
  expected=no; [ "$CLUSTER_WIDE" = true ] && expected=yes
  echo "[$cluster] $ns:$sa can list federations in all namespaces: $can"
  [ "$can" = "$expected" ] || err "[$cluster] $ns:$sa 'can-i list federations -A' is '$can', expected '$expected'"

  echo "[$cluster] manager RBAC:"
  kubectl --context "kind-$cluster" get clusterrolebindings,rolebindings -A -o name | { grep manager-rolebinding || true; } | sed 's/^/  /'
}

placement() {
  local cluster=$1 regex=$2 rows bad
  rows=$(kubectl --context "kind-$cluster" get "$CRS" -A --no-headers \
    -o custom-columns='NAMESPACE:.metadata.namespace,KIND:.kind,NAME:.metadata.name')
  echo "[$cluster] federation resources (expected namespace: /$regex/):"
  echo "$rows" | sed 's/^/  /'
  [ -n "$(echo "$rows" | awk '$2 == "Federation"')" ] || err "[$cluster] no Federation found"
  bad=$(echo "$rows" | awk -v re="$regex" 'NF && $1 !~ re')
  [ -z "$bad" ] || err "[$cluster] resources outside /$regex/: $(echo "$bad" | awk '{print $1"/"$2"/"$3}' | paste -sd' ')"
}

case "${1:-}" in
  install)   shift; install "$@" ;;
  placement) shift; placement "$@" ;;
  *) echo "usage: $0 install <cluster> <release-ns> | placement <cluster> <namespace-regex>" >&2; exit 2 ;;
esac
