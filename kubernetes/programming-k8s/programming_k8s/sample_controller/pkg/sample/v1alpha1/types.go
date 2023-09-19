package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient

type NamespacedResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NamespacedResourceSpec   `json:"spec"`
	Status NamespacedResourceStatus `json:"status"`
}

type NamespacedResourceSpec struct {
	Name     string `json:"name"`
	Replicas int32  `json:"replicas"`
}

type NamespacedResourceStatus struct {
	AvailableReplicas int32 `json:"availableReplicas"`
}

type NamespacedResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NamespacedResource `json:"items"`
}

// +genclient:nonNamespaced

type ClusterResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterResourceSpec   `json:"spec"`
	Status ClusterResourceStatus `json:"status"`
}

type ClusterResourceSpec struct {
	Name     string `json:"name"`
	Replicas int32  `json:"replicas"`
}

type ClusterResourceStatus struct {
	AvailableReplicas int32 `json:"availableReplicas"`
}

type ClusterResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ClusterResource `json:"items"`
}
