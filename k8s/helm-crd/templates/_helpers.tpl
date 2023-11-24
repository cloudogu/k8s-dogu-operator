{{/* Chart basics
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec) starting from
Kubernetes 1.4+.
*/}}
{{- define "k8s-dogu-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}


{{/* All-in-one labels */}}
{{- define "k8s-dogu-operator.labels" -}}
app: ces
helm.sh/chart:  {{- printf " %s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{ include "k8s-dogu-operator.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/* Selector labels */}}
{{- define "k8s-dogu-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "k8s-dogu-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
