package lokirule

import (
	"fmt"
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


// MockHTTPClient is a mock HTTP client that implements the http.Client interface.
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

// Do is a method on MockHTTPClient that allows you to mock HTTP requests.
func (c *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.DoFunc(req)
}

func TestValidateLogQLExpression(t *testing.T) {
	// Create a mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a successful response
		if r.URL.Query().Get("query") == "valid_expr" {
			w.WriteHeader(http.StatusOK)
		} else {
			// Simulate a non-200 response
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))

	defer mockServer.Close()

	// Create a mock HTTP client
	mockHTTPClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Simulate a successful response
			if req.URL.String() == mockServer.URL+"/loki/api/v1/query?query=valid_expr" {
				return &http.Response{
					StatusCode: http.StatusOK,
				}, nil
			}
			// Simulate a non-200 response
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
			}, nil
		},
	}

	// Replace the default HTTP client with the mock HTTP client
	// Create a custom HTTP client using the mock HTTP client
	client := &http.Client{
		Transport: mockHTTPClient,
	}
	// Test case 1: Valid expression
	validExpression := "valid_expr"
	if !ValidateLogQLExpression(validExpression, mockServer.URL) {
		t.Errorf("Expected ValidateLogQLExpression to return true for a valid expression, but got false")
	}

	// Test case 2: Invalid expression
	invalidExpression := "invalid_expr"
	if ValidateLogQLExpression(invalidExpression, mockServer.URL) {
		t.Errorf("Expected ValidateLogQLExpression to return false for an invalid expression, but got true")
	}

	// Test case 3: Network error (unreachable server)
	if ValidateLogQLExpression("valid_expr", "https://nonexistent.url") {
		t.Errorf("Expected ValidateLogQLExpression to return false for an unreachable server, but got true")
	}
}