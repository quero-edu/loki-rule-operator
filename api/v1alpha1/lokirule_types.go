/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RuleGroup struct {
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	Rules []Rule `json:"rules,omitempty" yaml:"rules,omitempty"`
}

// Rule defines a rule for a LokiRule
type Rule struct {
	Alert       string            `json:"alert,omitempty" yaml:"alert,omitempty"`
	Record      string            `json:"record,omitempty" yaml:"record,omitempty"`
	Expr        string            `json:"expr,omitempty" yaml:"expr"`
	For         string            `json:"for,omitempty" yaml:"for,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

// LokiRuleSpec defines the desired state of LokiRule
type LokiRuleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Groups []RuleGroup `json:"groups,omitempty" yaml:"groups"`
}

// LokiRuleStatus defines the observed state of LokiRule
type LokiRuleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LokiRule is the Schema for the lokiRules API
type LokiRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LokiRuleSpec   `json:"spec,omitempty"`
	Status LokiRuleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LokiRuleList contains a list of LokiRule
type LokiRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LokiRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LokiRule{}, &LokiRuleList{})
}
