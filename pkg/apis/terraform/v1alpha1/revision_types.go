/*
 * Copyright (C) 2022 Appvia Ltd <info@appvia.io>
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public License
 * as published by the Free Software Foundation; either version 2
 * of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	corev1alpha1 "github.com/appvia/terranetes-controller/pkg/apis/core/v1alpha1"
)

// RevisionKind is the kind for a revision
const RevisionKind = "Revision"

const (
	// RevisionPlanNameLabel is the label for the plan name
	RevisionPlanNameLabel = "terraform.appvia.io/plan"
	// RevisionLabel is the label for the plan version
	RevisionLabel = "terraform.appvia.io/revision"
	// RevisionNameLabel is the label for the revision name
	RevisionNameLabel = "terraform.appvia.io/revision-name"
)

const (
	// RevisionSkipUpdateProtectionAnnotation is the annotation to skip update protection
	RevisionSkipUpdateProtectionAnnotation = "terraform.appvia.io/revision.skip-update-protection"
	// RevisionUsageExampleAnnotation is the annotation for the example
	RevisionUsageExampleAnnotation = "terraform.appvia.io/revision.usage"
	// RevisionChangeLogAnnotation is the annotation for the change log
	RevisionChangeLogAnnotation = "terraform.appvia.io/revision.changelog"
	// RevisionSourceLinkAnnotation is the annotation for the source link
	RevisionSourceLinkAnnotation = "terraform.appvia.io/revision.sourcelink"
)

// NewRevision returns an empty configuration
func NewRevision(name string) *Revision {
	return &Revision{
		TypeMeta: metav1.TypeMeta{
			Kind:       RevisionKind,
			APIVersion: SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

// RevisionDefinition retains all the information related to the configuration plan
// such as description, version, category, etc.
type RevisionDefinition struct {
	// Name is the name which this revision is grouped by, such as mysql, redis, etc. Multiple
	// revisions can be grouped by the same name, presented as a list of revisions for a single
	// plan name
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Description is a short description of the revision and its purpose, capabilities, etc.
	// +kubebuilder:validation:Required
	Description string `json:"description"`
	// Categories is a list of categories which this revision is grouped by, such as database,
	// cache, etc.
	// +kubebuilder:validation:Optional
	Categories []string `json:"categories,omitempty"`
	// ChangeLog provides a human readable list of changes for this revision
	// +kubebuilder:validation:Optional
	ChangeLog string `json:"changeLog,omitempty"`
	// Revision is the version of the revision, such as 1.0.0, 1.0.1, etc.
	// +kubebuilder:validation:Required
	Revision string `json:"revision"`
}

// RevisionProviderDependency is a dependency on a provider
type RevisionProviderDependency struct {
	// Cloud is the name of the cloud vendor we are dependent on, such as aws, azurerm, The
	// controller we ensure we have the provider installed before we can apply the configuration
	// +kubebuilder:validation:Required
	Cloud string `json:"cloud"`
}

// RevisionTerranetesDependency is a dependency on a terranetes controller
type RevisionTerranetesDependency struct {
	// Version is used to specify the version of the terranetes resource we are dependent on.
	// This format is based on Semantic Versioning 2.0.0 and can use '>=', '>', '<=', and '<'
	// +kubebuilder:validation:Required
	Version string `json:"version"`
}

// RevisionContextDependency is a dependency on a context resource
type RevisionContextDependency struct {
	// Name is the name of the context resource we are dependent on
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Cloud is the name of the cloud vendor we are dependent on, such as aws, azurerm, which
	// the context resource is associated with
	// +kubebuilder:validation:Optional
	Cloud string `json:"cloud,omitempty"`
}

// RevisionDependency defined a dependency for this revision. Currently we support Provider,
// Revision or Terranetes version
type RevisionDependency struct {
	// Revision indicates this revision has a dependency on a context resource
	// +kubebuilder:validation:Optional
	Context *RevisionContextDependency `json:"context,omitempty"`
	// Provider indicates this revision has a dependency on a provider resource
	// +kubebuilder:validation:Optional
	Provider *RevisionProviderDependency `json:"provider,omitempty"`
	// Terranetes indicates this revision has a dependency on a terranetes controller
	// +kubebuilder:validation:Optional
	Terranetes *RevisionTerranetesDependency `json:"terranetes,omitempty"`
}

// RevisionInput is a user defined input for a revision, such as a database name or
// a cache size etc.
type RevisionInput struct {
	// Description is a short description of the input and its purpose, capabilities, etc.
	// +kubebuilder:validation:Required
	Description string `json:"description"`
	// Key is the name of the variable when presented to the terraform module. If this field
	// is not specified, the name will be used as the key instead
	// +kubebuilder:validation:Optional
	Key string `json:"key,omitempty"`
	// Required indicates whether this input is required or not by the revision
	// +kubebuilder:validation:Optional
	Required *bool `json:"required,omitempty"`
	// Default is the default value for this input, this is a map which must contain
	// the field 'value' => 'default value'. Default values can be any simple of complex
	// type, such as string, int, bool, etc.
	// +kubebuilder:validation:Optional
	Default *runtime.RawExtension `json:"default,omitempty"`
}

// IsRequired returns true if the input is required
func (c *RevisionInput) IsRequired() bool {
	if c.Required == nil {
		return false
	}

	return *c.Required
}

// GetKeyName returns either the key or defaults to the name
func (c *RevisionInput) GetKeyName() string {
	return c.Key
}

// RevisionSpec defines the desired state of a configuration plan revision
// +k8s:openapi-gen=tr
type RevisionSpec struct {
	// Configuration is the configuration which this revision is providing to the
	// consumer.
	// +kubebuilder:validation:Required
	Configuration ConfigurationSpec `json:"configuration"`
	// Dependencies is a collection of dependencies which this revision depends on
	// such as a Provider, Terranetes version, or Revision
	// +kubebuilder:validation:Optional
	Dependencies []RevisionDependency `json:"dependencies,omitempty"`
	// Inputs is a collection of inputs which this revision the consumer of this
	// revision can or must provide. This is usually limited to contextual information
	// such as a name for the database, the size required, a bucket name, or policy.
	// +kubebuilder:validation:Optional
	Inputs []RevisionInput `json:"inputs,omitempty"`
	// Plan contains the information related to the name, version, description of
	// the revision.
	// +kubebuilder:validation:Required
	Plan RevisionDefinition `json:"plan"`
}

// +kubebuilder:webhook:name=revisions.terraform.appvia.io,mutating=false,path=/validate/terraform.appvia.io/revisions,verbs=create;delete;update,groups="terraform.appvia.io",resources=revisions,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1
// +kubebuilder:webhook:name=revisions.terraform.appvia.io,mutating=true,path=/mutate/terraform.appvia.io/revisions,verbs=create;update,groups="terraform.appvia.io",resources=revisions,versions=v1alpha1,failurePolicy=fail,sideEffects=None,admissionReviewVersions=v1

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Revision is the schema for a revision
// +k8s:openapi-gen=true
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=revisions,scope=Cluster,categories={terraform}
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Plan",type="string",JSONPath=".spec.plan.name"
// +kubebuilder:printcolumn:name="Description",type="string",JSONPath=".spec.plan.description"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.plan.revision"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type Revision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RevisionSpec   `json:"spec,omitempty"`
	Status RevisionStatus `json:"status,omitempty"`
}

// ListOfInputs is a list of inputs for this revision
func (c *Revision) ListOfInputs() []string {
	var inputs []string

	for _, input := range c.Spec.Inputs {
		inputs = append(inputs, input.Key)
	}

	return inputs
}

// RevisionStatus defines the observed state of a terraform
// +k8s:openapi-gen=true
type RevisionStatus struct {
	corev1alpha1.CommonStatus `json:",inline"`
}

// GetCommonStatus returns the common status
func (c *Revision) GetCommonStatus() *corev1alpha1.CommonStatus {
	return &c.Status.CommonStatus
}

// GetNamespacedName returns the namespaced resource type
func (c *Revision) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: c.Namespace,
		Name:      c.Name,
	}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RevisionList contains a list of revisions
type RevisionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Revision `json:"items"`
}