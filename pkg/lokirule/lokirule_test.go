package lokirule

import (
	"testing"

	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateConfigMap(t *testing.T) {
	labelKey := "somekey"

	labels := map[string]string{
		labelKey: "somevalue",
	}

	lokiRuleInstance := &querocomv1alpha1.LokiRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lokirule",
			Namespace: "test-namespace",
			Labels:    labels,
		},
		Spec: querocomv1alpha1.LokiRuleSpec{
			Name: "test",
			Data: map[string]string{
				"foo": "bar",
			},
		},
	}

	configMap := GenerateConfigMap(lokiRuleInstance)

	if configMap.Name != "test" {
		t.Errorf("Expected configMap name to be 'test', got %s", configMap.Name)
	}

	if configMap.Namespace != "test-namespace" {
		t.Errorf("Expected configMap namespace to be 'test-namespace', got %s", configMap.Namespace)
	}

	if configMap.Data["foo"] != "bar" {
		t.Errorf("Expected configMap data to be 'bar', got %s", configMap.Data["foo"])
	}
	if configMap.Labels[labelKey] != lokiRuleInstance.Labels[labelKey] {
		t.Errorf(
			"Expected configMap label %s to match loki instance, wanted %s got %s",
			labelKey,
			lokiRuleInstance.Labels[labelKey],
			configMap.Labels[labelKey],
		)
	}

	if configMap.Labels["app.kubernetes.io/component"] != "loki-rule-cfg" {
		t.Errorf(
			"Expected configMap label app.kubernetes.io/component to be %s, got %s",
			"loki-rule-cfg",
			configMap.Labels["app.kubernetes.io/component"],
		)
	}

	if configMap.Labels["app.kubernetes.io/managed-by"] != "loki-rule-operator" {
		t.Errorf(
			"Expected configMap label app.kubernetes.io/managed-by to be %s, got %s",
			"loki-rule-operator",
			configMap.Labels["app.kubernetes.io/managed-by"],
		)
	}
}
