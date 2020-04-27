package events

import (
	"github.com/rh-messaging/shipshape/pkg/framework/log"
	kubeinformers "k8s.io/client-go/informers"
	appsv1 "k8s.io/client-go/informers/apps/v1"
	corev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type EventHandler struct {
	statefulSetInformer appsv1.StatefulSetInformer
	pvcInformer         corev1.PersistentVolumeClaimInformer
	podInformer         corev1.PodInformer
	callbacks           map[Emitter]map[EventType]Callback
}

type Emitter int

const (
	StatefulSet Emitter = iota
	Pvc
	Pod
)

type EventType int

const (
	Delete EventType = iota
	Update
	Add
)

type Callback func(obj ...interface{})

// CreateEventHandler Creates event hanlder on framework initialization
func (eh *EventHandler) CreateEventInformers(kubeInformerFactory kubeinformers.SharedInformerFactory) {
	eh.statefulSetInformer = kubeInformerFactory.Apps().V1().StatefulSets()
	eh.pvcInformer = kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	eh.podInformer = kubeInformerFactory.Core().V1().Pods()
	log.Logf("Created informers")
	eh.podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    eh.handlePodAddEvent,
		UpdateFunc: eh.handlePodUpdateEvent,
		DeleteFunc: eh.handlePodDeleteEvent,
	})

	eh.pvcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    eh.handlePvcAddEvent,
		UpdateFunc: eh.handlePvcUpdateEvent,
		DeleteFunc: eh.handlePvcDeleteEvent,
	})

	eh.statefulSetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    eh.handleSetAddEvent,
		UpdateFunc: eh.handleSetUpdateEvent,
		DeleteFunc: eh.handleSetDeleteEvent,
	})
}

func (eh *EventHandler) ClearCallbacks() {
	eh.callbacks = make(map[Emitter]map[EventType]Callback)
}

func (eh *EventHandler) AddEventHandler(emitter Emitter, eventType EventType, callback Callback) {
	if eh.callbacks != nil {
		localMap := make(map[EventType]Callback)
		localMap[eventType] = callback
		eh.callbacks[emitter] = localMap
	} else {
		eh.callbacks = make(map[Emitter]map[EventType]Callback)
		eh.AddEventHandler(emitter, eventType, callback)
	}

}

func (eh *EventHandler) handlePodDeleteEvent(obj interface{}) {
	eh.handleEvents(Pod, Delete, obj)
}

func (eh *EventHandler) handlePodAddEvent(obj interface{}) {
	eh.handleEvents(Pod, Add, obj)
}

func (eh *EventHandler) handlePodUpdateEvent(oldObj, newObj interface{}) {
	eh.handleEvents(Pod, Update, oldObj, newObj)
}

func (eh *EventHandler) handlePvcDeleteEvent(obj interface{}) {
	eh.handleEvents(Pvc, Delete, obj)
}

func (eh *EventHandler) handlePvcAddEvent(obj interface{}) {
	eh.handleEvents(Pvc, Add, obj)
}

func (eh *EventHandler) handlePvcUpdateEvent(oldObj, newObj interface{}) {
	eh.handleEvents(Pvc, Update, oldObj, newObj)
}

func (eh *EventHandler) handleSetDeleteEvent(obj interface{}) {
	eh.handleEvents(StatefulSet, Delete, obj)
}

func (eh *EventHandler) handleSetAddEvent(obj interface{}) {
	eh.handleEvents(StatefulSet, Add, obj)
}

func (eh *EventHandler) handleSetUpdateEvent(oldObj, newObj interface{}) {
	eh.handleEvents(StatefulSet, Update, oldObj, newObj)
}

func (eh *EventHandler) handleEvents(emitter Emitter, typ EventType, obj ...interface{}) {
	callable := eh.callbacks[emitter][typ]
	if callable != nil {
		callable(obj)
	}
}
