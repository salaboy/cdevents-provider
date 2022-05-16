package cloudevent

import (
	"context"
	"errors"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	uuid "github.com/satori/go.uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cdevents "github.com/cdfoundation/sig-events/cde/sdk/go/pkg/cdf/events"
)

// CrossplaneEvent defines an event in the crossplane realm
type CrossplaneEvent string

const (
	// EnvironmentCreated is the event used when the cluster is ready
	EnvironmentCreated CrossplaneEvent = "created"
	// EnvironmentModified is the event used when the cluster is modified
	EnvironmentModified CrossplaneEvent = "modified"
)

var (
	eventMap = crossplaneToCDEventMap{
		EnvironmentCreated:  cdevents.EnvironmentCreatedEventV1,
		EnvironmentModified: cdevents.EnvironmentModifiedEventV1,
	}
)

type crossplaneToCDEventMap map[CrossplaneEvent]cdevents.CDEventType

type cECKey struct{}

type key int

const (
	// EventSink defines the endpoint for sink for cloud events
	EventSink key = iota
	// Logger is the key in context for the logget object
	Logger
)

// func init() {
// 	injection.Default.RegisterClient(withCloudEventClient)
// }

// func withCloudEventClient(ctx context.Context, cfg *rest.Config) context.Context {
// 	logger := logging.FromContext(ctx)
//
// 	protocol, err := cloudevents.NewHTTP()
// 	if err != nil {
// 		logger.Panicf("Error creating the cloudevents http protocol: %s", err)
// 	}
//
// 	cloudEventClient, err := cloudevents.NewClient(protocol, cloudevents.WithUUIDs(), cloudevents.WithTimeNow())
// 	if err != nil {
// 		logger.Panicf("Error creating the cloudevents client: %s", err)
// 	}
//
// 	return context.WithValue(ctx, CECKey{}, cloudEventClient)
// }

// Get returns a cloud events client or error
func Get(ctx context.Context) (cloudevents.Client, error) {
	l := ctx.Value(Logger)
	if l == nil {
		return nil, fmt.Errorf("cannot get logger from context")
	}

	logger, ok := l.(logging.Logger) // logging.FromContext(ctx)
	if !ok {
		return nil, errors.New("cannot get logger from context")
	}

	untyped := ctx.Value(cECKey{})
	if untyped == nil {
		logger.Info(
			"Unable to fetch client from context.")
		return nil, fmt.Errorf("Unable to fetch client from context")
	}

	client, ok := untyped.(cloudevents.Client)
	if !ok {
		return nil, fmt.Errorf("cannot caste to clouevent client: %T", untyped)
	}

	return client, nil
}

// InjectClient allows callers to inject a cloud events client into the context
// with a well defined key
func InjectClient(ctx context.Context, client cloudevents.Client) context.Context {
	return context.WithValue(ctx, cECKey{}, client)
}

// SetTarget is used to inject the sink endpoint for the cloudevents
func SetTarget(ctx context.Context, target string) context.Context {
	return context.WithValue(ctx, EventSink, target)
}

// SendEvent is responsible for sending the crossplane event to the injected
// event sink endpoint
func SendEvent(ctx context.Context, eventType CrossplaneEvent, obj metav1.Object) error {
	logger, ok := ctx.Value(Logger).(logging.Logger) // logging.FromContext(ctx)
	if !ok {
		return errors.New("cannot get logger from context")
	}

	// if eventType == EnvironmentCreated {
	// 	logger.Info("SendEvent received", "event", EnvironmentCreated)
	// }

	cdEvent, ok := eventMap[eventType]
	if !ok {
		logger.Info("no known cloud event mapping found", "event type", eventType)
		return fmt.Errorf("no known cloud event mapping found for event type %s", eventType)
	}

	client, err := Get(ctx)
	if err != nil {
		logger.Info("unable to get cloud event client", "error", err.Error())
		return err
	}

	switch eventType {
	case EnvironmentCreated:
		event := createEvent(cdEvent.String(), obj)

		target := ctx.Value(EventSink).(string)

		ctx := injectIntoContext(ctx, target)
		result := client.Send(ctx, event)
		if !cloudevents.IsACK(result) {
			logger.Info("Failed to get ack for cloudevent: ", "error", result.Error())
			return result
		}

		if cloudevents.IsUndelivered(result) {
			logger.Info("failed sending cloud event, error: ", result.Error())
			return result
		}

		logger.Info("Sent event for type ", "event type", EnvironmentCreated)

	case EnvironmentModified:
		//		event := createEvent(cdEvent.String(), obj)
		//		ctx := injectIntoContext(ctx, "http://localhost:8080")
		//		result := client.Send(ctx, event)
		//		if !cloudevents.IsACK(result) {
		//			logger.Info("Failed to send cloudevent: ", result.Error())
		//		}
		//
		//		if cloudevents.IsUndelivered(result) {
		//			logger.Info("cloud event undelivered, error: ", result.Error())
		//		}
		//
		logger.Info("received event for type ", EnvironmentModified)

	default:
		logger.Info("unknown event type ", eventType)
	}

	return nil
}

func injectIntoContext(c context.Context, target string) context.Context {
	ctx := cloudevents.ContextWithRetriesExponentialBackoff(c, 10*time.Millisecond, 10)
	ctx = cloudevents.ContextWithTarget(ctx, target)

	return ctx
}

func createEvent(cdEventType string, obj metav1.Object) cloudevents.Event {
	event := cloudevents.NewEvent()
	event.SetSource(obj.GetNamespace() + "/" + obj.GetName())
	// event.SetSource("crossplane-controller")
	event.SetID(uuid.NewV4().String())
	event.SetType(cdEventType)
	event.SetTime(time.Now())
	event.SetData("application/json", map[string]interface{}{
		"path":            "workspace/source/config/",
		"git_source_name": "cluster-git",
		"docker_repo":     "ishankhare07",
	})

	return event
}
