package greeter

import (
	"context"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"github.com/wjiec/programming_k8s/greeter/pkg/apis/greeter/v1alpha1"
	clientset "github.com/wjiec/programming_k8s/greeter/pkg/generated/clientset/versioned"
	greeterscheme "github.com/wjiec/programming_k8s/greeter/pkg/generated/clientset/versioned/scheme"
	greeterinformers "github.com/wjiec/programming_k8s/greeter/pkg/generated/informers/externalversions/greeter/v1alpha1"
	greeterlisters "github.com/wjiec/programming_k8s/greeter/pkg/generated/listers/greeter/v1alpha1"
)

const (
	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Pod already existing
	MessageResourceExists = "Resource %q already exists and is not managed by Greeter"
	// MessageResourceSynced is the message used for an Event fired when a Greeter
	// is synced successfully
	MessageResourceSynced = "Greeter synced successfully"

	// ErrResourceExists is used as part of the Event 'reason' when a Greeter fails
	// to sync due to a Pod of the same name already existing.
	ErrResourceExists = "ErrResourceExists"
	// SuccessSynced is used as part of the Event 'reason' when a Greeter is synced
	SuccessSynced = "Synced"

	controllerAgentName = "greeter-controller"
)

type Controller struct {
	// kubeClientset is a standard kubernetes clientset
	kubeClientset kubernetes.Interface
	// greeterClientset is a clientset for our own API group
	greeterClientset clientset.Interface

	podLister corelister.PodLister
	podSynced cache.InformerSynced

	greeterLister greeterlisters.GreeterLister
	greeterSynced cache.InformerSynced

	// workQueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workQueue workqueue.RateLimitingInterface

	// recorder is an event recorder for recording Event resource to the
	// kubernetes API.
	recorder record.EventRecorder
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shut down the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(ctx context.Context, workers int) error {
	defer utilruntime.HandleCrash()
	defer c.workQueue.ShutDown()

	logger := klog.FromContext(ctx)

	// Start the informer factories to begin populating the informer caches
	logger.Info("Starting Greeter controller")

	// Wait for the caches to be synced before starting workers
	logger.Info("Waiting for informer caches to sync")

	if ok := cache.WaitForCacheSync(ctx.Done(), c.podSynced, c.greeterSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	logger.Info("Starting workers", "count", workers)
	// Launch workers to process Greeter resources
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	logger.Info("Started workers")
	<-ctx.Done()
	logger.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	object, shutdown := c.workQueue.Get()
	if shutdown {
		return false
	}

	logger := klog.FromContext(ctx)
	// We wrap this block in a func, so we can defer c.workQueue.Done.
	err := func(object any) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workQueue.Done(object)

		key, ok := "", false
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = object.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workQueue.Forget(object)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", object))
			return nil
		}

		// Run the syncHandler, passing it the namespace/name string of the
		// Greeter resource to be synced.
		if when, err := c.syncHandler(ctx, key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workQueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		} else if when != 0 {
			// Put the item back on the workqueue after the indicated duration.
			c.workQueue.AddAfter(key, when)
			return nil
		}

		// Finally, if no error occurs we Forget this item, so it does not
		// get queued again until another change happens.
		c.workQueue.Forget(object)
		logger.Info("Successfully synced", "resourceName", key)
		return nil
	}(object)

	if err != nil {
		utilruntime.HandleError(err)
	}

	return true
}

// syncHandler try to fetch this resource from the cluster and start
// comparing the state of the resource to the expected state.
func (c *Controller) syncHandler(ctx context.Context, key string) (time.Duration, error) {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return 0, nil
	}

	// Get the Greeter resource with this namespace/name
	greeter, err := c.greeterLister.Greeters(namespace).Get(name)
	if err != nil {
		// The Greeter resource may no longer exist, in which case we stop processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("greeter '%s' in work queue no longer exists", key))
			return 0, nil
		}
		return 0, err
	}

	return c.syncGreeter(ctx, greeter)
}

// syncGreeter compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Greeter resource
// with the current status of the resource.
//
//goland:noinspection GoDeprecation
func (c *Controller) syncGreeter(ctx context.Context, greeter *v1alpha1.Greeter) (time.Duration, error) {
	key, err := cache.MetaNamespaceKeyFunc(greeter)
	if err != nil {
		return 0, err
	}
	logger := klog.LoggerWithValues(klog.FromContext(ctx), "resourceName", key)

	// NEVER modify objects from the store. It's a read-only, owned by the lister.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	instance := greeter.DeepCopy()
	if instance.Status.Phase == "" {
		instance.Status.Phase = v1alpha1.PhasePending
	}

	// If no phase set, default to pending (the initial phase)
	switch instance.Status.Phase {
	case v1alpha1.PhasePending:
		logger.Info("pending greeter", "schedule", instance.Spec.Schedule)

		// Check if it's already time to execute
		when, err := timeUntilSchedule(instance.Spec.Schedule)
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("schedule parsing failed: %v", err))
			// Error reading the schedule - requeue the request:
			return 0, err
		}

		// Not yet time to execute, wait until the scheduled time
		if when > 0 {
			return when, nil
		}

		klog.Infof("it's time! ready to greet: %s", instance.Spec.Message)
		instance.Status.Phase = v1alpha1.PhaseRunning
	case v1alpha1.PhaseRunning:
		logger.Info("running greeter", "message", instance.Spec.Message)
		podForGreeter := newPodForGreeter(instance)

		// Set Greeter instance as the owner and controller
		owner := metav1.NewControllerRef(instance, v1alpha1.SchemeGroupVersion.WithKind("Greeter"))
		podForGreeter.OwnerReferences = append(podForGreeter.OwnerReferences, *owner)

		// Try to see if the pod already exists and if not
		// (which we expect) then create a one-shot pod as per spec
		found, err := c.podLister.Pods(podForGreeter.Namespace).Get(podForGreeter.Name)
		if err != nil && errors.IsNotFound(err) {
			found, err = c.kubeClientset.CoreV1().Pods(podForGreeter.Namespace).Create(ctx, podForGreeter, metav1.CreateOptions{})
			if err != nil {
				return 0, err
			}
			logger.Info("pod launched", "podName", podForGreeter.Name)
		} else if err != nil {
			// requeue with error
			return 0, err
		}

		// If the Pod is not controlled by this Greeter resource, we should log
		// a warning to the event recorder and return error msg.
		if !metav1.IsControlledBy(found, instance) {
			msg := fmt.Sprintf(MessageResourceExists, podForGreeter.Name)
			c.recorder.Event(instance, corev1.EventTypeWarning, ErrResourceExists, msg)
			return 0, fmt.Errorf("%s", msg)
		}

		if found.Status.Phase == corev1.PodFailed || found.Status.Phase == corev1.PodSucceeded {
			logger.Info("container terminated", "reason", found.Status.Phase)
			instance.Status.Phase = string(found.Status.Phase)
		}
	case v1alpha1.PhaseSucceeded:
		logger.Info("greeter succeeded")
	case v1alpha1.PhaseFailed:
		logger.Info("greeter failed")
	}

	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Greeter resource.
	if !reflect.DeepEqual(greeter, instance) {
		// UpdateStatus will not allow changes to the Spec of the resource,
		// which is ideal for ensuring nothing other than resource status has been updated.
		_, err := c.greeterClientset.GreeterV1alpha1().Greeters(instance.Namespace).UpdateStatus(ctx, instance, metav1.UpdateOptions{})
		if err != nil {
			return 0, err
		}

		c.recorder.Event(instance, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	}

	// Don't requeue. We should be reconciled because either the pod or the CR changes.
	return 0, nil
}

// enqueueGreeter takes a Greeter resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Greeter.
func (c *Controller) enqueueGreeter(object any) {
	if key, err := cache.MetaNamespaceKeyFunc(object); err != nil {
		utilruntime.HandleError(err)
	} else {
		c.workQueue.Add(key)
	}
}

// enqueuePod enqueue a pod and checks that the owner reference points to a Greeter object. It then
// enqueues this Greeter object.
func (c *Controller) enqueuePod(object any) {
	pod, ok := (*corev1.Pod)(nil), false
	if pod, ok = object.(*corev1.Pod); !ok {
		tombstone, ok := object.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding pod, invalid type"))
			return
		}

		pod, ok = tombstone.Obj.(*corev1.Pod)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding pod tombstone, invalid type"))
			return
		}
	}

	if ownerRef := metav1.GetControllerOf(pod); ownerRef != nil {
		if ownerRef.Kind != "Greeter" {
			return
		}

		greeter, err := c.greeterLister.Greeters(pod.Namespace).Get(ownerRef.Name)
		if err != nil {
			return
		}

		if metav1.IsControlledBy(pod, greeter) {
			c.enqueueGreeter(greeter)
		}
	}
}

// timeUntilSchedule parses the schedule string and returns the time until the schedule.
// When it is overdue, the duration is negative.
func timeUntilSchedule(schedule string) (time.Duration, error) {
	expected, err := time.ParseInLocation("2006-01-02 15:04:05", schedule, time.Local)
	if err != nil {
		return 0, err
	}
	return expected.Sub(time.Now().UTC()), nil
}

// newPodForGreeter returns a busybox pod with the same name/namespace as the greeter
func newPodForGreeter(greeter *v1alpha1.Greeter) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      greeter.Name + "-runner",
			Namespace: greeter.Namespace,
			Labels: map[string]string{
				"app.k8s.io/name": greeter.Name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Image:   "busybox",
					Name:    "greeter",
					Command: []string{"sh"},
					Args:    []string{"-c", "echo " + greeter.Spec.Message},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
}

// NewController returns a new greeter controller
func NewController(
	ctx context.Context,
	kubeClientset kubernetes.Interface,
	greeterClientset clientset.Interface,
	podInformer coreinformers.PodInformer,
	greeterInformer greeterinformers.GreeterInformer,
) *Controller {
	logger := klog.FromContext(ctx)

	// Create event broadcaster
	// Add greeter-controller types to the default Kubernetes Scheme so Events can be
	// logged for greeter-controller types.
	utilruntime.Must(greeterscheme.AddToScheme(scheme.Scheme))
	logger.V(4).Info("Creating event broadcaster")

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	rateLimiter := workqueue.DefaultControllerRateLimiter()
	controller := &Controller{
		kubeClientset:    kubeClientset,
		greeterClientset: greeterClientset,
		podLister:        podInformer.Lister(),
		podSynced:        podInformer.Informer().HasSynced,
		greeterLister:    greeterInformer.Lister(),
		greeterSynced:    greeterInformer.Informer().HasSynced,
		workQueue: workqueue.NewRateLimitingQueueWithConfig(rateLimiter, workqueue.RateLimitingQueueConfig{
			Name: "Greeters",
		}),
		recorder: recorder,
	}

	logger.Info("Setting up event handlers")
	// Set up an event handler for when Greeter resources change
	_, err := greeterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueGreeter,
		UpdateFunc: func(oldObj, newObj interface{}) {
			controller.enqueueGreeter(newObj)
		},
	})
	if err != nil {
		logger.Error(err, "Error setup event handler for Greeter")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	// Set up an event handler for when Pod resources change
	_, err = podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueuePod,
		UpdateFunc: func(oldObj, newObj interface{}) {
			controller.enqueuePod(newObj)
		},
	})
	if err != nil {
		logger.Error(err, "Error setup event handler for Pod")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	return controller
}
