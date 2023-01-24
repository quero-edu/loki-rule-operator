package lokirule

import (
	"fmt"

	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
)

func GenerateLokiRuleLabels(lokiRule *querocomv1alpha1.LokiRule) map[string]string {
	labels := lokiRule.Labels

	if labels == nil {
		labels = make(map[string]string)
	}

	labels["app.kubernetes.io/component"] = "loki-rule-cfg"
	labels["app.kubernetes.io/managed-by"] = "loki-rule-operator"

	return labels
}

func GenerateConfigMapName(lokiRule *querocomv1alpha1.LokiRule) string {
	return fmt.Sprintf("%s-%s", lokiRule.Spec.Name, lokiRule.Namespace)
}
