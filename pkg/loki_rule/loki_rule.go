package traveller

import (
	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenerateConfigMap(traveller *querocomv1alpha1.Traveller) *corev1.ConfigMap {
	labels := labels(traveller)

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      traveller.Spec.Name,
			Namespace: traveller.Namespace,
			Labels:    labels,
		},
		Data: traveller.Spec.Data,
	}

	return configMap
}

func labels(v *querocomv1alpha1.Traveller) map[string]string {
	return map[string]string{
		"app":             "visitors",
		"visitorssite_cr": v.Name,
	}
}
