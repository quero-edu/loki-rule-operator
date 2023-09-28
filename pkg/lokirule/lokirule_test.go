package lokirule

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLokiRule(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LokiRule Suite")
}

var _ = Describe("TestGenerateRuleConfigMapFile", func() {
	It("should generate a valid rule file", func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", `=~url-do-loki/.*`,
			httpmock.NewStringResponder(200, `{"status":"success"}`))

		rule := &querocomv1alpha1.LokiRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-rule",
				Namespace: "test-namespace",
			},
			Spec: querocomv1alpha1.LokiRuleSpec{
				Groups: []querocomv1alpha1.RuleGroup{
					{
						Name: "test-namespace-test-rule",
						Rules: []querocomv1alpha1.Rule{
							{
								Record: "test_record",
								Expr:   "{test=test}",
							},
						},
					},
				},
			},
		}

		expectedParsedFileName := "test-namespace-test-rule.yaml"
		expectedParsedYamlContent := map[string][]map[interface{}]interface{}{
			"groups": {
				{
					"name": "test-namespace-test-rule",
					"rules": []interface{}{
						map[interface{}]interface{}{
							"record": "test_record",
							"expr":   "{test=test}",
						},
					},
				},
			},
		}

		ruleFile, err := GenerateRuleConfigMapFile(rule, "url-do-loki")
		Expect(err).To(BeNil())

		parsedKeys := []string{}
		for key := range ruleFile {
			parsedKeys = append(parsedKeys, key)
		}

		Expect(len(parsedKeys)).To(Equal(1))

		parsedFileName := parsedKeys[0]
		Expect(parsedFileName).To(Equal(expectedParsedFileName))

		parsedRuleFileContent := map[string][]map[interface{}]interface{}{}
		err = yaml.Unmarshal([]byte(ruleFile[parsedFileName]), &parsedRuleFileContent)
		Expect(err).To(BeNil())

		GinkgoWriter.Printf(
			"Comparing\n---\nparsedRuleFileContent(type: %T): %v\nexpectedParsedYamlContent(type: %T): %v\n",
			parsedRuleFileContent,
			parsedRuleFileContent,
			expectedParsedYamlContent,
			expectedParsedYamlContent,
			ruleFile,
		)

		Expect(reflect.DeepEqual(parsedRuleFileContent, expectedParsedYamlContent)).To(BeTrue())
	})

	It("should generate an erro of Expr", func() {
		httpmock.Activate()
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder("GET", `=~url-do-loki/.*`,
			httpmock.NewStringResponder(500, `{"status":"fail"}`))

		rule := &querocomv1alpha1.LokiRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-rule",
				Namespace: "test-namespace",
			},
			Spec: querocomv1alpha1.LokiRuleSpec{
				Groups: []querocomv1alpha1.RuleGroup{
					{
						Name: "test-namespace-test-rule",
						Rules: []querocomv1alpha1.Rule{
							{
								Record: "test_record",
								Expr:   "INVALID_EXPR",
							},
						},
					},
				},
			},
		}

		_, err := GenerateRuleConfigMapFile(rule, "url-do-loki")
		Expect(err).To(Equal(fmt.Errorf("have an error on your LogQL %s", rule.Spec.Groups[0].Rules[0].Expr)))
	})
})
