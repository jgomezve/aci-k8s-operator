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

// SegmentationPolicySpec defines the desired state of SegmentationPolicy
type SegmentationPolicySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Namespaces []string   `json:"namespaces"`
	Rules      []RuleSpec `json:"rules"`
}

type RuleSpec struct {
	Eth  string `json:"eth,omitempty"`
	IP   string `json:"ip,omitempty"`
	Port int    `json:"port,omitempty"`
}

// SegmentationPolicyStatus defines the observed state of SegmentationPolicy
type SegmentationPolicyStatus struct {
	Namespaces string `json:"namespaces"`
	Rules      string `json:"rules"`
}

//+kubebuilder:object:root=true
//+kubebuilder:printcolumn:name="Namespaces",type="string",JSONPath=".status.namespaces",description="Namespaces"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:resource:shortName=segpol
//+kubebuilder:subresource:status

// Group is the Schema for the groups API
// SegmentationPolicy is the Schema for the segmentationpolicies API
type SegmentationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SegmentationPolicySpec   `json:"spec,omitempty"`
	Status SegmentationPolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SegmentationPolicyList contains a list of SegmentationPolicy
type SegmentationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SegmentationPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SegmentationPolicy{}, &SegmentationPolicyList{})
}
