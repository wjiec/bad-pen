package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Greeter struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Desired state of the Greeter resource.
	Spec GreeterSpec `json:"spec"`

	// Status of the Greeter. This is set and managed automatically.
	// +optional
	Status GreeterStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type GreeterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Greeter `json:"items"`
}

type GreeterSpec struct {
	Schedule string `json:"schedule"`
	Message  string `json:"message"`
}

type GreeterStatus struct {
	Phase string `json:"phase"`
}
