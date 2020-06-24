/*


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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SpringBootAppSpec defines the desired state of SpringBootApp
type SpringBootAppSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	ImageRepo string `json:"imageRepo"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	AppImage string `json:"appImage"`
	// +kubebuilder:validation:Optional
	ImagePullSecrets string `json:"imagePullSecrets,omitempty"`
	// +kubebuilder:validation:Optional
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// +kubebuilder:validation:Required
	Ports []corev1.ContainerPort `json:"ports"`
	// +kubebuilder:validation:Optional
	Version string `json:"version,omitempty"`
	// +kubebuilder:validation:Optional
	Env []corev1.EnvVar `json:"env,omitempty"`
	// +kubebuilder:validation:Optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// Periodic probe of container liveness.
	// Container will be restarted if the probe fails.
	// +kubebuilder:validation:Optional
	LivenessProbePath string `json:"livenessProbePath,omitempty"`
	// LivenessProbe     *corev1.Probe `json:"livenessProbe,omitempty"`
	// Periodic probe of container service readiness.
	// Container will be removed from service endpoints if the probe fails.
	// +kubebuilder:validation:Optional
	ReadinessProbePath string `json:"readinessProbePath,omitempty"`

	// +kubebuilder:validation:Optional
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	// +kubebuilder:validation:Optional
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`

	// +kubebuilder:validation:Optional
	TargetCPUUtilizationPercentage *int32 `json:"targetCpuUtilizationPercentage,omitempty"`

	// +kubebuilder:validation:Optional
	PodAffinity *corev1.PodAffinity `json:"podAffinity,omitempty"`
	// +kubebuilder:validation:Optional
	PodAntiAffinity *corev1.PodAntiAffinity `json:"podAntiAffinity,omitempty"`

	// +kubebuilder:validation:Optional
	ConfigMap string `json:"configMap,omitempty"`
}

// SpringBootAppStatus defines the observed state of SpringBootApp
type SpringBootAppStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// SpringBootApp is the Schema for the springbootapps API
type SpringBootApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpringBootAppSpec   `json:"spec,omitempty"`
	Status SpringBootAppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SpringBootAppList contains a list of SpringBootApp
type SpringBootAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SpringBootApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SpringBootApp{}, &SpringBootAppList{})
}
