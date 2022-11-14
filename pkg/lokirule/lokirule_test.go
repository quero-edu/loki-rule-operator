package lokirule

import (
	"testing"

	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateConfigMap(t *testing.T) {
	lokiRuleInstance := &querocomv1alpha1.LokiRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-lokirule",
			Namespace: "test-namespace",
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

	if configMap.Labels["visitorssite_cr"] != lokiRuleInstance.Name {
		t.Errorf(
			"Expected configMap label visitorssite_cr to be %s, got %s",
			lokiRuleInstance.Name,
			configMap.Labels["visitorssite_cr"],
		)
	}
}
