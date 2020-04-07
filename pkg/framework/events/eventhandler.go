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
}

type Callback func(obj interface{})

type TwoArgumentCallback func(obj interface{})

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

func (eh *EventHandler) handlePodDeleteEvent(obj interface{}) {}

func (eh *EventHandler) handlePodAddEvent(obj interface{}) {}

func (eh *EventHandler) handlePodUpdateEvent(oldObj, newObj interface{}) {}

func (eh *EventHandler) handlePvcDeleteEvent(obj interface{}) {}

func (eh *EventHandler) handlePvcAddEvent(obj interface{}) {}

func (eh *EventHandler) handlePvcUpdateEvent(oldObj, newObj interface{}) {}

func (eh *EventHandler) handleSetDeleteEvent(obj interface{}) {}

func (eh *EventHandler) handleSetAddEvent(obj interface{}) {}

func (eh *EventHandler) handleSetUpdateEvent(oldObj, newObj interface{}) {}
