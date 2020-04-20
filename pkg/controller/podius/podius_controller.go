package podius

import (
	"context"
	"os"

	codiusv1 "github.com/wilsonianb/codius-operator/pkg/apis/codius/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_podius")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Podius Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePodius{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("podius-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Podius
	err = c.Watch(&source.Kind{Type: &codiusv1.Podius{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Podius
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codiusv1.Podius{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &codiusv1.Podius{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcilePodius implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcilePodius{}

// ReconcilePodius reconciles a Podius object
type ReconcilePodius struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Podius object and makes changes based on the state read
// and what is in the Podius.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePodius) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Podius")

	// Fetch the Podius instance
	instance := &codiusv1.Podius{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Check if the deployment already exists, if not create a new one
	deployment := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		dep := deploymentForCR(instance)
		// Set Podius instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, dep, r.scheme); err != nil {
			return reconcile.Result{}, err
		}
		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return reconcile.Result{}, err
		}

		// Deployment created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Deployment")
		return reconcile.Result{}, err
	}

	// Check if the Service already exists, if not create a new one
	service := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, service)
	if err != nil && errors.IsNotFound(err) {
		ser := serviceForCR(instance)
		// Set Podius instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, ser, r.scheme); err != nil {
			return reconcile.Result{}, err
		}
		reqLogger.Info("Creating a new Service.", "Service.Namespace", ser.Namespace, "Service.Name", ser.Name)
		err = r.client.Create(context.TODO(), ser)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Service.", "Service.Namespace", ser.Namespace, "Service.Name", ser.Name)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Service.")
		return reconcile.Result{}, err
	}

	// Deployment and Service already exists - don't requeue
	reqLogger.Info("Skip reconcile: Deployment and Service already exist", "Namespace", deployment.Namespace, "Deployment.Name", deployment.Name, "Service.Name", service.Name)
	return reconcile.Result{}, nil
}

// deploymentForCR returns a podius Deployment object
func deploymentForCR(cr *codiusv1.Podius) *appsv1.Deployment {
	labels := labelsForCR(cr)
	containers := make([]corev1.Container, len(cr.Spec.Containers))
	for i, container := range cr.Spec.Containers {
		envVars := make([]corev1.EnvVar, len(container.Env))
		for j, env := range container.Env {
			envVars[j] = corev1.EnvVar{
				Name: env.Name,
				Value: env.Value,
			}
		}
		containers[i] = corev1.Container{
			Name: container.Name,
			Image: container.Image,
			Command: container.Command,
			Args: container.Args,
			WorkingDir: container.WorkingDir,
			Env: envVars,
		}
	}

	automountServiceAccountToken := false

	var pRuntimeClassName *string
	runtimeClassName := os.Getenv("RUNTIME_CLASS_NAME")
	if runtimeClassName != "" {
		pRuntimeClassName = &runtimeClassName
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			// Replicas: &replicas,   // Default to 1
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: containers,
					DNSPolicy: corev1.DNSDefault,
					AutomountServiceAccountToken: &automountServiceAccountToken,
					RuntimeClassName: pRuntimeClassName,
				},
			},
		},
	}
}

func serviceForCR(cr *codiusv1.Podius) *corev1.Service {
	labels := labelsForCR(cr)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Port: cr.Spec.Port,
				},
			},
		},
	}
}

// labelsForCR returns the labels for selecting the resources
// belonging to the given podius CR name.
func labelsForCR(cr *codiusv1.Podius) map[string]string {
	return map[string]string{"app": cr.Name}
}
