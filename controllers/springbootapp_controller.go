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

package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appsv1 "k8s.io/api/apps/v1"

	appv1 "github.com/cy18cn/spring-boot-operator/api/v1"
)

// SpringBootAppReconciler reconciles a SpringBootApp object
type SpringBootAppReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const (
	DefaultConfigVolumeName = "app-config"

	DefaultMinReplicas int32 = 2
	DefaultMaxReplicas int32 = 6
)

// +kubebuilder:rbac:groups=app.k8s.airparking.cn,resources=springbootapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.k8s.airparking.cn,resources=springbootapps/status,verbs=get;update;patch

func (r *SpringBootAppReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("springbootapp", req.NamespacedName)

	instance := &appv1.SpringBootApp{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "unable to found CronJob")
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error request get instance, requeue the request
		return ctrl.Result{}, err
	}

	labels := map[string]string{
		"app":        instance.Name,
		"version":    instance.Spec.Version,
		"deployment": fmt.Sprintf("%s-deployment", instance.Name),
	}

	// Define the desired Deployment object
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", instance.Name, instance.Spec.Version),
			Namespace: instance.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: imagePullSecrets(instance.Spec.ImagePullSecrets),
					Containers: []corev1.Container{
						springBootContainer(instance),
					},
					Affinity: podAffinity(instance.Spec.PodAffinity, instance.Spec.PodAntiAffinity),
					Volumes: []corev1.Volume{
						{
							Name: DefaultConfigVolumeName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: instance.Spec.ConfigMap,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(instance, deployment, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// check if deployment is existed
	foundedDep := &appsv1.Deployment{}
	if err := r.Get(
		ctx,
		types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace},
		foundedDep,
	); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating Deployment", "namespace", deployment.Namespace, "name", deployment.Name)
			err = r.Create(ctx, deployment)
		}
		return ctrl.Result{}, err
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       8080,
					TargetPort: intstr.IntOrString{IntVal: 8080},
				},
			},
			Selector: map[string]string{
				"app": instance.Name,
			},
		},
	}

	foundedSvc := &corev1.Service{}
	if err := r.Get(
		ctx,
		types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace},
		foundedSvc,
	); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating Service", "namespace", deployment.Namespace, "name", deployment.Name)
			err = r.Create(ctx, svc)
		}
		return ctrl.Result{}, err
	}

	// hpa
	hpa := &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       instance.Name,
				APIVersion: "apps/v1",
			},
		},
	}

	if instance.Spec.MinReplicas != nil {
		hpa.Spec.MinReplicas = instance.Spec.MinReplicas
	} else {
		minReplicas := DefaultMinReplicas
		hpa.Spec.MinReplicas = &minReplicas
	}
	if instance.Spec.MaxReplicas != nil {
		hpa.Spec.MaxReplicas = *instance.Spec.MaxReplicas
	} else {
		hpa.Spec.MaxReplicas = DefaultMaxReplicas
	}

	foundedHPA := &autoscalingv1.HorizontalPodAutoscaler{}
	if err := r.Get(
		ctx,
		types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace},
		foundedHPA,
	); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating HorizontalPodAutoscaler", "namespace", deployment.Namespace, "name", deployment.Name)
			err = r.Create(ctx, hpa)
		}
		return ctrl.Result{}, err
	}

	// update the found object and write result back if it is changed
	if !reflect.DeepEqual(deployment, foundedDep) {
		foundedDep.Spec = deployment.Spec
		log.Info("Updating deployment", "namespace", foundedDep.Namespace, "name", foundedDep.Name)
		err := r.Update(ctx, foundedDep)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if !reflect.DeepEqual(svc, foundedSvc) {
		foundedSvc.Spec = svc.Spec
		log.Info("Updating Service", "namespace", foundedDep.Namespace, "name", foundedDep.Name)
		err := r.Update(ctx, foundedSvc)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if !reflect.DeepEqual(hpa, foundedHPA) {
		foundedHPA.Spec = hpa.Spec
		log.Info("Updating Service", "namespace", foundedDep.Namespace, "name", foundedDep.Name)
		err := r.Update(ctx, foundedHPA)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *SpringBootAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.SpringBootApp{}).
		Complete(r)
}

func springBootContainer(springBootApp *appv1.SpringBootApp) corev1.Container {
	container := corev1.Container{
		Name:  springBootApp.Name,
		Image: fmt.Sprintf("%s:%s", springBootApp.Spec.ImageRepo, springBootApp.Spec.AppImage),
		Ports: springBootApp.Spec.Ports,
		Env:   springBootApp.Spec.Env,
	}

	if len(springBootApp.Spec.LivenessProbePath) > 0 {
		container.LivenessProbe = buildProbe(springBootApp.Spec.LivenessProbePath, 300, 30)
	}
	if len(springBootApp.Spec.ReadinessProbePath) > 0 {
		container.ReadinessProbe = buildProbe(springBootApp.Spec.ReadinessProbePath, 30, 5)
	}
	if len(springBootApp.Spec.ConfigMap) > 0 {
		container.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      DefaultConfigVolumeName,
				ReadOnly:  true,
				MountPath: "/usr/local/app/config/application.yml",
				SubPath:   "application.yml",
			},
		}
	}
	if springBootApp.Spec.Resources != nil {
		container.Resources = *springBootApp.Spec.Resources
	} else {
		container.Resources = corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    *resource.NewQuantity(1000, resource.DecimalSI),
				corev1.ResourceMemory: *resource.NewQuantity(1024, resource.BinarySI),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    *resource.NewQuantity(100, resource.DecimalSI),
				corev1.ResourceMemory: *resource.NewQuantity(512, resource.BinarySI),
			},
		}
	}
	if springBootApp.Spec.ImagePullPolicy != nil {
		container.ImagePullPolicy = *springBootApp.Spec.ImagePullPolicy
	} else {
		container.ImagePullPolicy = corev1.PullIfNotPresent
	}
	return container
}

const (
	DefaultFailureThreshold = 5
	DefaultSuccessThreshold = 1
	DefaultTimeoutSeconds   = 5
)

func buildProbe(path string, delay, timeout int32) *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   path,
				Port:   intstr.IntOrString{IntVal: 8080},
				Scheme: corev1.URISchemeHTTP,
			},
		},
		FailureThreshold:    DefaultFailureThreshold,
		InitialDelaySeconds: delay,
		SuccessThreshold:    DefaultSuccessThreshold,
		PeriodSeconds:       timeout,
		TimeoutSeconds:      DefaultTimeoutSeconds,
	}
}

func podAffinity(podAffinity *corev1.PodAffinity, podAntiAffinity *corev1.PodAntiAffinity) *corev1.Affinity {
	if podAffinity == nil && podAntiAffinity == nil {
		return nil
	}

	affinity := &corev1.Affinity{}
	if podAffinity != nil {
		affinity.PodAffinity = podAffinity
	}
	if podAntiAffinity != nil {
		affinity.PodAntiAffinity = podAntiAffinity
	}
	return affinity
}

func imagePullSecrets(imagePullSecrets string) []corev1.LocalObjectReference {
	if len(imagePullSecrets) == 0 {
		return nil
	}
	return []corev1.LocalObjectReference{
		{Name: imagePullSecrets},
	}
}
