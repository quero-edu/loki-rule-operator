package lokirule

import (
	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenerateConfigMap(lokiRule *querocomv1alpha1.LokiRule) *corev1.ConfigMap {
	labels := labels(lokiRule)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lokiRule.Spec.Name,
			Namespace: lokiRule.Namespace,
			Labels:    labels,
		},
		Data: lokiRule.Spec.Data,
	}

	return configMap
}

func labels(lokiRule *querocomv1alpha1.LokiRule) map[string]string {
	labels := lokiRule.Labels

	if labels == nil {
		labels = make(map[string]string)
	}

	labels["app.kubernetes.io/component"] = "loki-rule-cfg"
	labels["app.kubernetes.io/managed-by"] = "loki-rule-operator"

	return labels
}
