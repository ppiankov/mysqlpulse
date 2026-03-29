{{- define "mysqlpulse.name" -}}
mysqlpulse
{{- end -}}

{{- define "mysqlpulse.fullname" -}}
{{- if contains (include "mysqlpulse.name" .) .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name (include "mysqlpulse.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "mysqlpulse.targetFullname" -}}
{{- printf "%s-%s" (include "mysqlpulse.fullname" .root) .target.name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "mysqlpulse.labels" -}}
app.kubernetes.io/name: {{ include "mysqlpulse.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- end -}}

{{- define "mysqlpulse.selectorLabels" -}}
app.kubernetes.io/name: {{ include "mysqlpulse.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
