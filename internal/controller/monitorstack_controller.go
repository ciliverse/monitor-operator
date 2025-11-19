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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	monitoringv1 "github.com/ciliverse/monitor-operator/api/v1"
)

// MonitorStackReconciler 协调MonitorStack对象
// 这是主要的控制器，负责监听MonitorStack资源的变化并执行相应的操作
type MonitorStackReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=monitoring.cillian.website,resources=monitorstacks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.cillian.website,resources=monitorstacks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=monitoring.cillian.website,resources=monitorstacks/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// Reconcile 是主要的kubernetes协调循环的一部分
// 它负责确保MonitorStack资源的实际状态与期望状态一致
func (r *MonitorStackReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 步骤1: 获取MonitorStack实例
	var monitorStack monitoringv1.MonitorStack
	if err := r.Get(ctx, req.NamespacedName, &monitorStack); err != nil {
		if errors.IsNotFound(err) {
			// 资源已被删除，忽略此次协调
			logger.Info("MonitorStack resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get MonitorStack")
		return ctrl.Result{}, err
	}

	// 步骤2: 处理删除逻辑
	if monitorStack.DeletionTimestamp != nil {
		return r.handleDeletion(ctx, &monitorStack)
	}

	// 步骤3: 添加finalizer以确保正确的清理流程
	finalizerName := "monitoring.cillian.website/finalizer"
	if !controllerutil.ContainsFinalizer(&monitorStack, finalizerName) {
		controllerutil.AddFinalizer(&monitorStack, finalizerName)
		return ctrl.Result{}, r.Update(ctx, &monitorStack)
	}

	// 步骤4: 初始化状态
	if monitorStack.Status.Phase == "" {
		monitorStack.Status.Phase = "Pending"
		monitorStack.Status.Message = "Initializing MonitorStack"
		monitorStack.Status.LastUpdated = metav1.Now()
		if err := r.Status().Update(ctx, &monitorStack); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 步骤5: 协调Prometheus组件
	if monitorStack.Spec.Prometheus.Enabled {
		logger.Info("Reconciling Prometheus component")
		if err := r.reconcilePrometheus(ctx, &monitorStack); err != nil {
			logger.Error(err, "Failed to reconcile Prometheus")
			r.updateStatus(ctx, &monitorStack, "Failed", fmt.Sprintf("Prometheus reconciliation failed: %v", err))
			return ctrl.Result{RequeueAfter: time.Minute}, err
		}
	} else {
		// 如果Prometheus被禁用，清理相关资源
		logger.Info("Prometheus is disabled, cleaning up resources")
		if err := r.cleanupPrometheusResources(ctx, &monitorStack); err != nil {
			logger.Error(err, "Failed to cleanup Prometheus resources")
		}
	}

	// 步骤6: 协调Grafana组件
	if monitorStack.Spec.Grafana.Enabled {
		logger.Info("Reconciling Grafana component")
		if err := r.reconcileGrafana(ctx, &monitorStack); err != nil {
			logger.Error(err, "Failed to reconcile Grafana")
			r.updateStatus(ctx, &monitorStack, "Failed", fmt.Sprintf("Grafana reconciliation failed: %v", err))
			return ctrl.Result{RequeueAfter: time.Minute}, err
		}
	} else {
		// 如果Grafana被禁用，清理相关资源
		logger.Info("Grafana is disabled, cleaning up resources")
		if err := r.cleanupGrafanaResources(ctx, &monitorStack); err != nil {
			logger.Error(err, "Failed to cleanup Grafana resources")
		}
	}

	// 步骤7: 更新整体状态
	if err := r.updateOverallStatus(ctx, &monitorStack); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled MonitorStack")
	// 每5分钟重新协调一次，确保状态同步
	return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}

// handleDeletion 处理MonitorStack资源的删除
// 执行必要的清理工作，然后移除finalizer
func (r *MonitorStackReconciler) handleDeletion(ctx context.Context, monitorStack *monitoringv1.MonitorStack) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling MonitorStack deletion")

	// 清理Prometheus资源
	if err := r.cleanupPrometheusResources(ctx, monitorStack); err != nil {
		logger.Error(err, "Failed to cleanup Prometheus resources during deletion")
		return ctrl.Result{RequeueAfter: time.Second * 30}, err
	}

	// 清理Grafana资源
	if err := r.cleanupGrafanaResources(ctx, monitorStack); err != nil {
		logger.Error(err, "Failed to cleanup Grafana resources during deletion")
		return ctrl.Result{RequeueAfter: time.Second * 30}, err
	}

	// 移除finalizer，允许资源被删除
	controllerutil.RemoveFinalizer(monitorStack, "monitoring.cillian.website/finalizer")
	return ctrl.Result{}, r.Update(ctx, monitorStack)
}

// reconcilePrometheus 协调Prometheus相关资源
// 创建和管理Prometheus的ConfigMap、PVC、Deployment和Service
func (r *MonitorStackReconciler) reconcilePrometheus(ctx context.Context, monitorStack *monitoringv1.MonitorStack) error {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Prometheus resources")

	// 创建Prometheus配置ConfigMap
	if err := r.createPrometheusConfigMap(ctx, monitorStack); err != nil {
		return fmt.Errorf("failed to create Prometheus ConfigMap: %w", err)
	}

	// 如果配置了持久化存储，创建PVC
	if monitorStack.Spec.Prometheus.Storage.Size != "" {
		if err := r.createPrometheusPVC(ctx, monitorStack); err != nil {
			return fmt.Errorf("failed to create Prometheus PVC: %w", err)
		}
	}

	// 创建Prometheus Deployment
	if err := r.createPrometheusDeployment(ctx, monitorStack); err != nil {
		return fmt.Errorf("failed to create Prometheus Deployment: %w", err)
	}

	// 创建Prometheus Service
	if err := r.createPrometheusService(ctx, monitorStack); err != nil {
		return fmt.Errorf("failed to create Prometheus Service: %w", err)
	}

	// 检查Deployment状态并更新MonitorStack状态
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      r.getPrometheusName(monitorStack),
		Namespace: monitorStack.Namespace,
	}, deployment)
	if err != nil {
		return err
	}

	// 更新Prometheus组件状态
	monitorStack.Status.PrometheusStatus.Ready = deployment.Status.ReadyReplicas > 0
	monitorStack.Status.PrometheusStatus.Replicas = deployment.Status.Replicas
	if deployment.Status.ReadyReplicas > 0 {
		monitorStack.Status.PrometheusStatus.Message = "Ready"
		monitorStack.Status.PrometheusStatus.Endpoint = fmt.Sprintf("http://%s:%d",
			r.getPrometheusServiceName(monitorStack), monitorStack.Spec.Prometheus.Service.Port)
	} else {
		monitorStack.Status.PrometheusStatus.Message = "Not Ready"
	}

	return nil
}

// reconcileGrafana 协调Grafana相关资源
// 创建和管理Grafana的ConfigMap、Deployment和Service
func (r *MonitorStackReconciler) reconcileGrafana(ctx context.Context, monitorStack *monitoringv1.MonitorStack) error {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Grafana resources")

	// 如果配置了数据源，创建数据源ConfigMap
	if len(monitorStack.Spec.Grafana.Datasources) > 0 {
		if err := r.createGrafanaDatasourcesConfigMap(ctx, monitorStack); err != nil {
			return fmt.Errorf("failed to create Grafana datasources ConfigMap: %w", err)
		}
	}

	// 创建Grafana Deployment
	if err := r.createGrafanaDeployment(ctx, monitorStack); err != nil {
		return fmt.Errorf("failed to create Grafana Deployment: %w", err)
	}

	// 创建Grafana Service
	if err := r.createGrafanaService(ctx, monitorStack); err != nil {
		return fmt.Errorf("failed to create Grafana Service: %w", err)
	}

	// 检查Deployment状态并更新MonitorStack状态
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      r.getGrafanaName(monitorStack),
		Namespace: monitorStack.Namespace,
	}, deployment)
	if err != nil {
		return err
	}

	// 更新Grafana组件状态
	monitorStack.Status.GrafanaStatus.Ready = deployment.Status.ReadyReplicas > 0
	monitorStack.Status.GrafanaStatus.Replicas = deployment.Status.Replicas
	if deployment.Status.ReadyReplicas > 0 {
		monitorStack.Status.GrafanaStatus.Message = "Ready"
		monitorStack.Status.GrafanaStatus.Endpoint = fmt.Sprintf("http://%s:%d",
			r.getGrafanaServiceName(monitorStack), monitorStack.Spec.Grafana.Service.Port)
	} else {
		monitorStack.Status.GrafanaStatus.Message = "Not Ready"
	}

	return nil
}

// createPrometheusConfigMap 创建Prometheus配置ConfigMap
func (r *MonitorStackReconciler) createPrometheusConfigMap(ctx context.Context, monitorStack *monitoringv1.MonitorStack) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getPrometheusConfigMapName(monitorStack),
			Namespace: monitorStack.Namespace,
			Labels:    r.getLabels(monitorStack, "prometheus"),
		},
		Data: map[string]string{
			"prometheus.yml": r.getPrometheusConfig(monitorStack),
		},
	}

	// 设置OwnerReference，确保级联删除
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

	// 更新现有ConfigMap的数据
	existing.Data = configMap.Data
	return r.Update(ctx, existing)
}

// createPrometheusPVC 创建Prometheus持久化存储
func (r *MonitorStackReconciler) createPrometheusPVC(ctx context.Context, monitorStack *monitoringv1.MonitorStack) error {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getPrometheusPVCName(monitorStack),
			Namespace: monitorStack.Namespace,
			Labels:    r.getLabels(monitorStack, "prometheus"),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(monitorStack.Spec.Prometheus.Storage.Size),
				},
			},
		},
	}

	// 如果指定了StorageClass，设置它
	if monitorStack.Spec.Prometheus.Storage.StorageClass != "" {
		pvc.Spec.StorageClassName = &monitorStack.Spec.Prometheus.Storage.StorageClass
	}

	// 设置OwnerReference
	if err := controllerutil.SetControllerReference(monitorStack, pvc, r.Scheme); err != nil {
		return err
	}

	// 检查PVC是否已存在
	existing := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, pvc)
		}
		return err
	}

	// PVC已存在，不需要更新（PVC通常不允许修改）
	return nil
}

// createPrometheusDeployment 创建Prometheus Deployment
func (r *MonitorStackReconciler) createPrometheusDeployment(ctx context.Context, monitorStack *monitoringv1.MonitorStack) error {
	deployment := r.buildPrometheusDeployment(monitorStack)

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

// createPrometheusService 创建Prometheus Service
func (r *MonitorStackReconciler) createPrometheusService(ctx context.Context, monitorStack *monitoringv1.MonitorStack) error {
	service := r.buildPrometheusService(monitorStack)

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

// updateStatus 更新MonitorStack状态
func (r *MonitorStackReconciler) updateStatus(ctx context.Context, monitorStack *monitoringv1.MonitorStack, phase, message string) {
	monitorStack.Status.Phase = phase
	monitorStack.Status.Message = message
	monitorStack.Status.LastUpdated = metav1.Now()
	r.Status().Update(ctx, monitorStack)
}

// updateOverallStatus 更新整体状态
func (r *MonitorStackReconciler) updateOverallStatus(ctx context.Context, monitorStack *monitoringv1.MonitorStack) error {
	// 检查各组件状态
	prometheusReady := !monitorStack.Spec.Prometheus.Enabled || monitorStack.Status.PrometheusStatus.Ready
	grafanaReady := !monitorStack.Spec.Grafana.Enabled || monitorStack.Status.GrafanaStatus.Ready

	// 根据组件状态设置整体状态
	if prometheusReady && grafanaReady {
		monitorStack.Status.Phase = "Ready"
		monitorStack.Status.Message = "All enabled components are ready"
	} else {
		monitorStack.Status.Phase = "Pending"
		monitorStack.Status.Message = "Waiting for components to be ready"
	}

	monitorStack.Status.LastUpdated = metav1.Now()
	return r.Status().Update(ctx, monitorStack)
}

// cleanupPrometheusResources 清理Prometheus相关资源
func (r *MonitorStackReconciler) cleanupPrometheusResources(ctx context.Context, monitorStack *monitoringv1.MonitorStack) error {
	// 注意：由于设置了OwnerReference，当MonitorStack被删除时，
	// Kubernetes会自动删除相关的子资源，这里主要用于禁用组件时的清理

	// 删除Deployment
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      r.getPrometheusName(monitorStack),
		Namespace: monitorStack.Namespace,
	}, deployment)
	if err == nil {
		r.Delete(ctx, deployment)
	}

	// 删除Service
	service := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      r.getPrometheusServiceName(monitorStack),
		Namespace: monitorStack.Namespace,
	}, service)
	if err == nil {
		r.Delete(ctx, service)
	}

	// 删除ConfigMap
	configMap := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      r.getPrometheusConfigMapName(monitorStack),
		Namespace: monitorStack.Namespace,
	}, configMap)
	if err == nil {
		r.Delete(ctx, configMap)
	}

	return nil
}

// cleanupGrafanaResources 清理Grafana相关资源
func (r *MonitorStackReconciler) cleanupGrafanaResources(ctx context.Context, monitorStack *monitoringv1.MonitorStack) error {
	// 删除Deployment
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      r.getGrafanaName(monitorStack),
		Namespace: monitorStack.Namespace,
	}, deployment)
	if err == nil {
		r.Delete(ctx, deployment)
	}

	// 删除Service
	service := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      r.getGrafanaServiceName(monitorStack),
		Namespace: monitorStack.Namespace,
	}, service)
	if err == nil {
		r.Delete(ctx, service)
	}

	return nil
}

// SetupWithManager 设置控制器与Manager的关系
// 配置控制器监听的资源类型和拥有的资源类型
func (r *MonitorStackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1.MonitorStack{}).     // 监听MonitorStack资源
		Owns(&appsv1.Deployment{}).            // 拥有Deployment资源
		Owns(&corev1.Service{}).               // 拥有Service资源
		Owns(&corev1.ConfigMap{}).             // 拥有ConfigMap资源
		Owns(&corev1.PersistentVolumeClaim{}). // 拥有PVC资源
		Complete(r)
}
