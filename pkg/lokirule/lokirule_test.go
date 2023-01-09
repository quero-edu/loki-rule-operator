package lokirule

import (
	"reflect"
	"testing"

	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateLokiRuleLabels(t *testing.T) {
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

	generatedLabels := GenerateLokiRuleLabels(lokiRuleInstance)

	expectedLabels := map[string]string{
		"app.kubernetes.io/component":  "loki-rule-cfg",
		"app.kubernetes.io/managed-by": "loki-rule-operator",
		labelKey:                       "somevalue",
	}

	if !reflect.DeepEqual(generatedLabels, expectedLabels) {
		t.Errorf("Generated labels are not as expected. Expected: %v, Got: %v", expectedLabels, generatedLabels)
		return
	}
}
