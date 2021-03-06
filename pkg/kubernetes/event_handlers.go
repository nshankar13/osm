package kubernetes

import (
	"os"
	"reflect"

	"k8s.io/client-go/tools/cache"

	a "github.com/openservicemesh/osm/pkg/announcements"
	"github.com/openservicemesh/osm/pkg/constants"
	"github.com/openservicemesh/osm/pkg/kubernetes/events"
)

var emitLogs = os.Getenv(constants.EnvVarLogKubernetesEvents) == "true"

// observeFilter returns true for YES observe and false for NO do not pay attention to this
// This filter could be added optionally by anything using GetKubernetesEventHandlers()
type observeFilter func(obj interface{}) bool

// EventTypes is a struct helping pass the correct types to GetKubernetesEventHandlers
type EventTypes struct {
	Add    a.AnnouncementType
	Update a.AnnouncementType
	Delete a.AnnouncementType
}

// GetKubernetesEventHandlers creates Kubernetes events handlers.
func GetKubernetesEventHandlers(informerName, providerName string, shouldObserve observeFilter, eventTypes EventTypes) cache.ResourceEventHandlerFuncs {
	if shouldObserve == nil {
		shouldObserve = func(obj interface{}) bool { return true }
	}

	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if !shouldObserve(obj) {
				logNotObservedNamespace(obj, eventTypes.Add)
				return
			}
			events.GetPubSubInstance().Publish(events.PubSubMessage{
				AnnouncementType: eventTypes.Add,
				NewObj:           obj,
				OldObj:           nil,
			})
		},

		UpdateFunc: func(oldObj, newObj interface{}) {
			if !shouldObserve(newObj) {
				logNotObservedNamespace(newObj, eventTypes.Update)
				return
			}
			events.GetPubSubInstance().Publish(events.PubSubMessage{
				AnnouncementType: eventTypes.Update,
				NewObj:           oldObj,
				OldObj:           newObj,
			})
		},

		DeleteFunc: func(obj interface{}) {
			if !shouldObserve(obj) {
				logNotObservedNamespace(obj, eventTypes.Delete)
				return
			}
			events.GetPubSubInstance().Publish(events.PubSubMessage{
				AnnouncementType: eventTypes.Delete,
				NewObj:           nil,
				OldObj:           obj,
			})
		},
	}
}

func getNamespace(obj interface{}) string {
	return reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta").FieldByName("Namespace").String()
}

func logNotObservedNamespace(obj interface{}, eventType a.AnnouncementType) {
	if emitLogs {
		log.Debug().Msgf("Namespace %q is not observed by OSM; ignoring %s event", getNamespace(obj), eventType)
	}
}
