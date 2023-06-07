/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	corev1 "k8s.io/api/core/v1"
)

// +kubebuilder:object:generate=true

// ImageSpec defines the used OCI image
type ImageSpec struct {
	// +kubebuilder:validation:MinLength=1
	Repository string `json:"repository,omitempty"`
	// +kubebuilder:validation:MinLength=1
	Tag string `json:"tag,omitempty"`
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	// +kubebuilder:default=IfNotPresent
	PullPolicy string `json:"pullPolicy,omitempty"`
	PullSecret string `json:"pullSecret,omitempty"`
}

// +kubebuilder:object:generate=true

// KubernetesPodProperties defines K8s properties to be applied to the created workloads (pod level)
type KubernetesPodProperties struct {
	NodeSelector              map[string]string                 `json:"nodeSelector,omitempty"`
	Affinity                  *corev1.Affinity                  `json:"affinity,omitempty"`
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
	Tolerations               []corev1.Toleration               `json:"tolerations,omitempty"`
	PriorityClassName         *string                           `json:"priorityClassName,omitempty"`
	PodSecurityContext        *corev1.PodSecurityContext        `json:"podSecurityContext,omitempty"`
	Labels                    map[string]string                 `json:"podLabels,omitempty"`
	Annotations               map[string]string                 `json:"podAnnotations,omitempty"`
}

// +kubebuilder:object:generate=true

// KubernetesContainerProperties defines K8s properties to be applied to the created workloads (container level)
type KubernetesContainerProperties struct {
	SecurityContext *corev1.SecurityContext      `json:"securityContext,omitempty"`
	Resources       *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// +kubebuilder:object:generate=true

// KubernetesProperties defines a union of KubernetesPodProperties and KubernetesContainerProperties.
// Useful in cases where a controller pod has exactly one container.
type KubernetesProperties struct {
	KubernetesPodProperties       `json:",inline"`
	KubernetesContainerProperties `json:",inline"`
}

// +kubebuilder:object:generate=true

// ServiceProperties defines K8s properties to be applied to a managed service resource.
type ServiceProperties struct {
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer;ExternalName
	// +kubebuilder:default=ClusterIP
	Type                     corev1.ServiceType `json:"type,omitempty"`
	LoadBalancerSourceRanges []string           `json:"loadBalancerSourceRanges,omitempty"`
	// +kubebuilder:validation:Enum=Cluster;Local
	// +kubebuilder:default=Cluster
	ExternalTrafficPolicy corev1.ServiceExternalTrafficPolicyType `json:"externalTrafficPolicy,omitempty"`
	Labels                map[string]string                       `json:"labels,omitempty"`
	Annotations           map[string]string                       `json:"annotations,omitempty"`
}

// +kubebuilder:object:generate=true

// IngressProperties defines K8s properties to be applied to a managed ingress resource.
type IngressProperties struct {
	Class       string            `json:"class,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}
