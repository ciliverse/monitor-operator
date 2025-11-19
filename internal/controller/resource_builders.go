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
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	monitoringv1 "github.com/ciliverse/monitor-operator/api/v1"
)

// 资源构建器 - 负责构建Kubernetes资源对象
// 这些方法将MonitorStack的配置转换为具体的Kubernetes资源

// buildPrometheusDeployment 构建Prometheus Deployment
// 根据MonitorStack配置创建Prometheus的Deployment资源
func (r *MonitorStackReconciler) buildPrometheusDeployment(monitorStack *monitoringv1.MonitorStack) *appsv1.Deployment {
	labels := r.getLabels(monitorStack, "prometheus")
	replicas := int32(1) // Prometheus通常运行单实例

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getPrometheusName(monitorStack),
			Namespace: monitorStack.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					// 安全上下文 - 以非root用户运行
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
						RunAsUser:    &[]int64{65534}[0], // nobody用户
						FSGroup:      &[]int64{65534}[0],
					},
					Containers: []corev1.Container{
						{
							Name:  "prometheus",
							Image: fmt.Sprintf("%s:%s", monitorStack.Spec.Prometheus.Image, monitorStack.Spec.Prometheus.Tag),
							Ports: []corev1.ContainerPort{
								{
									Name:          "web",
									ContainerPort: 9090,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							// Prometheus启动参数
							Args: r.buildPrometheusArgs(monitorStack),
							// 卷挂载 - 配置文件和数据目录
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/prometheus",
									ReadOnly:  true,
								},
							},
							// 资源配置
							Resources: r.buildResourceRequirements(monitorStack.Spec.Prometheus.Resources),
							// 健康检查 - 存活探针
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/-/healthy",
										Port: intstr.FromInt(9090),
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
								TimeoutSeconds:      5,
								FailureThreshold:    3,
							},
							// 健康检查 - 就绪探针
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/-/ready",
										Port: intstr.FromInt(9090),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
								TimeoutSeconds:      3,
								FailureThreshold:    3,
							},
						},
					},
					// 卷定义 - 配置文件卷
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: r.getPrometheusConfigMapName(monitorStack),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// 添加数据存储卷
	r.addPrometheusDataVolume(deployment, monitorStack)

	return deployment
}

// addPrometheusDataVolume 添加Prometheus数据存储卷
// 根据配置决定使用PVC还是emptyDir
func (r *MonitorStackReconciler) addPrometheusDataVolume(deployment *appsv1.Deployment, monitorStack *monitoringv1.MonitorStack) {
	dataVolumeMount := corev1.VolumeMount{
		Name:      "data",
		MountPath: "/prometheus",
	}

	var dataVolume corev1.Volume

	if monitorStack.Spec.Prometheus.Storage.Size != "" {
		// 使用持久化存储
		dataVolume = corev1.Volume{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: r.getPrometheusPVCName(monitorStack),
				},
			},
		}
	} else {
		// 使用临时存储
		dataVolume = corev1.Volume{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
	}

	// 添加卷挂载和卷定义
	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
		dataVolumeMount,
	)
	deployment.Spec.Template.Spec.Volumes = append(
		deployment.Spec.Template.Spec.Volumes,
		dataVolume,
	)
}

// buildPrometheusService 构建Prometheus Service
// 创建用于访问Prometheus的Kubernetes Service
func (r *MonitorStackReconciler) buildPrometheusService(monitorStack *monitoringv1.MonitorStack) *corev1.Service {
	labels := r.getLabels(monitorStack, "prometheus")

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getPrometheusServiceName(monitorStack),
			Namespace: monitorStack.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceType(monitorStack.Spec.Prometheus.Service.Type),
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "web",
					Port:       monitorStack.Spec.Prometheus.Service.Port,
					TargetPort: intstr.FromInt(9090),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	// 如果是NodePort类型且指定了NodePort，设置它
	if monitorStack.Spec.Prometheus.Service.Type == "NodePort" && monitorStack.Spec.Prometheus.Service.NodePort > 0 {
		service.Spec.Ports[0].NodePort = monitorStack.Spec.Prometheus.Service.NodePort
	}

	// 合并用户自定义的服务标签
	for k, v := range monitorStack.Spec.Prometheus.Service.Labels {
		service.Labels[k] = v
	}

	return service
}

// buildGrafanaDeployment 构建Grafana Deployment
// 根据MonitorStack配置创建Grafana的Deployment资源
func (r *MonitorStackReconciler) buildGrafanaDeployment(monitorStack *monitoringv1.MonitorStack) *appsv1.Deployment {
	labels := r.getLabels(monitorStack, "grafana")
	replicas := int32(1) // Grafana通常运行单实例

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getGrafanaName(monitorStack),
			Namespace: monitorStack.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					// 安全上下文 - 以grafana用户运行
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
						RunAsUser:    &[]int64{472}[0], // grafana用户
						FSGroup:      &[]int64{472}[0],
					},
					Containers: []corev1.Container{
						{
							Name:  "grafana",
							Image: fmt.Sprintf("%s:%s", monitorStack.Spec.Grafana.Image, monitorStack.Spec.Grafana.Tag),
							Ports: []corev1.ContainerPort{
								{
									Name:          "grafana",
									ContainerPort: 3000,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							// 环境变量配置
							Env: r.buildGrafanaEnv(monitorStack),
							// 卷挂载 - 数据目录
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "grafana-storage",
									MountPath: "/var/lib/grafana",
								},
							},
							// 资源配置
							Resources: r.buildResourceRequirements(monitorStack.Spec.Grafana.Resources),
							// 健康检查 - 存活探针
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/api/health",
										Port: intstr.FromInt(3000),
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
								TimeoutSeconds:      5,
								FailureThreshold:    3,
							},
							// 健康检查 - 就绪探针
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/api/health",
										Port: intstr.FromInt(3000),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
								TimeoutSeconds:      3,
								FailureThreshold:    3,
							},
						},
					},
					// 卷定义 - 数据存储卷（使用emptyDir）
					Volumes: []corev1.Volume{
						{
							Name: "grafana-storage",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	// 如果配置了数据源，添加数据源配置卷
	if len(monitorStack.Spec.Grafana.Datasources) > 0 {
		r.addGrafanaDatasourceVolume(deployment, monitorStack)
	}

	return deployment
}

// addGrafanaDatasourceVolume 添加Grafana数据源配置卷
func (r *MonitorStackReconciler) addGrafanaDatasourceVolume(deployment *appsv1.Deployment, monitorStack *monitoringv1.MonitorStack) {
	datasourceVolumeMount := corev1.VolumeMount{
		Name:      "datasources",
		MountPath: "/etc/grafana/provisioning/datasources",
		ReadOnly:  true,
	}

	datasourceVolume := corev1.Volume{
		Name: "datasources",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: r.getGrafanaDatasourcesConfigMapName(monitorStack),
				},
			},
		},
	}

	// 添加卷挂载和卷定义
	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = append(
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts,
		datasourceVolumeMount,
	)
	deployment.Spec.Template.Spec.Volumes = append(
		deployment.Spec.Template.Spec.Volumes,
		datasourceVolume,
	)
}

// buildGrafanaService 构建Grafana Service
// 创建用于访问Grafana的Kubernetes Service
func (r *MonitorStackReconciler) buildGrafanaService(monitorStack *monitoringv1.MonitorStack) *corev1.Service {
	labels := r.getLabels(monitorStack, "grafana")

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getGrafanaServiceName(monitorStack),
			Namespace: monitorStack.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceType(monitorStack.Spec.Grafana.Service.Type),
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "grafana",
					Port:       monitorStack.Spec.Grafana.Service.Port,
					TargetPort: intstr.FromInt(3000),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	// 如果是NodePort类型且指定了NodePort，设置它
	if monitorStack.Spec.Grafana.Service.Type == "NodePort" && monitorStack.Spec.Grafana.Service.NodePort > 0 {
		service.Spec.Ports[0].NodePort = monitorStack.Spec.Grafana.Service.NodePort
	}

	// 合并用户自定义的服务标签
	for k, v := range monitorStack.Spec.Grafana.Service.Labels {
		service.Labels[k] = v
	}

	return service
}

// buildResourceRequirements 构建资源需求
// 将MonitorStack中的资源配置转换为Kubernetes ResourceRequirements
func (r *MonitorStackReconciler) buildResourceRequirements(resources monitoringv1.ResourceRequirements) corev1.ResourceRequirements {
	requirements := corev1.ResourceRequirements{}

	// 设置资源请求
	if resources.Requests.CPU != "" || resources.Requests.Memory != "" {
		requirements.Requests = corev1.ResourceList{}
		if resources.Requests.CPU != "" {
			requirements.Requests[corev1.ResourceCPU] = resource.MustParse(resources.Requests.CPU)
		}
		if resources.Requests.Memory != "" {
			requirements.Requests[corev1.ResourceMemory] = resource.MustParse(resources.Requests.Memory)
		}
	}

	// 设置资源限制
	if resources.Limits.CPU != "" || resources.Limits.Memory != "" {
		requirements.Limits = corev1.ResourceList{}
		if resources.Limits.CPU != "" {
			requirements.Limits[corev1.ResourceCPU] = resource.MustParse(resources.Limits.CPU)
		}
		if resources.Limits.Memory != "" {
			requirements.Limits[corev1.ResourceMemory] = resource.MustParse(resources.Limits.Memory)
		}
	}

	return requirements
}

// buildPrometheusArgs 构建Prometheus启动参数
// 根据配置生成Prometheus容器的启动参数
func (r *MonitorStackReconciler) buildPrometheusArgs(monitorStack *monitoringv1.MonitorStack) []string {
	args := []string{
		"--config.file=/etc/prometheus/prometheus.yml",              // 配置文件路径
		"--storage.tsdb.path=/prometheus",                           // 数据存储路径
		"--web.console.libraries=/etc/prometheus/console_libraries", // 控制台库路径
		"--web.console.templates=/etc/prometheus/consoles",          // 控制台模板路径
		"--web.enable-lifecycle",                                    // 启用生命周期API
		"--web.enable-admin-api",                                    // 启用管理API
	}

	// 添加数据保留时间配置
	if monitorStack.Spec.Prometheus.Retention != "" {
		args = append(args, fmt.Sprintf("--storage.tsdb.retention.time=%s", monitorStack.Spec.Prometheus.Retention))
	}

	return args
}

// buildGrafanaEnv 构建Grafana环境变量
// 根据配置生成Grafana容器的环境变量
func (r *MonitorStackReconciler) buildGrafanaEnv(monitorStack *monitoringv1.MonitorStack) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "GF_SECURITY_ADMIN_PASSWORD",
			Value: monitorStack.Spec.Grafana.AdminPassword,
		},
		{
			Name:  "GF_USERS_ALLOW_SIGN_UP",
			Value: "false", // 禁止用户注册
		},
		{
			Name:  "GF_PATHS_DATA",
			Value: "/var/lib/grafana", // 数据目录
		},
		{
			Name:  "GF_PATHS_LOGS",
			Value: "/var/log/grafana", // 日志目录
		},
		{
			Name:  "GF_PATHS_PLUGINS",
			Value: "/var/lib/grafana/plugins", // 插件目录
		},
		{
			Name:  "GF_PATHS_PROVISIONING",
			Value: "/etc/grafana/provisioning", // 配置供应目录
		},
	}

	return env
}

// createGrafanaDatasourcesConfigMap 创建Grafana数据源配置ConfigMap
func (r *MonitorStackReconciler) createGrafanaDatasourcesConfigMap(ctx context.Context, monitorStack *monitoringv1.MonitorStack) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getGrafanaDatasourcesConfigMapName(monitorStack),
			Namespace: monitorStack.Namespace,
			Labels:    r.getLabels(monitorStack, "grafana"),
		},
		Data: map[string]string{
			"datasources.yaml": r.buildGrafanaDatasourcesConfig(monitorStack),
		},
	}

	// 设置OwnerReference
	if err := controllerutil.SetControllerReference(monitorStack, configMap, r.Scheme); err != nil {
		return err
	}

	// 创建或更新ConfigMap
	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, configMap)
		}
		return err
	}

	// 更新现有ConfigMap
	existing.Data = configMap.Data
	return r.Update(ctx, existing)
}

// createGrafanaDeployment 创建Grafana Deployment
func (r *MonitorStackReconciler) createGrafanaDeployment(ctx context.Context, monitorStack *monitoringv1.MonitorStack) error {
	deployment := r.buildGrafanaDeployment(monitorStack)

	// 设置OwnerReference
	if err := controllerutil.SetControllerReference(monitorStack, deployment, r.Scheme); err != nil {
		return err
	}

	// 创建或更新Deployment
	existing := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, deployment)
		}
		return err
	}

	// 更新现有Deployment
	existing.Spec = deployment.Spec
	existing.Labels = deployment.Labels
	return r.Update(ctx, existing)
}

// createGrafanaService 创建Grafana Service
func (r *MonitorStackReconciler) createGrafanaService(ctx context.Context, monitorStack *monitoringv1.MonitorStack) error {
	service := r.buildGrafanaService(monitorStack)

	// 设置OwnerReference
	if err := controllerutil.SetControllerReference(monitorStack, service, r.Scheme); err != nil {
		return err
	}

	// 创建或更新Service
	existing := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, service)
		}
		return err
	}

	// 更新现有Service
	existing.Spec.Ports = service.Spec.Ports
	existing.Spec.Type = service.Spec.Type
	existing.Labels = service.Labels
	if service.Spec.Type == corev1.ServiceTypeNodePort && len(service.Spec.Ports) > 0 {
		existing.Spec.Ports[0].NodePort = service.Spec.Ports[0].NodePort
	}
	return r.Update(ctx, existing)
}

// buildGrafanaDatasourcesConfig 构建Grafana数据源配置
// 生成Grafana数据源的YAML配置
func (r *MonitorStackReconciler) buildGrafanaDatasourcesConfig(monitorStack *monitoringv1.MonitorStack) string {
	config := `apiVersion: 1
datasources:`

	for i, ds := range monitorStack.Spec.Grafana.Datasources {
		// 第一个Prometheus数据源设为默认
		isDefault := i == 0 && ds.Type == "prometheus"

		config += fmt.Sprintf(`
  - name: %s
    type: %s
    url: %s
    access: proxy
    isDefault: %t`, ds.Name, ds.Type, ds.URL, isDefault)
	}

	return config
}
