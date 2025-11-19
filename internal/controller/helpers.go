/*
Copyright 2023.

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

package controller

import (
	"fmt"

	monitoringv1 "github.com/ciliverse/monitor-operator/api/v1"
)

// 辅助函数 - 提供通用的工具方法
// 这些方法用于生成资源名称、标签等通用功能

// getPrometheusName 获取Prometheus Deployment的名称
// 命名规则: {MonitorStack名称}-prometheus
func (r *MonitorStackReconciler) getPrometheusName(monitorStack *monitoringv1.MonitorStack) string {
	return fmt.Sprintf("%s-prometheus", monitorStack.Name)
}

// getPrometheusServiceName 获取Prometheus Service的名称
// 命名规则: {MonitorStack名称}-prometheus
func (r *MonitorStackReconciler) getPrometheusServiceName(monitorStack *monitoringv1.MonitorStack) string {
	return fmt.Sprintf("%s-prometheus", monitorStack.Name)
}

// getPrometheusConfigMapName 获取Prometheus ConfigMap的名称
// 命名规则: {MonitorStack名称}-prometheus-config
func (r *MonitorStackReconciler) getPrometheusConfigMapName(monitorStack *monitoringv1.MonitorStack) string {
	return fmt.Sprintf("%s-prometheus-config", monitorStack.Name)
}

// getPrometheusPVCName 获取Prometheus PVC的名称
// 命名规则: {MonitorStack名称}-prometheus-data
func (r *MonitorStackReconciler) getPrometheusPVCName(monitorStack *monitoringv1.MonitorStack) string {
	return fmt.Sprintf("%s-prometheus-data", monitorStack.Name)
}

// getGrafanaName 获取Grafana Deployment的名称
// 命名规则: {MonitorStack名称}-grafana
func (r *MonitorStackReconciler) getGrafanaName(monitorStack *monitoringv1.MonitorStack) string {
	return fmt.Sprintf("%s-grafana", monitorStack.Name)
}

// getGrafanaServiceName 获取Grafana Service的名称
// 命名规则: {MonitorStack名称}-grafana
func (r *MonitorStackReconciler) getGrafanaServiceName(monitorStack *monitoringv1.MonitorStack) string {
	return fmt.Sprintf("%s-grafana", monitorStack.Name)
}

// getGrafanaDatasourcesConfigMapName 获取Grafana数据源ConfigMap的名称
// 命名规则: {MonitorStack名称}-grafana-datasources
func (r *MonitorStackReconciler) getGrafanaDatasourcesConfigMapName(monitorStack *monitoringv1.MonitorStack) string {
	return fmt.Sprintf("%s-grafana-datasources", monitorStack.Name)
}

// getLabels 获取资源标签
// 生成标准的Kubernetes标签，包括应用名称、实例、组件等
func (r *MonitorStackReconciler) getLabels(monitorStack *monitoringv1.MonitorStack, component string) map[string]string {
	// 基础标签 - 遵循Kubernetes推荐的标签规范
	labels := map[string]string{
		"app.kubernetes.io/name":       "monitor-operator", // 应用名称
		"app.kubernetes.io/instance":   monitorStack.Name,  // 实例名称
		"app.kubernetes.io/component":  component,          // 组件名称（prometheus/grafana）
		"app.kubernetes.io/managed-by": "monitor-operator", // 管理者
		"app.kubernetes.io/part-of":    "monitoring-stack", // 所属系统
	}

	// 合并用户自定义标签
	// 用户标签会覆盖同名的默认标签
	for k, v := range monitorStack.Spec.Labels {
		labels[k] = v
	}

	return labels
}

// getPrometheusConfig 获取Prometheus配置
// 如果用户提供了自定义配置，使用用户配置；否则使用默认配置
func (r *MonitorStackReconciler) getPrometheusConfig(monitorStack *monitoringv1.MonitorStack) string {
	// 如果用户提供了自定义配置，直接使用
	if monitorStack.Spec.Prometheus.Config != "" {
		return monitorStack.Spec.Prometheus.Config
	}

	// 使用默认的Prometheus配置
	// 这个配置包含基本的监控目标和Kubernetes服务发现
	return `# Prometheus默认配置
# 全局配置
global:
  scrape_interval: 15s        # 抓取间隔
  evaluation_interval: 15s    # 规则评估间隔

# 抓取配置
scrape_configs:
  # Prometheus自监控
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  # Kubernetes Pod监控
  # 通过注解prometheus.io/scrape=true来发现需要监控的Pod
  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      # 只监控有prometheus.io/scrape=true注解的Pod
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      # 使用prometheus.io/path注解指定的路径，默认为/metrics
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
      # 使用prometheus.io/port注解指定的端口
      - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        target_label: __address__
      # 添加Pod名称标签
      - source_labels: [__meta_kubernetes_pod_name]
        action: replace
        target_label: kubernetes_pod_name
      # 添加命名空间标签
      - source_labels: [__meta_kubernetes_namespace]
        action: replace
        target_label: kubernetes_namespace

  # Kubernetes Service监控
  - job_name: 'kubernetes-services'
    kubernetes_sd_configs:
      - role: service
    relabel_configs:
      # 只监控有prometheus.io/scrape=true注解的Service
      - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      # 使用prometheus.io/path注解指定的路径
      - source_labels: [__meta_kubernetes_service_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
      # 使用prometheus.io/port注解指定的端口
      - source_labels: [__address__, __meta_kubernetes_service_annotation_prometheus_io_port]
        action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        target_label: __address__
      # 添加Service名称标签
      - source_labels: [__meta_kubernetes_service_name]
        action: replace
        target_label: kubernetes_service_name
      # 添加命名空间标签
      - source_labels: [__meta_kubernetes_namespace]
        action: replace
        target_label: kubernetes_namespace

  # Kubernetes Node监控
  - job_name: 'kubernetes-nodes'
    kubernetes_sd_configs:
      - role: node
    relabel_configs:
      # 添加Node名称标签
      - source_labels: [__meta_kubernetes_node_name]
        action: replace
        target_label: kubernetes_node_name

# 规则文件配置（可选）
rule_files:
  # - "first_rules.yml"
  # - "second_rules.yml"

# 告警管理器配置（可选）
# alerting:
#   alertmanagers:
#     - static_configs:
#         - targets:
#           # - alertmanager:9093`
}

// validateMonitorStack 验证MonitorStack配置
// 检查配置的合理性，返回验证错误
func (r *MonitorStackReconciler) validateMonitorStack(monitorStack *monitoringv1.MonitorStack) error {
	// 验证至少启用一个组件
	if !monitorStack.Spec.Prometheus.Enabled && !monitorStack.Spec.Grafana.Enabled {
		return fmt.Errorf("at least one component (Prometheus or Grafana) must be enabled")
	}

	// 验证Prometheus配置
	if monitorStack.Spec.Prometheus.Enabled {
		if err := r.validatePrometheusConfig(monitorStack); err != nil {
			return fmt.Errorf("prometheus configuration error: %w", err)
		}
	}

	// 验证Grafana配置
	if monitorStack.Spec.Grafana.Enabled {
		if err := r.validateGrafanaConfig(monitorStack); err != nil {
			return fmt.Errorf("grafana configuration error: %w", err)
		}
	}

	return nil
}

// validatePrometheusConfig 验证Prometheus配置
func (r *MonitorStackReconciler) validatePrometheusConfig(monitorStack *monitoringv1.MonitorStack) error {
	prometheus := monitorStack.Spec.Prometheus

	// 验证端口范围
	if prometheus.Service.Port < 1 || prometheus.Service.Port > 65535 {
		return fmt.Errorf("service port must be between 1 and 65535, got %d", prometheus.Service.Port)
	}

	// 验证NodePort范围（如果指定）
	if prometheus.Service.Type == "NodePort" && prometheus.Service.NodePort > 0 {
		if prometheus.Service.NodePort < 30000 || prometheus.Service.NodePort > 32767 {
			return fmt.Errorf("nodePort must be between 30000 and 32767, got %d", prometheus.Service.NodePort)
		}
	}

	// 验证镜像配置
	if prometheus.Image == "" {
		return fmt.Errorf("prometheus image cannot be empty")
	}

	if prometheus.Tag == "" {
		return fmt.Errorf("prometheus tag cannot be empty")
	}

	return nil
}

// validateGrafanaConfig 验证Grafana配置
func (r *MonitorStackReconciler) validateGrafanaConfig(monitorStack *monitoringv1.MonitorStack) error {
	grafana := monitorStack.Spec.Grafana

	// 验证端口范围
	if grafana.Service.Port < 1 || grafana.Service.Port > 65535 {
		return fmt.Errorf("service port must be between 1 and 65535, got %d", grafana.Service.Port)
	}

	// 验证NodePort范围（如果指定）
	if grafana.Service.Type == "NodePort" && grafana.Service.NodePort > 0 {
		if grafana.Service.NodePort < 30000 || grafana.Service.NodePort > 32767 {
			return fmt.Errorf("nodePort must be between 30000 and 32767, got %d", grafana.Service.NodePort)
		}
	}

	// 验证镜像配置
	if grafana.Image == "" {
		return fmt.Errorf("grafana image cannot be empty")
	}

	if grafana.Tag == "" {
		return fmt.Errorf("grafana tag cannot be empty")
	}

	// 验证管理员密码
	if grafana.AdminPassword == "" {
		return fmt.Errorf("grafana admin password cannot be empty")
	}

	// 验证数据源配置
	for i, ds := range grafana.Datasources {
		if ds.Name == "" {
			return fmt.Errorf("datasource[%d] name cannot be empty", i)
		}
		if ds.Type == "" {
			return fmt.Errorf("datasource[%d] type cannot be empty", i)
		}
		if ds.URL == "" {
			return fmt.Errorf("datasource[%d] URL cannot be empty", i)
		}
	}

	return nil
}

// setDefaultValues 设置默认值
// 为未指定的配置项设置合理的默认值
func (r *MonitorStackReconciler) setDefaultValues(monitorStack *monitoringv1.MonitorStack) {
	// 设置Prometheus默认值
	if monitorStack.Spec.Prometheus.Enabled {
		r.setPrometheusDefaults(&monitorStack.Spec.Prometheus)
	}

	// 设置Grafana默认值
	if monitorStack.Spec.Grafana.Enabled {
		r.setGrafanaDefaults(&monitorStack.Spec.Grafana)
	}
}

// setPrometheusDefaults 设置Prometheus默认值
func (r *MonitorStackReconciler) setPrometheusDefaults(prometheus *monitoringv1.PrometheusSpec) {
	if prometheus.Image == "" {
		prometheus.Image = "prom/prometheus"
	}
	if prometheus.Tag == "" {
		prometheus.Tag = "latest"
	}
	if prometheus.Service.Port == 0 {
		prometheus.Service.Port = 9090
	}
	if prometheus.Service.Type == "" {
		prometheus.Service.Type = "ClusterIP"
	}
	if prometheus.Retention == "" {
		prometheus.Retention = "15d"
	}
	if prometheus.Resources.Requests.CPU == "" {
		prometheus.Resources.Requests.CPU = "100m"
	}
	if prometheus.Resources.Requests.Memory == "" {
		prometheus.Resources.Requests.Memory = "256Mi"
	}
}

// setGrafanaDefaults 设置Grafana默认值
func (r *MonitorStackReconciler) setGrafanaDefaults(grafana *monitoringv1.GrafanaSpec) {
	if grafana.Image == "" {
		grafana.Image = "grafana/grafana"
	}
	if grafana.Tag == "" {
		grafana.Tag = "latest"
	}
	if grafana.Service.Port == 0 {
		grafana.Service.Port = 3000
	}
	if grafana.Service.Type == "" {
		grafana.Service.Type = "ClusterIP"
	}
	if grafana.AdminPassword == "" {
		grafana.AdminPassword = "admin"
	}
	if grafana.Resources.Requests.CPU == "" {
		grafana.Resources.Requests.CPU = "100m"
	}
	if grafana.Resources.Requests.Memory == "" {
		grafana.Resources.Requests.Memory = "128Mi"
	}
}
