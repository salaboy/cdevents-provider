/*
Copyright 2020 The Crossplane Authors.

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

package cluster

import (
	"context"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	commonv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	containerv1beta2 "github.com/crossplane/provider-gcp/apis/container/v1beta2"

	apisv1alpha1 "github.com/salaboy/cdevents-provider/apis/v1alpha1"
	cloudeventclient "github.com/salaboy/cdevents-provider/internal/cloudevents"
)

const (
	// errNotMyType    = "managed resource is not a MyType custom resource"
	errNotCluster   = "managed resource is not a Cluster resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	// errGetPC        = "cannot get ProviderConfig"
	// errGetCreds     = "cannot get credentials"

	errNewClient = "cannot create new Service"
)

// A NoOpService does nothing.
type NoOpService struct{}

var (
	newNoOpService = func(_ []byte) (interface{}, error) { return &NoOpService{}, nil }
)

// Setup adds a controller that reconciles MyType managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	// name := managed.ControllerName(v1alpha1.MyTypeGroupKind)
	name := managed.ControllerName(containerv1beta2.ClusterGroupKind)

	o := controller.Options{
		RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl), //nolint
	}

	r := managed.NewReconciler(mgr,
		// resource.ManagedKind(v1alpha1.MyTypeGroupVersionKind),
		resource.ManagedKind(containerv1beta2.ClusterGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: newNoOpService,
			logger:       l,
		}),
		managed.WithLogger(l.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		// For(&v1alpha1.MyType{}).
		For(&containerv1beta2.Cluster{}).
		Complete(r)
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        resource.Tracker
	newServiceFn func(creds []byte) (interface{}, error)
	logger       logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	// cr, ok := mg.(*v1alpha1.MyType)
	_, ok := mg.(*containerv1beta2.Cluster)
	if !ok {
		return nil, errors.New(errNotCluster)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	// pc := &apisv1alpha1.ProviderConfig{}
	// if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
	// 	return nil, errors.Wrap(err, errGetPC)
	// }

	// cd := pc.Spec.Credentials
	// data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	// if err != nil {
	// 	return nil, errors.Wrap(err, errGetCreds)
	// }

	svc, err := c.newServiceFn([]byte{}) // data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	protocol, err := cloudevents.NewHTTP()
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	cloudEventClient, err := cloudevents.NewClient(protocol, cloudevents.WithUUIDs(), cloudevents.WithTimeNow())
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{
		service:          svc,
		cloudEventClient: cloudEventClient,
		logger:           c.logger,
		kubeclient:       c.kube,
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	// A 'client' used to connect to the external resource API. In practice this
	// would be something like an AWS SDK client.
	service interface{}

	logger           logging.Logger
	cloudEventClient cloudevents.Client
	kubeclient       client.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	// cr, ok := mg.(*v1alpha1.MyType)
	cr, ok := mg.(*containerv1beta2.Cluster)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotCluster)
	}

	// These fmt statements should be removed in the real implementation.
	// fmt.Printf("Observing: %+v", cr)
	if len(cr.Status.Conditions) > 0 {
		for _, cond := range cr.Status.Conditions {
			if cond.Type == commonv1.TypeReady {
				if cond.Status == corev1.ConditionTrue {
					fmt.Println("*************** Cluster ready **************")

					// check if create event is already successfully sent
					ok, err := checkClusterCreationSuccessEvent(ctx, e.kubeclient, cr.Name)
					if err != nil {
						e.logger.Info("error checking event in register", "error", err.Error())
						return managed.ExternalObservation{
							ResourceExists:   true,
							ResourceUpToDate: true,
						}, err
					}

					if !ok {
						// this is the first occurrence of successful cluster creation
						ctx = context.WithValue(ctx, cloudeventclient.Logger, e.logger)
						ctx = cloudeventclient.InjectClient(ctx, e.cloudEventClient)
						ctx = cloudeventclient.SetTarget(ctx, "http://broker-ingress.knative-eventing.svc.cluster.local/default/default")

						err := cloudeventclient.SendEvent(ctx, cloudeventclient.EnvironmentCreated, cr)
						if err != nil {
							e.logger.Info("error sending cloud event", "error", err.Error())
							return managed.ExternalObservation{
								ResourceExists:   true,
								ResourceUpToDate: true,
							}, err
						}

						// register this for future de-duplication
						err = registerClusterCreationSuccessEvent(ctx, e.kubeclient, cr.Name)
						if err != nil {
							e.logger.Info("error registering successful event sent", "error", err.Error())
						}

						e.logger.Info("successfully registered sent out cloudevent with configmap")
					}
				}
			}
		}
	}

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the external resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: true,

		// Return any details that may be required to connect to the external
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	// cr, ok := mg.(*v1alpha1.MyType)
	cr, ok := mg.(*containerv1beta2.Cluster)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotCluster)
	}

	fmt.Printf("Creating: %+v", cr)

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	// cr, ok := mg.(*v1alpha1.MyType)
	cr, ok := mg.(*containerv1beta2.Cluster)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotCluster)
	}

	fmt.Printf("Updating: %+v", cr)

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	// cr, ok := mg.(*v1alpha1.MyType)
	cr, ok := mg.(*containerv1beta2.Cluster)
	if !ok {
		return errors.New(errNotCluster)
	}

	fmt.Printf("Deleting: %+v", cr)

	return nil
}
