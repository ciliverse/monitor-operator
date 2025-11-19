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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MonitorStackSpec defines the desired state of MonitorStack
// MonitorStackSpec 定义MonitorStack的期望状态
// MonitorStack用于管理Prometheus和Grafana监控栈的完整生命周期
type MonitorStackSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// foo is an example field of MonitorStack. Edit monitorstack_types.go to remove/update
	// +optional
	// Prometheus配置
	// +kubebuilder:validation:Required
	Prometheus PrometheusSpec `json:"prometheus"`

	// Grafana配置
	// +kubebuilder:validation:Required
	Grafana GrafanaSpec `json:"grafana"`

	// 通用配置 - 应用于整个监控栈的配置
	// 目标命名空间，如果为空则使用当前命名空间

	Namespace string `json:"namespace,omitempty"`

	// 资源标签
	Labels map[string]string `json:"labels,omitempty"`
}

// PrometheusSpec defines Prometheus configuration
type PrometheusSpec struct {
	// 是否启用Prometheus
	Enabled bool `json:"enabled"`

	// 镜像配置
	// +kubebuilder:default="prom/prometheus"
	Image string `json:"image,omitempty"`
	// +kubebuilder:default="latest"
	Tag string `json:"tag,omitempty"`

	// 资源配置
	Resources ResourceRequirements `json:"resources,omitempty"`

	// 存储配置
	Storage StorageSpec `json:"storage,omitempty"`

	// 服务配置
	Service ServiceSpec `json:"service,omitempty"`

	// 配置文件
	Config string `json:"config,omitempty"`

	// 数据保留时间
	// +kubebuilder:validation:Pattern=`^[0-9]+[smhdy]$`
	// +kubebuilder:default="15d"
	Retention string `json:"retention,omitempty"`
}

// GrafanaSpec defines Grafana configuration
type GrafanaSpec struct {
	// 是否启用Grafana
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// 镜像配置
	// +kubebuilder:default="grafana/grafana"
	Image string `json:"image,omitempty"`
	// +kubebuilder:default="latest"
	Tag string `json:"tag,omitempty"`

	// 资源配置
	Resources ResourceRequirements `json:"resources,omitempty"`

	// 服务配置
	Service ServiceSpec `json:"service,omitempty"`

	// 管理员密码
	// +kubebuilder:default="admin"
	AdminPassword string `json:"adminPassword,omitempty"`

	// 数据源配置
	Datasources []DatasourceSpec `json:"datasources,omitempty"`

	// 仪表板配置
	Dashboards []DashboardSpec `json:"dashboards,omitempty"`
}

// ResourceRequirements defines resource limits and requests
type ResourceRequirements struct {
	Limits   ResourceList `json:"limits,omitempty"`
	Requests ResourceList `json:"requests,omitempty"`
}

// ResourceList defines CPU and memory resources
type ResourceList struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// StorageSpec defines storage configuration
type StorageSpec struct {
	Size         string `json:"size,omitempty"`
	StorageClass string `json:"storageClass,omitempty"`
}

// ServiceSpec defines service configuration
type ServiceSpec struct {
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer;ExternalName
	// +kubebuilder:default="ClusterIP"
	Type string `json:"type,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port,omitempty"`
	// +kubebuilder:validation:Minimum=30000
	// +kubebuilder:validation:Maximum=32767
	NodePort int32 `json:"nodePort,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`
}

// DatasourceSpec defines Grafana datasource
type DatasourceSpec struct {
	// 数据源名称
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// +kubebuilder:validation:Required
	Type string `json:"type"`
	// +kubebuilder:validation:Required
	URL string `json:"url"`
}

// DashboardSpec defines Grafana dashboard
type DashboardSpec struct {
	Name string `json:"name"`
	JSON string `json:"json,omitempty"`
	URL  string `json:"url,omitempty"`
}

// MonitorStackStatus defines the observed state of MonitorStack.
type MonitorStackStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the MonitorStack resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// 整体状态 - Pending, Ready, Failed等
	// +kubebuilder:validation:Enum=Pending;Ready;Failed;Updating
	Phase string `json:"phase,omitempty"`

	// 状态消息 - 详细的状态描述
	Message string `json:"message,omitempty"`

	// Prometheus组件状态
	PrometheusStatus ComponentStatus `json:"prometheusStatus,omitempty"`

	// Grafana组件状态
	GrafanaStatus ComponentStatus `json:"grafanaStatus,omitempty"`

	// 最后更新时间
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`

	// 条件列表 - 详细的状态条件
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ComponentStatus defines component status
type ComponentStatus struct {
	Ready bool `json:"ready"`

	// 副本数量
	Replicas int32 `json:"replicas,omitempty"`

	// 状态消息
	Message string `json:"message,omitempty"`

	// 服务端点 - 可访问的服务地址
	Endpoint string `json:"endpoint,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Namespaced
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Prometheus",type="boolean",JSONPath=".status.prometheusStatus.ready"
//+kubebuilder:printcolumn:name="Grafana",type="boolean",JSONPath=".status.grafanaStatus.ready"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// MonitorStack is the Schema for the monitorstacks API
type MonitorStack struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// 期望状态 - 用户定义的配置
	Spec MonitorStackSpec `json:"spec,omitempty"`

	// 观察状态 - 控制器维护的实际状态
	Status MonitorStackStatus `json:"status,omitempty"`
	// metadata is a standard object metadata
	// // +optional
	// metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// // spec defines the desired state of MonitorStack
	// // +required
	// Spec MonitorStackSpec `json:"spec"`

	// // status defines the observed state of MonitorStack
	// // +optional
	// Status MonitorStackStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Namespaced
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Prometheus",type="boolean",JSONPath=".status.prometheusStatus.ready"
//+kubebuilder:printcolumn:name="Grafana",type="boolean",JSONPath=".status.grafanaStatus.ready"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// MonitorStackList contains a list of MonitorStack
type MonitorStackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MonitorStack `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MonitorStack{}, &MonitorStackList{})
}
