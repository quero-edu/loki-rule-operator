package lokirule

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

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

func TestExprValid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	}))

	defer server.Close()

	isValid := ExprValid("test", "server.URL")

	if isValid == true {
		t.Errorf("Expected false, got %t", isValid)
	}

}

func ExampleResponseRecorder() {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html><body>Hello World!</body></html>")
	}

	req := httptest.NewRequest("GET", "https://loki-front.mock.test", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	fmt.Println(resp.StatusCode)
	fmt.Println(resp.Header.Get("Content-Type"))
	fmt.Println(string(body))

	// Output:
	// 200
	// text/html; charset=utf-8
	// <html><body>Hello World!</body></html>
}

var _ = Describe("TestGenerateRuleConfigMapFile", func() {
	It("should generate a valid rule file", func() {
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
								Expr:   "test_expr",
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
							"expr":   "test_expr",
						},
					},
				},
			},
		}

		handleExpValidationMock := func(expr string, lokiURL string) bool {
			return true
		}

		ruleFile, err := GenerateRuleConfigMapFile(rule, handleExpValidationMock)
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

		handleExpValidationMock := func(expr string, lokiURL string) bool {
			return false
		}
		_, err := GenerateRuleConfigMapFile(rule, handleExpValidationMock)
		Expect(err).To(Equal(fmt.Errorf("have an error on your LogQL %s", rule.Spec.Groups[0].Rules[0].Expr)))
	})
})
