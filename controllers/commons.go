package controllers

import (
	mydomainv1alpha1 "github.com/quero-edu/loki-rule-operator/api/v1alpha1"
)

func labels(v *mydomainv1alpha1.Traveller) map[string]string {
	// Fetches and sets labels

	return map[string]string{
		"app":             "visitors",
		"visitorssite_cr": v.Name,
	}
}
