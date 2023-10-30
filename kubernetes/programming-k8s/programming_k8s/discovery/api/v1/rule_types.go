/*
Copyright 2023 Jayson Wang.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RuleSpec defines the desired state of Rule
type RuleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The name of the referenced Ingress object
	// +kubebuilder:validation:Required
	IngressRef string `json:"ingressRef,omitempty"`

	// The name of the backend server
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=^\w+(-[\w]+)*$
	ServerName string `json:"serverName,omitempty"`
}

// RuleStatus defines the observed state of Rule
type RuleStatus struct {
	// Whether the current rule is accepted
	Accepted bool `json:"assigned,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Rule is the Schema for the rules API
type Rule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RuleSpec   `json:"spec,omitempty"`
	Status RuleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RuleList contains a list of Rule
type RuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Rule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Rule{}, &RuleList{})
}
