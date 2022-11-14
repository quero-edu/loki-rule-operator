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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LokiRuleSpec defines the desired state of LokiRule
type LokiRuleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Name      string               `json:"name,omitempty" yaml:"name"`
	Selector  metav1.LabelSelector `json:"selector,omitempty" yaml:"selector"`
	MountPath string               `json:"mountPath,omitempty" yaml:"mountPath"`
	Data      map[string]string    `json:"data,omitempty" yaml:"data"`
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
