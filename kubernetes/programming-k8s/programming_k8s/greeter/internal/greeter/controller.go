package greeter

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	clientset "github.com/wjiec/programming_k8s/greeter/pkg/clientset/versioned"
	greeterscheme "github.com/wjiec/programming_k8s/greeter/pkg/clientset/versioned/scheme"
	greeterinformers "github.com/wjiec/programming_k8s/greeter/pkg/informers/externalversions/greeter/v1alpha1"
	greeterlisters "github.com/wjiec/programming_k8s/greeter/pkg/listers/greeter/v1alpha1"
)

const (
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

	return nil
}

// enqueueGreeter takes a Greeter resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Greeter.
func (c *Controller) enqueueGreeter(object any) {
	if key, err := cache.MetaNamespaceIndexFunc(object); err != nil {
		utilruntime.HandleError(err)
	} else {
		c.workQueue.Add(key)
	}
}

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
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueGreeter(new)
		},
	})
	if err != nil {
		logger.Error(err, "Error setup event handler for greeter")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	return controller
}
