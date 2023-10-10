/*
Copyright (c) 2023 Jayson Wang

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

// ClusterResourceSpecApplyConfiguration represents an declarative configuration of the ClusterResourceSpec type for use
// with apply.
type ClusterResourceSpecApplyConfiguration struct {
	Name     *string `json:"name,omitempty"`
	Replicas *int32  `json:"replicas,omitempty"`
}

// ClusterResourceSpecApplyConfiguration constructs an declarative configuration of the ClusterResourceSpec type for use with
// apply.
func ClusterResourceSpec() *ClusterResourceSpecApplyConfiguration {
	return &ClusterResourceSpecApplyConfiguration{}
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *ClusterResourceSpecApplyConfiguration) WithName(value string) *ClusterResourceSpecApplyConfiguration {
	b.Name = &value
	return b
}

// WithReplicas sets the Replicas field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Replicas field is set to the value of the last call.
func (b *ClusterResourceSpecApplyConfiguration) WithReplicas(value int32) *ClusterResourceSpecApplyConfiguration {
	b.Replicas = &value
	return b
}
