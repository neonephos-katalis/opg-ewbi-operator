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
package watcher

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

// Usiamo una mappa per tracciare per quali host stiamo già facendo watching
var activeWatchers sync.Map
var RemoteFederationEvents = make(chan event.GenericEvent)

func StartRemoteWatcherFederation(ctx context.Context, hostClient dynamic.Interface, namespace, localFedName, localFedNamespace string) {
	// Chiave univoca per evitare di lanciare mille watch per lo stesso target
	watchKey := fmt.Sprintf("%s/%s", namespace, localFedName)

	if _, loaded := activeWatchers.LoadOrStore(watchKey, true); loaded {
		return // Watcher già in esecuzione per questo host
	}

	log.Info(">>> [Federation] Starting background watcher for remote host ", "key", watchKey)

	// Creiamo un informer dinamico per la risorsa remota
	gvr := schema.GroupVersionResource{Group: "opg.ewbi.nby.one", Version: "v1beta1", Resource: "federations"}
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(hostClient, 0, namespace, nil)
	informer := factory.ForResource(gvr).Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldU, okOld := oldObj.(*unstructured.Unstructured)
			newU, okNew := newObj.(*unstructured.Unstructured)

			// Safety check: se l'oggetto non è di tipo Unstructured (es. in caso di eventi anomali o tombstones), ignoriamo
			if !okOld || !okNew {
				return
			}

			// Estraiamo spec e status (se non esistono, la mappa restituirà nil, che reflect.DeepEqual gestisce correttamente)
			oldSpec := oldU.Object["spec"]
			newSpec := newU.Object["spec"]
			oldStatus := oldU.Object["status"]
			newStatus := newU.Object["status"]

			// Verifichiamo se c'è stato un effettivo cambiamento in spec o status
			specChanged := !reflect.DeepEqual(oldSpec, newSpec)
			statusChanged := !reflect.DeepEqual(oldStatus, newStatus)

			// Se non è cambiato né lo spec né lo status (es. aggiornamento di label, annotation, resourceVersion), scartiamo l'evento
			if !specChanged && !statusChanged {
				return
			}

			// Quando la risorsa remota cambia, inviamo un evento al Reconciler LOCALE
			// Questo rimetterà in coda la TUA risorsa Federation locale.
			RemoteFederationEvents <- event.GenericEvent{
				Object: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"metadata": map[string]interface{}{
							"name":      localFedName,
							"namespace": localFedNamespace,
						},
					},
				},
			}
		},
	})

	// Avviamo l'informer in background
	go informer.Run(ctx.Done())
}

func StopRemoteWatcherFederation(namespace, localFedName string) {
	watchKey := fmt.Sprintf("%s/%s", namespace, localFedName)
	if cancelVal, loaded := activeWatchers.LoadAndDelete(watchKey); loaded {
		cancelFunc, ok := cancelVal.(context.CancelFunc)
		if ok {
			cancelFunc()
			log.Info(">>> [Federation] Successfully stopped and cleaned up background watcher", "key", watchKey)
		} else {
			log.Error(nil, ">>> [Federation] Invalid type found in activeWatchers map, expected context.CancelFunc", "key", watchKey)
		}
	} else {
		log.Info(">>> [Federation] No active watcher found to stop for key", "key", watchKey)
	}
}
