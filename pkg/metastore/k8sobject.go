package metastore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	k8scli "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	opgv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/v1beta1"
)

type Opt func(obj metav1.Object) error

func WithOwnerReference(owner metav1.Object, scheme *runtime.Scheme) Opt {
	return func(obj metav1.Object) error {
		if err := ctrl.SetControllerReference(owner, obj, scheme); err != nil {
			return fmt.Errorf("failed to set owner reference: %w", err)
		}
		return nil
	}
}

// buildOwnerReferenceOption generates an Opt function that sets the owner reference
// of a Kubernetes Custom Resource to the specified Federation in a k8s object.
func (c *k8sClient) buildOwnerReferenceOption(federationContextID string) (Opt, error) {
	federation, err := c.getKubernetesObject(federationContextID, &opgv1beta1.FederationList{}, federationContextID)
	if err != nil {
		return nil, err
	}
	return WithOwnerReference(federation, c.getScheme()), nil
}

func (c *k8sClient) createK8sObject(object k8scli.Object) error {
	if err := c.kubernetes.Create(context.TODO(), object, &k8scli.CreateOptions{}); err != nil {
		errDetails := fmt.Sprintf("Failed to create %s (ID: %s)", getObjectKind(object), getObjectID(object))
		log.WithError(err).Error(errDetails)
		if k8serrors.IsAlreadyExists(err) {
			return errors.Wrapf(ErrAlreadyExists, "%s", errDetails)
		}
		return errors.Wrapf(err, "%s", errDetails)
	}
	return nil
}

// getKubernetesCallbackObject retrieves a Kubernetes object by id and federation callback id.
// It retrieve the objects searching for the id and federation callback labels.
func (c *k8sClient) getKubernetesCallbackObject(identifier string, objectList k8scli.ObjectList, fedCallbackID string) (k8scli.Object, error) {
	return c.searchKubernetesObject(objectList, labels.Set{
		opgLabel(federationCallbackIDLabel): fedCallbackID,
		opgLabel(idLabel):                   identifier,
		opgLabel(federationRelation):        guest,
	})
}

// getKubernetesObject retrieves a Kubernetes object by id and federation context id.
// It retrieve the objects searching for the id and federation context labels.
func (c *k8sClient) getKubernetesObject(identifier string, objectList k8scli.ObjectList, fedContextID string) (k8scli.Object, error) {
	return c.searchKubernetesObject(objectList, labels.Set{
		opgLabel(federationContextIDLabel): fedContextID,
		opgLabel(idLabel):                  identifier,
		opgLabel(federationRelation):       host,
	})
}

// searchKubernetesObject searches for a Kubernetes object using the specified labels.
// If multiple objects match, it returns the first one.
func (c *k8sClient) searchKubernetesObject(objectList k8scli.ObjectList, searchLabels labels.Set) (k8scli.Object, error) {
	objectList, err := c.searchKubernetesObjects(objectList, searchLabels)
	if err != nil {
		log.Errorf("failed to searchKubernetesObjects with labels '%v'", searchLabels)
		return nil, err
	}

	item, err := getFirstItemFromObjectList(objectList)
	if err != nil {
		kind := getListKind(objectList)
		log.WithError(err).Errorf("failed to search '%s' with labels '%v'", kind, searchLabels)
		return nil, fmt.Errorf("%s %w", kind, ErrNotFound)
	}
	return item, nil
}

// searchKubernetesObject searches for a Kubernetes object using the specified labels.
func (c *k8sClient) searchKubernetesObjects(objectList k8scli.ObjectList, searchLabels labels.Set) (k8scli.ObjectList, error) {
	kind := getListKind(objectList)
	selector := labels.SelectorFromSet(searchLabels)

	err := c.kubernetes.List(context.TODO(), objectList, &k8scli.ListOptions{
		Namespace:     c.getNamespace(),
		LabelSelector: selector,
	})
	if err != nil {
		log.Errorf("failed to list '%s' with labels '%v'", kind, searchLabels)
		return nil, errors.New("internal error")
	}

	return objectList, nil
}

func (c *k8sClient) updateK8sObject(object k8scli.Object) error {
	if err := c.kubernetes.Update(context.TODO(), object, &k8scli.UpdateOptions{}); err != nil {
		return errors.Wrapf(err, "unable to update object %T", object)
	}
	return nil
}

func (c *k8sClient) updateK8sObjectStatus(object k8scli.Object, status string) error {
	patch := []byte(fmt.Sprintf(`{"status":{"state":"%s"}}`, status)) // JSON Patch

	if err := c.kubernetes.Status().Patch(
		context.TODO(),
		object,
		k8scli.RawPatch(k8scli.Merge.Type(), patch),
		&k8scli.SubResourcePatchOptions{},
	); err != nil {
		return errors.Wrapf(err, "unable to update object %T", object)
	}
	return nil
}

func (c *k8sClient) updateK8sObjectAppInstStatus(object k8scli.Object, updates *models.AppInstCallbackLinkJSONRequestBody) (err error) {
	info := updates.AppInstanceInfo
	var patch struct {
		AccessPointInfo *models.AccessPointInfo `json:"accessPointInfo,omitempty"`
		State           *models.InstanceState    `json:"state,omitempty"`
	}

	if info.AppInstanceState != nil {
		patch.State = info.AppInstanceState
	}
	patch.AccessPointInfo = info.AccesspointInfo

	patchBytes, err := json.Marshal(map[string]any{"status": patch})
	if err != nil {
		return errors.Wrap(err, "failed to marshal status patch")
	}

	if err := c.kubernetes.Status().Patch(
		context.TODO(),
		object,
		k8scli.RawPatch(types.MergePatchType, patchBytes), // Usa types.MergePatchType
		&k8scli.SubResourcePatchOptions{},
	); err != nil {
		return errors.Wrapf(err, "unable to update object %T", object)
	}

	return nil
}

func getFirstItemFromObjectList(list k8scli.ObjectList) (k8scli.Object, error) {
	kind := getListKind(list)
	switch typedList := list.(type) {
	case *opgv1beta1.ApplicationInstanceList:
		if len(typedList.Items) == 0 {
			return nil, fmt.Errorf("no '%s' items found", kind)
		}
		return &typedList.Items[0], nil
	case *opgv1beta1.ApplicationList:
		if len(typedList.Items) == 0 {
			return nil, fmt.Errorf("no '%s' items found", kind)
		}
		return &typedList.Items[0], nil
	case *opgv1beta1.ArtefactList:
		if len(typedList.Items) == 0 {
			return nil, fmt.Errorf("no '%s' items found", kind)
		}
		return &typedList.Items[0], nil
	case *opgv1beta1.AvailabilityZoneList:
		if len(typedList.Items) == 0 {
			return nil, fmt.Errorf("no '%s' items found", kind)
		}
		return &typedList.Items[0], nil
	case *opgv1beta1.FederationList:
		if len(typedList.Items) == 0 {
			return nil, fmt.Errorf("no '%s' items found", kind)
		}
		return &typedList.Items[0], nil
	case *opgv1beta1.FileList:
		if len(typedList.Items) == 0 {
			return nil, fmt.Errorf("no '%s' items found", kind)
		}
		return &typedList.Items[0], nil
	default:
		return nil, fmt.Errorf("unsupported list type: %T", list)
	}
}

func getListKind(list k8scli.ObjectList) string {
	switch list.(type) {
	case *opgv1beta1.ApplicationInstanceList:
		return applicationInstanceKind
	case *opgv1beta1.ApplicationList:
		return applicationKind
	case *opgv1beta1.ArtefactList:
		return artefactKind
	case *opgv1beta1.AvailabilityZoneList:
		return availabilityZoneKind
	case *opgv1beta1.FederationList:
		return federationKind
	case *opgv1beta1.FileList:
		return fileKind
	default:
		return "Unknown"
	}
}

func getObjectKind(obj k8scli.Object) string {
	switch obj.(type) {
	case *opgv1beta1.ApplicationInstance:
		return applicationInstanceKind
	case *opgv1beta1.Application:
		return applicationKind
	case *opgv1beta1.Artefact:
		return artefactKind
	case *opgv1beta1.AvailabilityZone:
		return availabilityZoneKind
	case *opgv1beta1.Federation:
		return federationKind
	case *opgv1beta1.File:
		return fileKind
	default:
		return "Unknown"
	}
}

func getObjectID(object k8scli.Object) string {
	labels := object.GetLabels()
	return labels[opgLabel(idLabel)]
}
