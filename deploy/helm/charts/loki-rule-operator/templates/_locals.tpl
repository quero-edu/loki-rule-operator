{{- define "loki-rule-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "loki-rule-operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}


{{/*
Defines resource names used in multiple places
*/}}
{{- define "loki-rule-operator.locals" -}}
commonResources:
  leaderElectionRole:
    name: {{ include "loki-rule-operator.fullname" . }}-leader-election-role
  managerClusterRole:
    name: {{ include "loki-rule-operator.fullname" . }}-manager-role
  serviceAccount:
    name: {{ include "loki-rule-operator.serviceAccountName" . }}
{{- end }}