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

// NamespacedResourceStatusApplyConfiguration represents an declarative configuration of the NamespacedResourceStatus type for use
// with apply.
type NamespacedResourceStatusApplyConfiguration struct {
	AvailableReplicas *int32 `json:"availableReplicas,omitempty"`
}

// NamespacedResourceStatusApplyConfiguration constructs an declarative configuration of the NamespacedResourceStatus type for use with
// apply.
func NamespacedResourceStatus() *NamespacedResourceStatusApplyConfiguration {
	return &NamespacedResourceStatusApplyConfiguration{}
}

// WithAvailableReplicas sets the AvailableReplicas field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AvailableReplicas field is set to the value of the last call.
func (b *NamespacedResourceStatusApplyConfiguration) WithAvailableReplicas(value int32) *NamespacedResourceStatusApplyConfiguration {
	b.AvailableReplicas = &value
	return b
}
