{{/*
Expand the name of the chart.
*/}}
{{- define "inferencehub.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "inferencehub.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "inferencehub.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "inferencehub.labels" -}}
helm.sh/chart: {{ include "inferencehub.chart" . }}
{{ include "inferencehub.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: inferencehub
{{- end }}

{{/*
Selector labels
*/}}
{{- define "inferencehub.selectorLabels" -}}
app.kubernetes.io/name: {{ include "inferencehub.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
OpenWebUI labels
*/}}
{{- define "inferencehub.openwebui.labels" -}}
helm.sh/chart: {{ include "inferencehub.chart" . }}
{{ include "inferencehub.openwebui.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: inferencehub
app.kubernetes.io/component: ui
{{- end }}

{{/*
OpenWebUI selector labels
*/}}
{{- define "inferencehub.openwebui.selectorLabels" -}}
app.kubernetes.io/name: {{ include "inferencehub.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: ui
{{- end }}

{{/*
LiteLLM labels
*/}}
{{- define "inferencehub.litellm.labels" -}}
helm.sh/chart: {{ include "inferencehub.chart" . }}
{{ include "inferencehub.litellm.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: inferencehub
app.kubernetes.io/component: gateway
{{- end }}

{{/*
LiteLLM selector labels
*/}}
{{- define "inferencehub.litellm.selectorLabels" -}}
app.kubernetes.io/name: {{ include "inferencehub.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: gateway
{{- end }}

{{/*
PostgreSQL labels
*/}}
{{- define "inferencehub.postgresql.labels" -}}
helm.sh/chart: {{ include "inferencehub.chart" . }}
{{ include "inferencehub.postgresql.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: inferencehub
app.kubernetes.io/component: database
{{- end }}

{{/*
PostgreSQL selector labels
*/}}
{{- define "inferencehub.postgresql.selectorLabels" -}}
app.kubernetes.io/name: {{ include "inferencehub.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: database
{{- end }}

{{/*
Redis labels — accepts dict with "app" (openwebui|litellm) and "context" (root .)
*/}}
{{- define "inferencehub.redis.labels" -}}
helm.sh/chart: {{ include "inferencehub.chart" .context }}
{{ include "inferencehub.redis.selectorLabels" . }}
{{- if .context.Chart.AppVersion }}
app.kubernetes.io/version: {{ .context.Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .context.Release.Service }}
app.kubernetes.io/part-of: inferencehub
app.kubernetes.io/component: cache-{{ .app }}
{{- end }}

{{/*
Redis selector labels — accepts dict with "app" (openwebui|litellm) and "context" (root .)
*/}}
{{- define "inferencehub.redis.selectorLabels" -}}
app.kubernetes.io/name: {{ include "inferencehub.name" .context }}
app.kubernetes.io/instance: {{ .context.Release.Name }}
app.kubernetes.io/component: cache-{{ .app }}
{{- end }}

{{/*
OpenWebUI service name — derived from the subchart alias "openwebui".
Helm names subchart resources as "{release-name}-{alias}".
*/}}
{{- define "inferencehub.openwebui.serviceName" -}}
{{- printf "%s-openwebui" .Release.Name }}
{{- end }}

{{/*
LiteLLM service name — derived from the subchart alias "litellm".
Helm names subchart resources as "{release-name}-{alias}".
*/}}
{{- define "inferencehub.litellm.serviceName" -}}
{{- printf "%s-litellm" .Release.Name }}
{{- end }}

{{/*
PostgreSQL connection string for OpenWebUI database.
*/}}
{{- define "inferencehub.postgresql.openwebuiConnectionString" -}}
{{- if .Values.postgresql.external.enabled }}
{{- .Values.postgresql.external.openwebuiConnectionString }}
{{- else }}
{{- printf "postgresql://%s:%s@%s-postgresql:5432/openwebui"
    .Values.postgresql.auth.username
    .Values.postgresql.auth.password
    (include "inferencehub.fullname" .) }}
{{- end }}
{{- end }}

{{/*
PostgreSQL connection string for LiteLLM database.
*/}}
{{- define "inferencehub.postgresql.litellmConnectionString" -}}
{{- if .Values.postgresql.external.enabled }}
{{- .Values.postgresql.external.litellmConnectionString }}
{{- else }}
{{- printf "postgresql://%s:%s@%s-postgresql:5432/litellm"
    .Values.postgresql.auth.username
    .Values.postgresql.auth.password
    (include "inferencehub.fullname" .) }}
{{- end }}
{{- end }}

{{/*
OpenWebUI Redis host
*/}}
{{- define "inferencehub.redis.openwebui.host" -}}
{{- if .Values.redis.openwebui.external.enabled }}
{{- .Values.redis.openwebui.external.host }}
{{- else }}
{{- printf "%s-redis-openwebui" (include "inferencehub.fullname" .) }}
{{- end }}
{{- end }}

{{/*
OpenWebUI Redis port
*/}}
{{- define "inferencehub.redis.openwebui.port" -}}
{{- if .Values.redis.openwebui.external.enabled }}
{{- .Values.redis.openwebui.external.port | default 6379 }}
{{- else }}
{{- 6379 }}
{{- end }}
{{- end }}

{{/*
LiteLLM Redis host
*/}}
{{- define "inferencehub.redis.litellm.host" -}}
{{- if .Values.redis.litellm.external.enabled }}
{{- .Values.redis.litellm.external.host }}
{{- else }}
{{- printf "%s-redis-litellm" (include "inferencehub.fullname" .) }}
{{- end }}
{{- end }}

{{/*
LiteLLM Redis port
*/}}
{{- define "inferencehub.redis.litellm.port" -}}
{{- if .Values.redis.litellm.external.enabled }}
{{- .Values.redis.litellm.external.port | default 6379 }}
{{- else }}
{{- 6379 }}
{{- end }}
{{- end }}

{{/*
SearXNG service name
*/}}
{{- define "inferencehub.searxng.serviceName" -}}
{{- printf "%s-searxng" (include "inferencehub.fullname" .) }}
{{- end }}

{{/*
SearXNG labels
*/}}
{{- define "inferencehub.searxng.labels" -}}
helm.sh/chart: {{ include "inferencehub.chart" . }}
{{ include "inferencehub.searxng.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: inferencehub
app.kubernetes.io/component: websearch
{{- end }}

{{/*
SearXNG selector labels
*/}}
{{- define "inferencehub.searxng.selectorLabels" -}}
app.kubernetes.io/name: {{ include "inferencehub.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: websearch
{{- end }}

{{/*
Namespace
Use .Release.Namespace directly so parent chart resources and subchart resources
(e.g. litellm-helm migrations Job) all land in the same namespace. The release
namespace is set by the CLI via helm.Client.Namespace (or -n flag for manual runs).
*/}}
{{- define "inferencehub.namespace" -}}
{{- .Release.Namespace }}
{{- end }}

{{/*
Redis Deployment — call with dict: app (openwebui|litellm), redis (sub-config), context (root .)
*/}}
{{- define "inferencehub.redis.deployment" -}}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "inferencehub.fullname" .context }}-redis-{{ .app }}
  namespace: {{ include "inferencehub.namespace" .context }}
  labels:
    {{- include "inferencehub.redis.labels" . | nindent 4 }}
spec:
  replicas: 1
  selector:
    matchLabels:
      {{- include "inferencehub.redis.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "inferencehub.redis.selectorLabels" . | nindent 8 }}
    spec:
      containers:
        - name: redis
          image: "{{ .redis.image.repository }}:{{ .redis.image.tag }}"
          imagePullPolicy: {{ .redis.image.pullPolicy }}
          ports:
            - name: redis
              containerPort: 6379
              protocol: TCP
          {{- if .redis.auth.password }}
          command:
            - redis-server
            - --requirepass
            - $(REDIS_PASSWORD)
          env:
            - name: REDIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{ include "inferencehub.fullname" .context }}-redis-{{ .app }}-secret
                  key: redis-password
          {{- end }}
          livenessProbe:
            tcpSocket:
              port: 6379
            initialDelaySeconds: 30
            periodSeconds: 10
            timeoutSeconds: 5
            successThreshold: 1
            failureThreshold: 5
          readinessProbe:
            exec:
              command:
                - redis-cli
                - ping
            initialDelaySeconds: 5
            periodSeconds: 10
            timeoutSeconds: 5
            successThreshold: 1
            failureThreshold: 5
          {{- if .redis.persistence.enabled }}
          volumeMounts:
            - name: data
              mountPath: /data
          {{- end }}
          {{- if .redis.resources }}
          resources:
            {{- toYaml .redis.resources | nindent 12 }}
          {{- end }}
      {{- if .redis.persistence.enabled }}
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: {{ include "inferencehub.fullname" .context }}-redis-{{ .app }}-data
      {{- end }}
{{- end }}

{{/*
Redis Service — call with dict: app (openwebui|litellm), redis (sub-config), context (root .)
*/}}
{{- define "inferencehub.redis.service" -}}
apiVersion: v1
kind: Service
metadata:
  name: {{ include "inferencehub.fullname" .context }}-redis-{{ .app }}
  namespace: {{ include "inferencehub.namespace" .context }}
  labels:
    {{- include "inferencehub.redis.labels" . | nindent 4 }}
spec:
  type: {{ .redis.service.type }}
  selector:
    {{- include "inferencehub.redis.selectorLabels" . | nindent 4 }}
  ports:
    - port: {{ .redis.service.port }}
      targetPort: 6379
      protocol: TCP
      name: redis
{{- end }}

{{/*
Redis Secret — call with dict: app (openwebui|litellm), redis (sub-config), context (root .)
Rendered only when auth.password is set.
*/}}
{{- define "inferencehub.redis.secret" -}}
apiVersion: v1
kind: Secret
metadata:
  name: {{ include "inferencehub.fullname" .context }}-redis-{{ .app }}-secret
  namespace: {{ include "inferencehub.namespace" .context }}
  labels:
    {{- include "inferencehub.redis.labels" . | nindent 4 }}
type: Opaque
stringData:
  redis-password: {{ .redis.auth.password | quote }}
{{- end }}

{{/*
Redis PVC — call with dict: app (openwebui|litellm), redis (sub-config), context (root .)
*/}}
{{- define "inferencehub.redis.pvc" -}}
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: {{ include "inferencehub.fullname" .context }}-redis-{{ .app }}-data
  namespace: {{ include "inferencehub.namespace" .context }}
  labels:
    {{- include "inferencehub.redis.labels" . | nindent 4 }}
spec:
  accessModes:
    - {{ .redis.persistence.accessMode | quote }}
  {{- if .redis.persistence.storageClass }}
  {{- if (eq "-" .redis.persistence.storageClass) }}
  storageClassName: ""
  {{- else }}
  storageClassName: {{ .redis.persistence.storageClass | quote }}
  {{- end }}
  {{- end }}
  resources:
    requests:
      storage: {{ .redis.persistence.size | quote }}
{{- end }}
