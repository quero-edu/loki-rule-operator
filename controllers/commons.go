package controllers

import (
	"fmt"

	mydomainv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
)

func labels(v *mydomainv1alpha1.Traveller) map[string]string {
	// Fetches and sets labels

	return map[string]string{
		"app":             "visitors",
		"visitorssite_cr": v.Name,
	}
}

func generateVolumeName(v *mydomainv1alpha1.Traveller) string {
	return fmt.Sprintf("%s-volume", v.Spec.Name)
}
