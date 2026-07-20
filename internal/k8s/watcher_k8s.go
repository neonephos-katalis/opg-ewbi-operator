/*
Copyright 2025.

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
package k8s

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/labstack/gommon/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var activeResourceWatchers sync.Map

var FederationRemoteEvents = make(chan event.GenericEvent)
var FileRemoteEvents = make(chan event.GenericEvent)
var ArtefactRemoteEvents = make(chan event.GenericEvent)
var ApplicationRemoteEvents = make(chan event.GenericEvent)
var ApplicationInstanceRemoteEvents = make(chan event.GenericEvent)

func StartRemoteResourceWatcher(ctx context.Context, hostClient dynamic.Interface, namespace, localResourceName, localResourceNS string, group, version, resource string) {
	watchKey := fmt.Sprintf("%s/%s", namespace, localResourceName)

	if _, loaded := activeResourceWatchers.LoadOrStore(watchKey, true); loaded {
		return
	}

	log.Info(">>> [K8s Watcher] Starting background watcher for remote host ", "key", watchKey)

	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(hostClient, 0, namespace, nil)
	informer := factory.ForResource(gvr).Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldU, okOld := oldObj.(*unstructured.Unstructured)
			newU, okNew := newObj.(*unstructured.Unstructured)

			if !okOld || !okNew {
				return
			}
			oldStatus := oldU.Object["status"]
			newStatus := newU.Object["status"]
			statusChanged := !reflect.DeepEqual(oldStatus, newStatus)
			if resource == "federations" {
				oldSpec := oldU.Object["spec"]
				newSpec := newU.Object["spec"]
				specChanged := !reflect.DeepEqual(oldSpec, newSpec)
				if !specChanged && !statusChanged {
					return
				}
			} else {
				if !statusChanged {
					return
				}
			}
			var targetChannel chan event.GenericEvent
			switch resource {
			case "federations":
				targetChannel = FederationRemoteEvents
			case "files":
				targetChannel = FileRemoteEvents
			case "artefacts":
				targetChannel = ArtefactRemoteEvents
			case "applications":
				targetChannel = ApplicationRemoteEvents
			case "applicationinstances":
				targetChannel = ApplicationInstanceRemoteEvents
			default:
				log.Error(nil, ">>> [K8s Watcher] Unknown resource type for event routing", "resource", resource)
				return
			}

			// Invio l'evento solo al canale della risorsa interessata
			targetChannel <- event.GenericEvent{
				Object: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{
							"name":      localResourceName,
							"namespace": localResourceNS,
						},
					},
				},
			}
		},
	})

	// Avviamo l'informer in background
	go informer.Run(ctx.Done())
}

func StopRemoteResourceWatcher(namespace, localResourceName string) {
	watchKey := fmt.Sprintf("%s/%s", namespace, localResourceName)
	if cancelVal, loaded := activeResourceWatchers.LoadAndDelete(watchKey); loaded {
		cancelFunc, ok := cancelVal.(context.CancelFunc)
		if ok {
			cancelFunc()
			log.Info(">>> [K8s Watcher] Successfully stopped and cleaned up background watcher", "key", watchKey)
		} else {
			log.Error(nil, ">>> [K8s Watcher] Invalid type found in activeResourceWatchers map, expected context.CancelFunc", "key", watchKey)
		}
	} else {
		log.Info(">>> [K8s Watcher] No active watcher found to stop for key", "key", watchKey)
	}
}
