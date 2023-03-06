package lokirule

import (
	"fmt"

	querocomv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	"gopkg.in/yaml.v2"
)

func GenerateRuleConfigMapFile(rule *querocomv1alpha1.LokiRule) (map[string]string, error) {
	fileName := fmt.Sprintf("%s-%s.yaml", rule.Namespace, rule.Name)

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
