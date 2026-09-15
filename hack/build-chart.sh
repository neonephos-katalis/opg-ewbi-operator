#!/bin/bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

make generate
make manifests
make base-chart
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
      {{- if .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml .Values.imagePullSecrets | nindent 8 }}
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
      serviceAccountName: {{ .Values.federation.serviceAccountName }}
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
{{- if .Values.networkPolicy.enable }}
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

cat > dist/chart/templates/rbac/host_delete_protection_policy.yaml <<'EOF'
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: opg-ewbi-host-delete-protection-policy-{{ .Release.Namespace }}
spec:
  failurePolicy: Fail
  matchConstraints:
    resourceRules:
    - apiGroups: ["opg.ewbi.katalis.com"]
      apiVersions: ["v1beta1"]
      operations: ["DELETE"]
      resources:
      - federations
      - images
      - artefacts
      - availabilityzones
      - applicationonboardings
      - applicationdeployments
  matchConditions:
  - name: exclude-controller-manager
    # WARNING update the following line with the correct name for guest's service account
    expression: >-
      request.userInfo.username != "system:serviceaccount:host:guest-sa"
  validations:
  - expression: >-
      !(
        request.resource.resource == "federations"
          ? (has(oldObject.spec.federationData) && has(oldObject.spec.federationData.relationType) && oldObject.spec.federationData.relationType == "HOST")
          : (has(oldObject.spec) && has(oldObject.spec.relationType) && oldObject.spec.relationType == "HOST")
      )
    messageExpression: >-
      "deletion of HOST-relation resources is prohibited except by the controller-manager itself (received identity: '" + request.userInfo.username + "')"
---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: opg-ewbi-host-delete-protection-policy-{{ .Release.Namespace }}-binding
spec:
  policyName: opg-ewbi-host-delete-protection-policy-{{ .Release.Namespace }}
  validationActions: ["Deny"]
  matchResources:
    namespaceSelector:
      matchLabels:
        kubernetes.io/metadata.name: {{ .Release.Namespace }}
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
controllerManager:
  replicas: 1
  container:
    enable: true
    image:
      repository: ghcr.io/ipcei-tim-t2/operator/katalis-operator
      tag: dev-0.0.312
      pullPolicy: Always
    args:
      - "--leader-elect"
      - "--metrics-bind-address=:8443"
      - "--health-probe-bind-address=:8081"
    opgInsecureSkipVerify: false
    resources:
      limits:
        cpu: 500m
        memory: 128Mi
      requests:
        cpu: 10m
        memory: 64Mi
    livenessProbe:
      initialDelaySeconds: 15
      periodSeconds: 20
      httpGet:
        path: /healthz
        port: 8081
    readinessProbe:
      initialDelaySeconds: 5
      periodSeconds: 10
      httpGet:
        path: /readyz
        port: 8081
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
          - "ALL"
  securityContext:
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
  terminationGracePeriodSeconds: 10
  serviceAccountName: opg-ewbi-operator-controller-manager

federation:
  enable: true
  replicas: 1
  image:
    repository: ghcr.io/ipcei-tim-t2/operator/katalis-api
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
  serviceAccountName: opg-ewbi-operator-controller-manager
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

imagePullSecrets:
  - name: opg-registry-secret

# [RBAC]: To enable RBAC (Permissions) configurations
rbac:
  enable: true

# [CRDs]: To enable the CRDs
crd:
  # This option determines whether the CRDs are included
  # in the installation process.
  enable: false

  # Enabling this option adds the "helm.sh/resource-policy": keep
  # annotation to the CRD, ensuring it remains installed even when
  # the Helm release is uninstalled.
  # NOTE: Removing the CRDs will also remove all cert-manager CR(s)
  # (Certificates, Issuers, ...) due to garbage collection.
  keep: true

# [METRICS]: Set to true to generate manifests for exporting metrics.
# To disable metrics export set false, and ensure that the
# ControllerManager argument "--metrics-bind-address=:8443" is removed.
metrics:
  enable: false

# [PROMETHEUS]: To enable a ServiceMonitor to export metrics to Prometheus set true
prometheus:
  enable: false

# [CERT-MANAGER]: To enable cert-manager injection to webhooks set true
certmanager:
  enable: false

# [NETWORK POLICIES]: To enable NetworkPolicies set true
networkPolicy:
  enable: false
EOF

cat > dist/chart/templates/_helpers.tpl <<'EOF'
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

cat > dist/chart/templates/manager/manager.yaml <<'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "katalis.fullname" (list . "ewbi-operator-manager") }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "chart.labels" . | nindent 4 }}
    control-plane: controller-manager
spec:
  replicas:  {{ .Values.controllerManager.replicas }}
  selector:
    matchLabels:
      {{- include "chart.selectorLabels" . | nindent 6 }}
      control-plane: controller-manager
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
      labels:
        {{- include "chart.labels" . | nindent 8 }}
        control-plane: controller-manager
        {{- if and .Values.controllerManager.pod .Values.controllerManager.pod.labels }}
        {{- range $key, $value := .Values.controllerManager.pod.labels }}
        {{ $key }}: {{ $value }}
        {{- end }}
        {{- end }}
    spec:
      imagePullSecrets:
        {{- if .Values.imagePullSecrets }}
        {{- toYaml .Values.imagePullSecrets | nindent 8 }}
        {{- end }}
      containers:
        {{- if .Values.controllerManager.container.enable }}
        - name: manager
          args:
            {{- range .Values.controllerManager.container.args }}
            - {{ . }}
            {{- end }}
            {{- if .Values.controllerManager.container.opgInsecureSkipVerify }}
            - --opg-insecure-skip-verify
             {{- end }}
          command:
            - /manager
          image: {{ .Values.controllerManager.container.image.repository }}:{{ .Values.controllerManager.container.image.tag }}
          imagePullPolicy: {{ .Values.controllerManager.container.image.pullPolicy | default "Always" }}
          env:
            - name: NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
          {{- if .Values.controllerManager.container.env }}
            {{- range $key, $value := .Values.controllerManager.container.env }}
            - name: {{ $key }}
              value: {{ $value }}
            {{- end }}
          {{- end }}
          livenessProbe:
            {{- toYaml .Values.controllerManager.container.livenessProbe | nindent 12 }}
          readinessProbe:
            {{- toYaml .Values.controllerManager.container.readinessProbe | nindent 12 }}
          resources:
            {{- toYaml .Values.controllerManager.container.resources | nindent 12 }}
          securityContext:
            {{- toYaml .Values.controllerManager.container.securityContext | nindent 12 }}
          {{- if and .Values.certmanager.enable (or .Values.webhook.enable .Values.metrics.enable) }}
          volumeMounts:
            {{- if and .Values.metrics.enable .Values.certmanager.enable }}
            - name: metrics-certs
              mountPath: /tmp/k8s-metrics-server/metrics-certs
              readOnly: true
            {{- end }}
          {{- end }}
        {{- end }}
      securityContext:
        {{- toYaml .Values.controllerManager.securityContext | nindent 8 }}
      serviceAccountName: {{ .Values.controllerManager.serviceAccountName }}
      terminationGracePeriodSeconds: {{ .Values.controllerManager.terminationGracePeriodSeconds }}
      {{- if and .Values.certmanager.enable (or .Values.webhook.enable .Values.metrics.enable) }}
      volumes:
        {{- if and .Values.metrics.enable .Values.certmanager.enable }}
        - name: metrics-certs
          secret:
            secretName: metrics-server-cert
        {{- end }}
      {{- end }}
EOF

sed -i \
  -e 's/name: opg-ewbi-manager-role$/name: opg-ewbi-manager-role-{{ .Release.Namespace }}/' \
  dist/chart/templates/rbac/role.yaml
sed -i \
  's/name: opg-ewbi-manager-role$/name: opg-ewbi-manager-role-{{ .Release.Namespace }}/g' \
  dist/chart/templates/rbac/role_binding.yaml

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
