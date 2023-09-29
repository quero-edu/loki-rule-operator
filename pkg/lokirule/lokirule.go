package lokirule

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	"gopkg.in/yaml.v2"
)

type handleExpValidation func(expr string, lokiURL string) bool

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

func GenerateRuleConfigMapFile(rule *querocomv1alpha1.LokiRule, expValidation handleExpValidation) (map[string]string, error) {
	fileName := fmt.Sprintf("%s-%s.yaml", rule.Namespace, rule.Name)

	if expValidation == nil {
		expValidation = ExprValid
	}

	for _, group := range rule.Spec.Groups {
		for _, ruleGroup := range group.Rules {
			if !expValidation(ruleGroup.Expr, "https://fake.loki.url") {
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
