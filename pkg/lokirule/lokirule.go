package lokirule

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	"gopkg.in/yaml.v2"
)

func GenerateRuleConfigMapFile(rule *querocomv1alpha1.LokiRule, lokiURL string) (map[string]string, error) {
	fileName := fmt.Sprintf("%s-%s.yaml", rule.Namespace, rule.Name)

	for _, group := range rule.Spec.Groups {
		for _, ruleGroup := range group.Rules {
			if !ExprValid(ruleGroup.Expr, lokiURL) {
				return nil, fmt.Errorf("have an error on your LogQL %s", ruleGroup.Expr)
			}
		}
	}

	marshaledGroupData, err := yaml.Marshal(rule.Spec)
	if err != nil {
		return nil, err
	}

	stringifiedGroupData := string(marshaledGroupData)

	ruleFile := map[string]string{
		fileName: stringifiedGroupData,
	}

	return ruleFile, nil
}

func ExprValid(expr string, lokiURL string) bool {

	query := url.QueryEscape(expr)
	url := lokiURL + "/loki/api/v1/query?query=" + query
	resp, err := http.Get(url)

	if err != nil {
		log.Fatal(err)
		return false
	}

	if resp.StatusCode != 200 {
		return false
	}

	return true
}
