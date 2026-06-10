{{- define "manga-cdc.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "manga-cdc.labels" -}}
helm.sh/chart: {{ include "manga-cdc.name" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}
