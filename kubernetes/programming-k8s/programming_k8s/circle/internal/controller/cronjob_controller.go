/*
Copyright 2023 Jayson Wang.

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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	batchv1 "github.com/wjiec/programming_k8s/circle/api/v1"
)

// CronJobReconciler reconciles a CronJob object
type CronJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Clock
}

//+kubebuilder:rbac:groups=batch.example.org,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch.example.org,resources=cronjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch.example.org,resources=cronjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups=v1,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=v1,resources=pods/status,verbs=get

const (
	jobOwnerKey = ".metadata.controlled-by"
)

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *CronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var cronJob batchv1.CronJob
	if err := r.Get(ctx, req.NamespacedName, &cronJob); err != nil {
		logger.Error(err, "unable to fetch CronJob")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.V(1).Info("fetched cronjob", "cronjob", cronJob)

	var childPods corev1.PodList
	if err := r.List(ctx, &childPods, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
		logger.Error(err, "unable to list child Pods")
		return ctrl.Result{}, err
	}
	logger.V(1).Info("child of the cronjob", "pods", childPods)

	var activePods, failedPods, succeedPods []*corev1.Pod
	for idx, pod := range childPods.Items {
		switch pod.Status.Phase {
		case corev1.PodSucceeded:
			succeedPods = append(succeedPods, &childPods.Items[idx])
		case corev1.PodFailed:
			failedPods = append(failedPods, &childPods.Items[idx])
		default:
			activePods = append(activePods, &childPods.Items[idx])
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CronJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Clock == nil {
		r.Clock = realClock{}
	}

	err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, jobOwnerKey, func(object client.Object) []string {
		if pod, ok := object.(*corev1.Pod); ok {
			if ownerRef := metav1.GetControllerOf(pod); ownerRef != nil {
				if ownerRef.APIVersion == batchv1.GroupVersion.String() && ownerRef.Kind == "CronJob" {
					return []string{ownerRef.Name}
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.CronJob{}).
		Complete(r)
}

// Clock knows how to get the current time.
// It can be used to fake out timing for testing.
type Clock interface {
	// Now returns the current local time.
	Now() time.Time
}

type realClock struct{}

// Now returns the current local time.
func (realClock) Now() time.Time { return time.Now() }
