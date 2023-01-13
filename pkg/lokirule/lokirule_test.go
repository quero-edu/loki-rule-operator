package lokirule

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLokiRule(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LokiRule Suite")
}

var _ = Describe("TestGenerateLokiRuleLabels", func() {
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

	It("should generate labels as expected", func() {
		generatedLabels := GenerateLokiRuleLabels(lokiRuleInstance)
		expectedLabels := map[string]string{
			"app.kubernetes.io/component":  "loki-rule-cfg",
			"app.kubernetes.io/managed-by": "loki-rule-operator",
			labelKey:                       "somevalue",
		}

		Expect(generatedLabels).To(Equal(expectedLabels))
	})
})
