#!/bin/bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

make generate
make crd-chart

mkdir -p dist/chart/templates/federation

cat > dist/chart/templates/federation/deployment.yaml <<'EOF'
{{- if .Values.federation.enable }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "katalis.fullname" (list . "fm-api") }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "chart.labels" . | nindent 4 }}
    control-plane: katalis-fm-api
spec:
  replicas:  {{ .Values.federation.replicas }}
  selector:
    matchLabels:
      {{- include "chart.selectorLabels" . | nindent 6 }}
      control-plane: katalis-fm-api
  template:
    metadata:
      labels:
        {{- include "chart.labels" . | nindent 8 }}
        control-plane: katalis-fm-api
    spec:
      {{- if .Values.manager.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml .Values.manager.imagePullSecrets | nindent 8 }}
      {{- end }}
      containers:
        - name: api
          image: {{ .Values.federation.image.repository }}:{{ .Values.federation.image.tag }}
          imagePullPolicy: {{ .Values.federation.image.pullPolicy | default "Always" }}
          env:
          - name: CONTROLLER_NAMESPACE
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
          - name: API_CLUSTER_WIDE
            value: {{ .Values.clusterWide | default false | quote }}
          - name: HYDRA_BASE_ADDR
            value: "http://{{ .Values.federation.externalServices.hydra.name }}:{{ .Values.federation.externalServices.hydra.port }}"
          - name: CAMARA_LOG_LEVEL
            value: "{{ .Values.federation.log.level }}"
          - name: CAMARA_API_ROOT
            value: "{{ .Values.federation.services.federation.name }}:{{ .Values.federation.services.federation.port }}"
          - name: CAMARA_HOST_AGENT_ADDR
            value: "0.0.0.0:8080"
          - name: USERBUSTER_HOST
            value: "{{ .Values.federation.externalServices.userBuster.name }}"
          - name: USERBUSTER_PORT
            value: "{{ .Values.federation.externalServices.userBuster.port }}"
          ports:
          - containerPort: 8080
            name: api
            protocol: TCP
          resources:
            {{- toYaml .Values.federation.resources | nindent 12 }}
          securityContext:
            {{- toYaml .Values.federation.securityContext | nindent 12 }}
      serviceAccountName: {{ include "opg-ewbi.serviceAccountName" . }}
      terminationGracePeriodSeconds: {{ .Values.federation.terminationGracePeriodSeconds }}
{{- end }}
EOF

cat > dist/chart/templates/federation/service.yaml <<'EOF'
{{- if .Values.federation.enable }}
apiVersion: v1
kind: Service
metadata:
  name: {{ .Values.federation.services.federation.name }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "chart.labels" . | nindent 4 }}
spec:
  type: NodePort
  ports:
    - port: {{ .Values.federation.services.federation.port }}
      targetPort: api
      protocol: TCP
      {{- if .Values.federation.services.federation.nodePort }}
      nodePort: {{ .Values.federation.services.federation.nodePort }}
      {{- end }}
  selector:
    control-plane: katalis-fm-api
{{- end }}
EOF

cat > dist/chart/templates/network-policy/allow-webhook-traffic.yaml <<'EOF'
{{- if .Values.networkPolicy.enabled }}
# This NetworkPolicy allows ingress traffic to your webhook server running
# as part of the controller-manager from specific namespaces and pods. CR(s) which uses webhooks
# will only work when applied in namespaces labeled with 'webhook: enabled'
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  labels:
    {{- include "chart.labels" . | nindent 4 }}
  name: allow-webhook-traffic
  namespace: {{ .Release.Namespace }}
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
      app.kubernetes.io/name: opg-ewbi
  policyTypes:
    - Ingress
  ingress:
    # This allows ingress traffic from any namespace with the label webhook: enabled
    - from:
      - namespaceSelector:
          matchLabels:
            webhook: enabled # Only from namespaces with this label
      ports:
        - port: 443
          protocol: TCP
{{- end -}}
EOF

cat > dist/chart/Chart.yaml <<'EOF'
apiVersion: v2
name: opg-ewbi-operator
description: opg-ewbi Operator helm chart
type: application
version: 0.0.6
appVersion: "0.0.5"
icon: https://blocklogos.s3.eu-west-1.amazonaws.com/nearbycomputing.png
EOF

cat > dist/chart/values.yaml <<'EOF'
# [MANAGER]: Manager Deployment Configurations
manager:
  enabled: true
  replicas: 1
  image:
    repository: ghcr.io/neonephos-katalis/opg-ewbi-operator
    tag: dev-0.0.312
    pullPolicy: Always
  args:
    - "--leader-elect"
  opgInsecureSkipVerify: false
  env:
    - name: NAMESPACE
      valueFrom:
        fieldRef:
          fieldPath: metadata.namespace
  imagePullSecrets:
    - name: opg-registry-secret
  resources:
    limits:
      cpu: 500m
      memory: 128Mi
    requests:
      cpu: 10m
      memory: 64Mi
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop:
        - "ALL"
  podSecurityContext:
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
  affinity: {}
  nodeSelector: {}
  tolerations: []
  terminationGracePeriodSeconds: 10

federation:
  enable: true
  replicas: 1
  image:
    repository: ghcr.io/neonephos-katalis/opg-ewbi-api
    tag: dev-0.0.312
    pullPolicy: Always
  resources:
    limits:
      cpu: 500m
      memory: 128Mi
    requests:
      cpu: 10m
      memory: 64Mi
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop:
        - "ALL"
  terminationGracePeriodSeconds: 10
  log:
    level: "info"
  externalServices:
    hydra:
      name: katalis-fm-hydra-admin
      port: 4445
    userBuster:
      name: katalis-fm-user-buster
      port: 80
  services:
    federation:
      name: katalis-fm-api
      port: 8080
      # nodePort: 30081  # Optional: specify a specific nodePort (30000-32767), or leave empty for auto-assignment

fullnameOverride: katalis-fm

# [RBAC]: To enable RBAC (Permissions) configurations
rbac:
  namespaced: false
  helpers:
    enabled: false

serviceAccount:
  enabled: true

# [CRDs]: To enable the CRDs
crd:
  # This option determines whether the CRDs are included
  # in the installation process.
  enabled: false

  # Enabling this option adds the "helm.sh/resource-policy": keep
  # annotation to the CRD, ensuring it remains installed even when
  # the Helm release is uninstalled.
  # NOTE: Removing the CRDs will also remove all cert-manager CR(s)
  # (Certificates, Issuers, ...) due to garbage collection.
  keep: true

# [METRICS]: Set to true to generate manifests for exporting metrics.
metrics:
  enabled: false
  port: 8443
  secure: true

# [PROMETHEUS]: To enable a ServiceMonitor to export metrics to Prometheus set true
prometheus:
  enabled: false

# [CERT-MANAGER]: To enable cert-manager injection to webhooks set true
certManager:
  enabled: false

# [NETWORK POLICIES]: To enable NetworkPolicies set true
networkPolicy:
  enabled: false

clusterWide: false
EOF

cat > dist/chart/templates/_katalis.tpl <<'EOF'
{{- define "chart.name" -}}
{{- if .Chart }}
  {{- if .Chart.Name }}
    {{- .Chart.Name | trunc 63 | trimSuffix "-" }}
  {{- else if .Values.nameOverride }}
    {{ .Values.nameOverride | trunc 63 | trimSuffix "-" }}
  {{- else }}
    opg-ewbi-operator
  {{- end }}
{{- else }}
  opg-ewbi-operator
{{- end }}
{{- end }}


{{- define "chart.labels" -}}
{{- if .Chart.AppVersion -}}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
{{- if .Chart.Version }}
helm.sh/chart: {{ .Chart.Version | quote }}
{{- end }}
app.kubernetes.io/name: {{ include "chart.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}


{{- define "chart.selectorLabels" -}}
app.kubernetes.io/name: {{ include "chart.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}


{{- define "chart.hasMutatingWebhooks" -}}
{{- $hasMutating := false }}
{{- range . }}
  {{- if eq .type "mutating" }}
    $hasMutating = true }}{{- end }}
{{- end }}
{{ $hasMutating }}}}{{- end }}


{{- define "chart.hasValidatingWebhooks" -}}
{{- $hasValidating := false }}
{{- range . }}
  {{- if eq .type "validating" }}
    $hasValidating = true }}{{- end }}
{{- end }}
{{ $hasValidating }}}}{{- end }}

{{/* Generate resource name */}}
{{- define "katalis.fullname" -}}
{{- $root := index . 0 -}}
{{- $objectName := index . 1 -}}
{{- if $root.Values.fullnameOverride }}
{{- $root.Values.fullnameOverride }}-{{ $objectName }}
{{- else }}
{{- $objectName }}
{{- end -}}
{{- end -}}
EOF

sed -i \
  -e 's/^\( *image: \)"\(.*\)"$/\1\2/' \
  -e 's/^\( *\){{- range .Values.manager.args }}$/\1{{- if .Values.manager.opgInsecureSkipVerify }}\n\1- --opg-insecure-skip-verify\n\1{{- end }}\n&/' \
  dist/chart/templates/manager/manager.yaml

cat > dist/chart-crd/Chart.yaml <<'EOF'
apiVersion: v2
name: opg-ewbi-crd
description: A Helm chart to distribute the opg-ewbi-crd
type: application
version: 0.0.3
appVersion: "0.0.5"
icon: https://blocklogos.s3.eu-west-1.amazonaws.com/nearbycomputing.png
EOF

helm lint dist/chart
helm lint dist/chart-crd

echo "dist/chart and dist/chart-crd regenerated and patched."
