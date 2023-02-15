package lokirule

import (
	"fmt"

	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	"gopkg.in/yaml.v2"
)

func generateRuleGroups(rule *querocomv1alpha1.LokiRule, ruleGroupName string) (string, error) {
	ruleFileMap := map[string][]map[string]interface{}{
		"groups": {
			{
				"name":  ruleGroupName,
				"rules": rule.Spec.Rules,
			},
		},
	}

	yamlfiedGroups, err := yaml.Marshal(ruleFileMap)
	if err != nil {
		return "", err
	}

	return string(yamlfiedGroups), err
}

func GenerateRuleConfigMapFile(rule *querocomv1alpha1.LokiRule) (map[string]string, error) {
	ruleGroupName := fmt.Sprintf("%s-%s", rule.Namespace, rule.Name)

	groups, err := generateRuleGroups(rule, ruleGroupName)
	if err != nil {
		return nil, err
	}

	fileName := fmt.Sprintf("%s.yaml", ruleGroupName)

	ruleFile := map[string]string{
		fileName: groups,
	}

	return ruleFile, nil
}
