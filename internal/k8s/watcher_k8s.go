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

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var activeResourceWatchers sync.Map

var FederationRemoteEvents = make(chan event.GenericEvent)
var ImageRemoteEvents = make(chan event.GenericEvent)
var ArtefactRemoteEvents = make(chan event.GenericEvent)
var ApplicationOnboardingRemoteEvents = make(chan event.GenericEvent)
var ApplicationDeploymentRemoteEvents = make(chan event.GenericEvent)
var AvailabilityZoneRemoteEvents = make(chan event.GenericEvent)

func StartRemoteResourceWatcher(ctx context.Context, dynClient dynamic.Interface, namespace, localResourceName, localResourceNS string, group, version, resource string) {
	watchKey := fmt.Sprintf("%s/%s", namespace, localResourceName)
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	if _, loaded := activeResourceWatchers.LoadOrStore(watchKey, cancelFunc); loaded {
		cancelFunc()
		return
	}
	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynClient, 0, namespace, nil)
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
			case "availabilityzones":
				targetChannel = AvailabilityZoneRemoteEvents
			case "images":
				targetChannel = ImageRemoteEvents
			case "artefacts":
				targetChannel = ArtefactRemoteEvents
			case "applicationonboardings":
				targetChannel = ApplicationOnboardingRemoteEvents
			case "applicationdeployments":
				targetChannel = ApplicationDeploymentRemoteEvents
			default:
				return
			}
			// Send the event to the appropriate channel
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

	// Start the informer in a separate goroutine
	go informer.Run(cancelCtx.Done())
}

func StopRemoteResourceWatcher(namespace, localResourceName string) bool {
	watchKey := fmt.Sprintf("%s/%s", namespace, localResourceName)
	if cancelVal, loaded := activeResourceWatchers.LoadAndDelete(watchKey); loaded {
		cancelFunc, ok := cancelVal.(context.CancelFunc)
		if ok {
			cancelFunc()
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

const WatcherStoppedAnnotation = "opg-ewbi-katalis.com/watcher-stopped-for-lock"

// StopRemoteResourceWatcherFedLocked stops the remote resource watcher for a given resource and sets an annotation on the resource to indicate that the watcher has been stopped due to federation being locked.
func StopRemoteResourceWatcherFedLocked(ctx context.Context, localClient client.Client, resource client.Object, remoteNamespace string, localName string) error {
	// Stop the remote resource watcher for the given resource
	StopRemoteResourceWatcher(remoteNamespace, localName)
	annotations := resource.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	// Set the annotation to indicate that the watcher has been stopped due to federation being locked
	if annotations[WatcherStoppedAnnotation] != "true" {
		annotations[WatcherStoppedAnnotation] = "true"
		resource.SetAnnotations(annotations)
		err := localClient.Update(ctx, resource)
		if err != nil {
			return err
		}
	}
	return nil
}

// RestartAllRemoteWatcher restarts the remote resource watcher for a given resource if it was previously stopped due to federation being locked, and removes the annotation indicating that the watcher was stopped.
func RestartAllRemoteWatcher(ctx context.Context, c client.Client, fed *v1beta1.Federation, scheme *runtime.Scheme, namespace string, federationContextId string, lists ...client.ObjectList) error {
	annotations := fed.GetAnnotations()
	// Check if the annotation indicating that the watcher was stopped due to federation being locked exists
	if val, exists := annotations[WatcherStoppedAnnotation]; exists && val == "true" {
		_, dynClient, err := buildHostClient(ctx, fed, c, scheme)
		if err != nil {
			return err
		}
		for _, list := range lists {
			// List all resources of the given type in the specified namespace
			if err := c.List(ctx, list, client.InNamespace(namespace)); err != nil {
				return err
			}
			// Extract the items from the list and iterate over them
			objs, err := meta.ExtractList(list)
			if err != nil {
				return err
			}
			for _, obj := range objs {
				metaObj, err := meta.Accessor(obj)
				if err != nil {
					continue
				}
				resourceName := metaObj.GetName()
				resourceNamespace := metaObj.GetNamespace()
				resourceGroup := obj.GetObjectKind().GroupVersionKind().Group
				resourceVersion := obj.GetObjectKind().GroupVersionKind().Version
				resourcePlural := obj.GetObjectKind().GroupVersionKind().Kind
				var fedContextId string
				// Extract the FederationContextId based on the type of Custom Resource
				switch cr := obj.(type) {
				case *v1beta1.Image:
					fedContextId = cr.Spec.FederationContextId
				case *v1beta1.AvailabilityZone:
					fedContextId = cr.Spec.FederationContextId
				case *v1beta1.Artefact:
					fedContextId = cr.Spec.FederationContextId
				case *v1beta1.ApplicationDeployment:
					fedContextId = cr.Spec.FederationContextId
				case *v1beta1.ApplicationOnboarding:
					fedContextId = cr.Spec.FederationContextId
				default:
					continue
				}
				// Compare the FederationContextId of the CR with the target FederationContextId
				if fedContextId == federationContextId {
					StartRemoteResourceWatcher(ctx, dynClient, fed.Spec.FederationData.K8sOptions.Namespace, resourceName, resourceNamespace, resourceGroup, resourceVersion, resourcePlural)

				}
			}
		}
		delete(annotations, WatcherStoppedAnnotation)
		fed.SetAnnotations(annotations)
		if err := c.Update(ctx, fed); err != nil {
			return err
		}
	}
	return nil
}

func StopAllRemoteWatchers(ctx context.Context, c client.Client, fed *v1beta1.Federation, namespace string, federationContextId string, lists ...client.ObjectList) error {
	for _, list := range lists {
		// List all resources of the given type in the specified namespace
		if err := c.List(ctx, list, client.InNamespace(namespace)); err != nil {
			return err
		}
		// Extract the items from the list and iterate over them
		objs, err := meta.ExtractList(list)
		if err != nil {
			return err
		}
		for _, obj := range objs {
			metaObj, err := meta.Accessor(obj)
			if err != nil {
				continue
			}
			resourceName := metaObj.GetName()
			var fedContextId string
			// Extract the FederationContextId based on the type of Custom Resource
			switch cr := obj.(type) {
			case *v1beta1.Image:
				fedContextId = cr.Spec.FederationContextId
			case *v1beta1.AvailabilityZone:
				fedContextId = cr.Spec.FederationContextId
			case *v1beta1.Artefact:
				fedContextId = cr.Spec.FederationContextId
			case *v1beta1.ApplicationDeployment:
				fedContextId = cr.Spec.FederationContextId
			case *v1beta1.ApplicationOnboarding:
				fedContextId = cr.Spec.FederationContextId
			default:
				continue
			}
			// Compare the FederationContextId of the CR with the target FederationContextId
			if fedContextId == federationContextId {
				StopRemoteResourceWatcher(fed.Spec.FederationData.K8sOptions.Namespace, resourceName)
			}
		}
	}
	annotations := fed.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	// Set the annotation to indicate that the watcher has been stopped due to federation being locked
	if annotations[WatcherStoppedAnnotation] != "true" {
		annotations[WatcherStoppedAnnotation] = "true"
		fed.SetAnnotations(annotations)
		err := c.Update(ctx, fed)
		if err != nil {
			return err
		}
	}
	return nil
}
