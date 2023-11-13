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
	"errors"
	"sort"
	"time"

	"github.com/robfig/cron"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/reference"
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

	// Stage 2: List all active jobs, and update the status

	var childPods corev1.PodList
	if err := r.List(ctx, &childPods, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
		logger.Error(err, "unable to list child Pods")
		return ctrl.Result{}, err
	}
	logger.V(1).Info("child of the cronjob", "pods", childPods)

	var lastScheduledTime = cronJob.CreationTimestamp.Time
	var activePods, failedPods, successfulPods []*corev1.Pod
	for idx, pod := range childPods.Items {
		switch pod.Status.Phase {
		case corev1.PodSucceeded:
			successfulPods = append(successfulPods, &childPods.Items[idx])
		case corev1.PodFailed:
			failedPods = append(failedPods, &childPods.Items[idx])
		default:
			activePods = append(activePods, &childPods.Items[idx])
		}

		podLastScheduledTime, err := getScheduleTimeForPod(&pod)
		if err != nil {
			logger.Error(err, "unable to parse schedule time for child pod", "pod", &pod)
			continue
		}

		if podLastScheduledTime.After(lastScheduledTime) {
			lastScheduledTime = podLastScheduledTime
		}
	}
	logger.V(1).Info("job count", "active", len(activePods), "failed", len(failedPods), "successful", len(successfulPods))

	cronJob.Status.Active = nil
	cronJob.Status.LastScheduleTime = &metav1.Time{Time: lastScheduledTime}
	for _, pod := range activePods {
		podRef, err := reference.GetReference(r.Scheme, pod)
		if err != nil {
			logger.Error(err, "unable to make reference to active job", "pod", pod)
			continue
		}
		cronJob.Status.Active = append(cronJob.Status.Active, *podRef)
	}

	if err := r.Status().Update(ctx, &cronJob); err != nil {
		logger.Error(err, "unable to update CronJob status")
		return ctrl.Result{}, err
	}

	// Stage 3: Clean up old jobs according to the history limit

	// NB: deleting these are "best effort" -- if we fail on a particular one,
	// we won't requeue just to finish the deleting.
	if cronJob.Spec.FailedJobsHistoryLimit != nil {
		sort.Slice(failedPods, func(i, j int) bool {
			if failedPods[i].Status.StartTime == nil {
				return failedPods[j].Status.StartTime != nil
			}
			return failedPods[i].Status.StartTime.Before(failedPods[j].Status.StartTime)
		})

		for i := 0; i <= len(failedPods)-int(*cronJob.Spec.FailedJobsHistoryLimit); i++ {
			if err := r.Delete(ctx, failedPods[i], client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				logger.Error(err, "unable to delete old failed pod", "pod", failedPods[i])
			} else {
				logger.V(0).Info("deleted old failed pod", "pod", failedPods[i])
			}
		}
	}
	if cronJob.Spec.SuccessfulJobsHistoryLimit != nil {
		sort.Slice(successfulPods, func(i, j int) bool {
			if successfulPods[i].Status.StartTime == nil {
				return successfulPods[j].Status.StartTime != nil
			}
			return successfulPods[i].Status.StartTime.Before(successfulPods[j].Status.StartTime)
		})

		for i := 0; i <= len(successfulPods)-int(*cronJob.Spec.SuccessfulJobsHistoryLimit); i++ {
			if err := r.Delete(ctx, successfulPods[i], client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				logger.Error(err, "unable to delete old failed pod", "pod", successfulPods[i])
			} else {
				logger.V(0).Info("deleted old failed pod", "pod", successfulPods[i])
			}
		}
	}

	// Stage 4: Check if we’re suspended

	if cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend {
		logger.V(1).Info("cronjob suspended, skipping")
		return ctrl.Result{}, nil
	}

	// Stage 5: Get the next scheduled run

	// figure out the next times that we need to create jobs at (or anything we missed).
	missedRun, nextRun, err := getNextScheduledTime(&cronJob, r.Now())
	if err != nil {
		logger.Error(err, "unable to figure out CronJob schedule")
		// we don't really care about requeuing until we get an update that
		// fixes the schedule, so don't return an error
		return ctrl.Result{}, nil
	}

	// Stage 6: Run a new job if it’s on schedule, not past the deadline, and not blocked by our concurrency policy
	waitingNextScheduleResult := ctrl.Result{RequeueAfter: nextRun.Sub(r.Now())}
	if missedRun.IsZero() {
		logger.V(1).Info("no upcoming scheduled times, sleeping until next")
		return waitingNextScheduleResult, nil
	}

	// If we’ve missed a run, and we’re still within the deadline to start it, we’ll need to run a job.
	if cronJob.Spec.StartingDeadlineSeconds != nil {
		// make sure we're not too late to start the run
		schedulingDeadline := missedRun.Add(time.Second * time.Duration(*cronJob.Spec.StartingDeadlineSeconds))
		if schedulingDeadline.Before(r.Now()) {
			logger.V(1).Info("missed starting deadline for last run, sleeping till next")
			return waitingNextScheduleResult, nil
		}
	}

	// now, we actually have to run a job, we’ll need to either wait till existing
	// ones finish, replace the existing ones, or just add new ones.
	if cronJob.Spec.ConcurrencyPolicy == batchv1.ForbidConcurrent && len(activePods) > 0 {
		logger.V(1).Info("concurrency policy blocks concurrent runs, skipping", "num active", len(activePods))
		return waitingNextScheduleResult, nil
	}
	if cronJob.Spec.ConcurrencyPolicy == batchv1.ReplaceConcurrent {
		for _, activePod := range activePods {
			// we don't care if the job was already deleted
			if err = r.Delete(ctx, activePod, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				logger.Error(err, "unable to delete active pod", "pod", activePod)
				return ctrl.Result{}, err
			}
		}
	}

	// we’ll actually create our desired job
	pod, err := r.newPodForCronJob(&cronJob, missedRun)
	if err != nil {
		logger.Error(err, "unable to construct job from template")
		// don't requeue until we get a change to the spec
		return waitingNextScheduleResult, nil
	}
	if err = r.Create(ctx, pod); err != nil {
		logger.Error(err, "unable to create Pod for CronJob", "pod", pod)
		return ctrl.Result{}, err
	}

	// Stage 7: Requeue when we either see a running pod or it’s time for the next scheduled run

	logger.V(1).Info("created Pod for CronJob run", "pod", pod)
	// we'll requeue once we see the running pod, and update our status
	return waitingNextScheduleResult, nil
}

// newPodForCronJob construct a pod based on our CronJob’s template.
// We’ll copy over the spec from the template and copy some basic object meta.
func (r *CronJobReconciler) newPodForCronJob(cronJob *batchv1.CronJob, scheduledTime time.Time) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels:       map[string]string{},
			Annotations:  map[string]string{},
			GenerateName: cronJob.Name + "-",
			Namespace:    cronJob.Namespace,
		},
		Spec: *cronJob.Spec.JobTemplate.Spec.DeepCopy(),
	}

	for k, v := range cronJob.Spec.JobTemplate.Labels {
		pod.Labels[k] = v
	}
	for k, v := range cronJob.Spec.JobTemplate.Annotations {
		pod.Annotations[k] = v
	}
	pod.Annotations[scheduledTimeAnnotation] = scheduledTime.Format(time.RFC3339)

	if err := ctrl.SetControllerReference(cronJob, pod, r.Scheme); err != nil {
		return nil, err
	}

	return pod, nil
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

var (
	scheduledTimeAnnotation = "batch.example.org/scheduled-at"

	ErrScheduleTimeNotFound = errors.New("scheduled time not found in the pod")
)

// getScheduleTimeForPod extract the scheduled time from the annotation
// that we added during job creation.
func getScheduleTimeForPod(pod *corev1.Pod) (time.Time, error) {
	scheduledTime := pod.Annotations[scheduledTimeAnnotation]
	if len(scheduledTime) != 0 {
		return time.Parse(time.RFC3339, scheduledTime)
	}
	return time.Time{}, ErrScheduleTimeNotFound
}

// getNextScheduledTime calculate what time we should execute the new jobs based on
// the earliest time, as well as calculate the next run time after the current time.
func getNextScheduledTime(cronJob *batchv1.CronJob, now time.Time) (time.Time, time.Time, error) {
	schedule, err := cron.ParseStandard(cronJob.Spec.Schedule)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	// we’ll start calculating appropriate times from our last run, or
	// the creation of the CronJob if we can’t find a last run.
	var earliestTime time.Time
	if cronJob.Status.LastScheduleTime != nil {
		earliestTime = cronJob.Status.LastScheduleTime.Time
	} else {
		earliestTime = cronJob.CreationTimestamp.Time
	}

	if cronJob.Spec.StartingDeadlineSeconds != nil {
		// controller is not going to schedule anything below this point
		schedulingDeadline := now.Add(-time.Second * time.Duration(*cronJob.Spec.StartingDeadlineSeconds))
		if schedulingDeadline.After(earliestTime) {
			earliestTime = schedulingDeadline
		}
	}

	// There are currently no jobs to execute
	if earliestTime.After(now) {
		return time.Time{}, schedule.Next(now), nil
	}

	lastMissed, missingJobs := time.Time{}, 0
	for t := schedule.Next(earliestTime); !t.After(now); t = schedule.Next(t) {
		lastMissed = t
		// An object might miss several starts. For example, if
		// controller gets wedged on Friday at 5:01pm when everyone has
		// gone home, and someone comes in on Tuesday AM and discovers
		// the problem and restarts the controller, then all the hourly
		// jobs, more than 80 of them for one hourly scheduledJob, should
		// all start running with no further intervention (if the scheduledJob
		// allows concurrency and late starts).
		//
		// However, if there is a bug somewhere, or incorrect clock
		// on controller's server or apiservers (for setting creationTimestamp)
		// then there could be so many missed start times (it could be off
		// by decades or more), that it would eat up all the CPU and memory
		// of this controller. In that case, we want to not try to list
		// all the missed start times.
		if missingJobs++; missingJobs > 100 {
			// We can't get the most recent times so just return an empty slice
			return time.Time{}, time.Time{}, errors.New("too many missed jobs")
		}
	}

	return lastMissed, schedule.Next(now), nil
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
