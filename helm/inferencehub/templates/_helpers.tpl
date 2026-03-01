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
Redis labels
*/}}
{{- define "inferencehub.redis.labels" -}}
helm.sh/chart: {{ include "inferencehub.chart" . }}
{{ include "inferencehub.redis.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: inferencehub
app.kubernetes.io/component: cache
{{- end }}

{{/*
Redis selector labels
*/}}
{{- define "inferencehub.redis.selectorLabels" -}}
app.kubernetes.io/name: {{ include "inferencehub.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: cache
{{- end }}

{{/*
Service account name for LiteLLM
*/}}
{{- define "inferencehub.litellm.serviceAccountName" -}}
{{- if .Values.litellm.serviceAccount.create }}
{{- default (printf "%s-litellm" (include "inferencehub.fullname" .)) .Values.litellm.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.litellm.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
PostgreSQL connection string
*/}}
{{- define "inferencehub.postgresql.connectionString" -}}
{{- if .Values.postgresql.external.enabled }}
{{- .Values.postgresql.external.connectionString | default (printf "postgresql://%s:%s@%s:%d/%s" .Values.postgresql.external.username .Values.postgresql.external.password .Values.postgresql.external.host (.Values.postgresql.external.port | int) .Values.postgresql.external.database) }}
{{- else }}
{{- printf "postgresql://%s:%s@%s-postgresql:%d/%s" .Values.postgresql.auth.username .Values.postgresql.auth.password (include "inferencehub.fullname" .) (5432 | int) .Values.postgresql.auth.database }}
{{- end }}
{{- end }}

{{/*
Redis host
*/}}
{{- define "inferencehub.redis.host" -}}
{{- if .Values.redis.external.enabled }}
{{- .Values.redis.external.host }}
{{- else }}
{{- printf "%s-redis" (include "inferencehub.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Redis port
*/}}
{{- define "inferencehub.redis.port" -}}
{{- if .Values.redis.external.enabled }}
{{- .Values.redis.external.port | default 6379 }}
{{- else }}
{{- 6379 }}
{{- end }}
{{- end }}

{{/*
Namespace
*/}}
{{- define "inferencehub.namespace" -}}
{{- .Values.global.namespace | default .Release.Namespace }}
{{- end }}
