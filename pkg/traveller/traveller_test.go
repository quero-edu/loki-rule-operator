package traveller

import (
	"testing"

	mydomainv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateConfigMap(t *testing.T) {
	travellerInstance := &mydomainv1alpha1.Traveller{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-traveller",
			Namespace: "test-namespace",
		},
		Spec: mydomainv1alpha1.TravellerSpec{
			Name: "test",
			Data: map[string]string{
				"foo": "bar",
			},
		},
	}

	configMap := GenerateConfigMap(travellerInstance)

	if configMap.Name != "test" {
		t.Errorf("Expected configmap name to be 'test', got %s", configMap.Name)
	}

	if configMap.Namespace != "test-namespace" {
		t.Errorf("Expected configmap namespace to be 'test-namespace', got %s", configMap.Namespace)
	}

	if configMap.Data["foo"] != "bar" {
		t.Errorf("Expected configmap data to be 'bar', got %s", configMap.Data["foo"])
	}

	if configMap.Labels["visitorssite_cr"] != travellerInstance.Name {
		t.Errorf(
			"Expected configmap label visitorssite_cr to be %s, got %s",
			travellerInstance.Name,
			configMap.Labels["visitorssite_cr"],
		)
	}
}
