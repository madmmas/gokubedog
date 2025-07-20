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

type ViolatedResourceSpec struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// PolicyViolationReportSpec defines the desired state of PolicyViolationReport.
type PolicyViolationReportSpec struct {
	ViolatedResource ViolatedResourceSpec `json:"violatedResource"`
	ProfileName      string               `json:"profileName"`
	Drift            map[string]string    `json:"drift"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PolicyViolationReport is the Schema for the policyviolationreports API.
type PolicyViolationReport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PolicyViolationReportSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// PolicyViolationReportList contains a list of PolicyViolationReport.
type PolicyViolationReportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PolicyViolationReport `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PolicyViolationReport{}, &PolicyViolationReportList{})
}
