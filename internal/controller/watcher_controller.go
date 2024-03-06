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

	retry, err := r.deployWatcherWeb(ctx, watcher)
	if retry {
		return ctrl.Result{Requeue: true}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	retry, err := r.deployWorkers(ctx, watcher)
	if retry {
		return ctrl.Result{Requeue: true}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	retry, err := r.deployFirstRun(ctx, watcher)
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

func (r *WatcherReconciler) deployWatcherWeb(ctx context.Context, w *mediav1.Watcher) (bool, error) {

	return false, nil
}

func (r *WatcherReconciler) deployWorkers(ctx context.Context, w *mediav1.Watcher) (bool, error) {

	return false, nil
}

func (r *WatcherReconciler) deployFirstRun(ctx context.Context, w *mediav1.Watcher) (bool, error) {

	return false, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WatcherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mediav1.Watcher{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}
