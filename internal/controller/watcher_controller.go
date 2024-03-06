/*
Copyright 2024.

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

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	// rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	// "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/client"
	// "sigs.k8s.io/controller-runtime/pkg/log"

	mediav1 "github.com/flussonic/watcher-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/lithammer/shortuuid/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// WatcherReconciler reconciles a Watcher object
type WatcherReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=media.flussonic.com,resources=watchers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=media.flussonic.com,resources=watchers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=media.flussonic.com,resources=watchers/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Watcher object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *WatcherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// _ = log.FromContext(ctx)
	log := r.Log.WithValues("Watcher", req.NamespacedName, "ReconcileId", shortuuid.New())

	// TODO(user): your logic here
	log.Info("Processing WatcherReconciler")

	watcher := &mediav1.Watcher{}
	err := r.Client.Get(ctx, req.NamespacedName, watcher)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Watcher resource is not found, ignoring further work")
			return ctrl.Result{}, nil
		}
		log.Error(err, "error getting Watcher")
		return ctrl.Result{}, err
	}

	retry, err := r.deployRedis(ctx, watcher)
	if retry {
		return ctrl.Result{Requeue: true}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	retry, err = r.deployWatcherWeb(ctx, watcher)
	if retry {
		return ctrl.Result{Requeue: true}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	retry, err = r.deployWorkers(ctx, watcher)
	if retry {
		return ctrl.Result{Requeue: true}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	retry, err = r.deployFirstRun(ctx, watcher)
	if retry {
		return ctrl.Result{Requeue: true}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *WatcherReconciler) deployRedis(ctx context.Context, w *mediav1.Watcher) (bool, error) {

	redisName := w.Name + "-redis"

	labels := map[string]string{"app": redisName}
	ss1 := &appsv1.StatefulSet{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: redisName, Namespace: w.Namespace}, ss1)
	if err != nil && errors.IsNotFound(err) {

		replicas := int32(1)
		spec := corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:            "redis",
				Image:           "redis",
				ImagePullPolicy: "IfNotPresent",
				Ports: []corev1.ContainerPort{{
					ContainerPort: 9017,
				}},
				Command: []string{"redis-server", "--port", "9017", "--save", ""},
			}},
		}

		ss := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      redisName,
				Namespace: w.Namespace,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: spec,
				},
			},
		}
		ctrl.SetControllerReference(w, ss, r.Scheme)
		err = r.Client.Create(ctx, ss)
		if err != nil {
			return false, err
		}
		return true, nil
	} else if err != nil {
		return false, err
	}

	s1 := &corev1.Service{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: redisName, Namespace: w.Namespace}, s1)
	if err != nil && errors.IsNotFound(err) {

		s := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      redisName,
				Namespace: w.Namespace,
			},
			Spec: corev1.ServiceSpec{
				Selector: labels,
				Ports: []corev1.ServicePort{{
					Name:       "redis",
					Port:       9017,
					TargetPort: intstr.FromInt(9017),
				}},
			},
		}
		ctrl.SetControllerReference(w, s, r.Scheme)
		err = r.Client.Create(ctx, s)
		if err != nil {
			return false, err
		}
		return true, nil
	} else if err != nil {
		return false, err
	}

	return false, nil
}

func commonEnv(w *mediav1.Watcher) []corev1.EnvVar {
	env := []corev1.EnvVar{}

	env = append(env, corev1.EnvVar{
		Name:  "REDIS",
		Value: "redis://" + w.Name + "-redis." + w.Namespace + ".svc.cluster.local:9017",
	})

	env = append(env, corev1.EnvVar{
		Name:  "CENTRAL_URL",
		Value: w.Spec.Central,
	})

	env = append(env, corev1.EnvVar{
		Name:  "DB",
		Value: w.Spec.Database,
	})

	return env
}

func webEnv(w *mediav1.Watcher) []corev1.EnvVar {
	env := commonEnv(w)

	env = append(env, corev1.EnvVar{
		Name:  "PORT",
		Value: "9015",
	})

	env = append(env, corev1.EnvVar{
		Name:  "LISTEN_HOST",
		Value: "0.0.0.0",
	})

	env = append(env, corev1.EnvVar{
		Name:  "NODBCHECK",
		Value: "true",
	})

	env = append(env, w.Spec.PodEnvVariables...)
	return env
}

func (r *WatcherReconciler) deployWatcherWeb(ctx context.Context, w *mediav1.Watcher) (bool, error) {
	serviceName := w.Name + "-web"
	webName := w.Name + "-web"

	labels := map[string]string{"app": webName}

	svc1 := &corev1.Service{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: w.Namespace}, svc1)
	if err != nil && errors.IsNotFound(err) {

		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: w.Namespace,
			},
			Spec: corev1.ServiceSpec{
				Selector: labels,
				Type:     "ClusterIP",
				Ports: []corev1.ServicePort{{
					Name:       "watcher",
					Port:       80,
					TargetPort: intstr.FromInt(9015),
				}},
			},
		}
		ctrl.SetControllerReference(w, svc, r.Scheme)
		err = r.Client.Create(ctx, svc)
		if err != nil {
			return false, err
		}
		return true, nil
	} else if err != nil {
		return false, err
	}

	env := webEnv(w)
	webWorkers := w.Spec.WebWorkers
	if webWorkers == 0 {
		webWorkers = 2
	}
	replicas := int32(webWorkers)

	deploy1 := &appsv1.Deployment{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: w.Namespace}, deploy1)
	if err != nil && errors.IsNotFound(err) {

		spec := corev1.PodSpec{
			NodeSelector: w.Spec.WebNodeSelector,
			Containers: []corev1.Container{{
				Name:            "watcher",
				Image:           w.Spec.Image,
				ImagePullPolicy: "IfNotPresent",
				Env:             env,
			}},
		}

		deploy := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      webName,
				Namespace: w.Namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Replicas: &replicas,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
					Spec: spec,
				},
			},
		}
		ctrl.SetControllerReference(w, deploy, r.Scheme)
		err = r.Client.Create(ctx, deploy)
		if err != nil {
			return false, err
		}
		return true, nil
	} else if err != nil {
		return false, err
	}

	deploy1.Spec.Template.Spec.NodeSelector = w.Spec.WebNodeSelector
	deploy1.Spec.Template.Spec.Containers[0].Image = w.Spec.Image
	deploy1.Spec.Template.Spec.Containers[0].Env = env
	deploy1.Spec.Replicas = &replicas
	err = r.Client.Update(ctx, deploy1)
	if err != nil {
		return false, err
	}

	return false, nil
}

func (r *WatcherReconciler) deployWorkers(ctx context.Context, w *mediav1.Watcher) (bool, error) {

	return false, nil
}

func (r *WatcherReconciler) deployFirstRun(ctx context.Context, w *mediav1.Watcher) (bool, error) {

	env := commonEnv(w)
	env = append(env, w.Spec.PodEnvVariables...)

	spec := corev1.PodSpec{
		RestartPolicy: "OnFailure",
		Containers: []corev1.Container{{
			Name:            "firstrun",
			Image:           w.Spec.Image,
			ImagePullPolicy: "IfNotPresent",
			Env:             env,
			Command: []string{
				"/bin/sh",
				"-c",
				`/opt/flussonic/bin/python3 -m manage check &&
		          /opt/flussonic/bin/python3 -m manage ensure_api_key &&
		          /opt/flussonic/bin/watcher-firstrun.sh`,
			},
		}},
	}

	jobName := w.Name + "-firstrun"
	j1 := &batchv1.Job{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: jobName, Namespace: w.Namespace}, j1)
	if err != nil && errors.IsNotFound(err) {

		parallelism := int32(1)
		timeout := int64(180)
		j := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jobName,
				Namespace: w.Namespace,
			},
			Spec: batchv1.JobSpec{
				ActiveDeadlineSeconds: &timeout,
				Parallelism:           &parallelism,
				Template: corev1.PodTemplateSpec{
					Spec: spec,
				},
			},
		}
		ctrl.SetControllerReference(w, j, r.Scheme)
		err = r.Client.Create(ctx, j)
		if err != nil {
			return false, err
		}
		return true, nil
	} else if err != nil {
		return false, err
	}

	return false, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WatcherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mediav1.Watcher{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
