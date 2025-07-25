/*
Copyright 2025.

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

// MatchSpec defines the match criteria for a policy profile.
type MatchSpec struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
}

// PolicyProfileSpec defines the desired state of PolicyProfile.
type PolicyProfileSpec struct {
	Match  MatchSpec         `json:"match"`
	Policy map[string]string `json:"policy,omitempty"`
}

// PolicyProfileStatus defines the observed state of PolicyProfile.
type PolicyProfileStatus struct {
	LastChecked metav1.Time `json:"lastChecked,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PolicyProfile is the Schema for the policyprofiles API.
type PolicyProfile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PolicyProfileSpec   `json:"spec,omitempty"`
	Status PolicyProfileStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PolicyProfileList contains a list of PolicyProfile.
type PolicyProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PolicyProfile `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PolicyProfile{}, &PolicyProfileList{})
}
