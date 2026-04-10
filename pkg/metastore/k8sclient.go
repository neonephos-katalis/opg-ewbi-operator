package metastore

import (
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8scli "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	opgv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/v1beta1"
)

type k8sClient struct {
	kubernetes k8scli.Client
	namespace  string
}

func NewK8sClient(c k8scli.Client, namespace string) *k8sClient {
	return &k8sClient{c, namespace}
}

func (c *k8sClient) AddApplicationInstance(ctx context.Context, dep *ApplicationInstance) (*opgv1beta1.ApplicationInstance, error) {
	if _, err := c.GetApplication(ctx, dep.FederationContextId, dep.AppId); err != nil {
		if IsNotFoundError(err) {
			return nil, errors.Wrap(ErrBadRequest, err.Error())
		}
	}
	opt, err := c.buildOwnerReferenceOption(dep.FederationContextId)
	if err != nil {
		return nil, err
	}
	obj, err := dep.k8sCustomResource(c.getNamespace(), opt)
	if err != nil {
		return nil, err
	}
	err = c.createK8sObject(obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (c *k8sClient) getFederation(federationContextID string) (*opgv1beta1.Federation, error) {
	obj, err := c.getKubernetesObject(federationContextID, &opgv1beta1.FederationList{}, federationContextID)
	if err != nil {
		return nil, err
	}
	fed, ok := obj.(*opgv1beta1.Federation)
	if !ok {
		return nil, missMatchErr("federation", federationContextID, federationContextID, &opgv1beta1.Federation{}, obj)
	}
	return fed, nil
}

func (c *k8sClient) AddAvailabilityZones(ctx context.Context, federationContextID string, azs []string) error {
	obj, err := c.getFederation(federationContextID)
	if err != nil {
		return err
	}
	obj.Spec.AcceptedAvailabilityZones = mergeUnique(obj.Spec.AcceptedAvailabilityZones, azs)
	return c.updateK8sObject(obj)
}

func (c *k8sClient) CreateFederation(ctx context.Context, input *Federation) (*Federation, error) {
	obj, err := c.searchKubernetesObject(&opgv1beta1.FederationList{}, labels.Set{
		opgLabel(clientIDLabel):      input.ClientCredentials.ClientID,
		opgLabel(federationRelation): host,
	})
	if err != nil {
		return nil, err
	}

	fed := obj.(*opgv1beta1.Federation)

	// ensure federation is not already set
	if !fed.Spec.InitialDate.IsZero() {
		return nil, errors.Wrapf(ErrAlreadyExists, "Failed to create federation (ClientID: %s)", input.ClientCredentials.ClientID)
	}

	cr := input.updatek8sCustomResource(fed)

	if err := c.updateK8sObject(cr); err != nil {
		return nil, err
	}

	res, err := federationFromK8sCustomResource(fed)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *k8sClient) GetApplication(ctx context.Context, federationContextID, id string) (*Application, error) {
	app, err := c.getKubernetesObject(id, &opgv1beta1.ApplicationList{}, federationContextID)
	if err != nil {
		return nil, err
	}
	res, ok := app.(*opgv1beta1.Application)
	if !ok {
		return nil, missMatchErr("application", id, federationContextID, &opgv1beta1.Application{}, app)
	}
	return applicationFromK8sCustomResource(*res)
}

func (c *k8sClient) GetArtefact(ctx context.Context, federationContextID, id string) (*Artefact, error) {
	artefact, err := c.getKubernetesObject(id, &opgv1beta1.ArtefactList{}, federationContextID)
	if err != nil {
		return nil, err
	}
	res, ok := artefact.(*opgv1beta1.Artefact)
	if !ok {
		return nil, missMatchErr("artefact", id, federationContextID, &opgv1beta1.Artefact{}, artefact)
	}
	return artefactFromK8sCustomResource(*res)
}

func (c *k8sClient) GetAvailabilityZone(ctx context.Context, federationContextID, id string) (*PartnerAvailabilityZone, error) {
	obj := &opgv1beta1.AvailabilityZone{}
	if err := c.kubernetes.Get(context.TODO(), types.NamespacedName{Name: id, Namespace: c.getNamespace()}, obj, &k8scli.GetOptions{}); err != nil {
		return nil, errors.Wrapf(err, "unable to find the requested az")
	}
	paz, err := partnerAvailabilityZoneFromK8sAvailabilityZone(obj)
	if err != nil {
		return nil, err
	}
	return paz, err
}

func (c *k8sClient) GetFederation(ctx context.Context, federationContextID string) (*Federation, error) {
	obj, err := c.getFederation(federationContextID)
	if err != nil {
		return nil, err
	}

	fed, err := federationFromK8sCustomResource(obj)
	if err != nil {
		return nil, err
	}
	return fed, nil
}

func (c *k8sClient) GetFile(ctx context.Context, federationContextID, id string) (*File, error) {
	file, err := c.getKubernetesObject(id, &opgv1beta1.FileList{}, federationContextID)
	if err != nil {
		return nil, err
	}
	res, ok := file.(*opgv1beta1.File)
	if !ok {
		return nil, missMatchErr("file", id, federationContextID, &opgv1beta1.File{}, file)
	}
	return fileFromK8sCustomResource(id, *res)
}

func (c *k8sClient) ListAvailabilityZones(ctx context.Context) ([]*PartnerAvailabilityZone, error) {
	azList := &opgv1beta1.AvailabilityZoneList{}

	if err := c.kubernetes.List(context.TODO(), azList, &k8scli.ListOptions{Namespace: c.getNamespace()}); err != nil {
		return nil, errors.Wrapf(err, "failed to list availability zones")
	}
	var pazs []*PartnerAvailabilityZone
	azs := azList.Items
	for _, az := range azs {
		paz, err := partnerAvailabilityZoneFromK8sAvailabilityZone(&az)
		if err != nil {
			return nil, err
		}
		pazs = append(pazs, paz)
	}
	return pazs, nil
}

func (c *k8sClient) OnboardApplication(ctx context.Context, app *OnboardApplication) (*opgv1beta1.Application, error) {
	for _, artefact := range app.artefacts() {
		if _, err := c.GetArtefact(ctx, app.FederationContextId, artefact); err != nil {
			if IsNotFoundError(err) {
				return nil, errors.Wrap(ErrBadRequest, err.Error())
			}
		}
	}
	opt, err := c.buildOwnerReferenceOption(app.FederationContextId)
	if err != nil {
		return nil, err
	}
	obj, err := app.k8sCustomResource(c.getNamespace(), opt)
	if err != nil {
		return nil, err
	}
	err = c.createK8sObject(obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (c *k8sClient) RemoveApplication(ctx context.Context, federationContextID, id string) error {
	appId := k8sCustomResourceNameFromApplicationID(federationContextID, id)
	if err := c.kubernetes.Delete(context.TODO(), &opgv1beta1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appId,
			Namespace: c.getNamespace(),
		},
	}, &k8scli.DeleteOptions{}); err != nil {
		return errors.Wrapf(err, "unable to remove application")
	}
	return nil
}

func (c *k8sClient) RemoveApplicationInstance(ctx context.Context, federationContextID, id string) error {
	appIns := k8sCustomResourceNameFromApplicationInstance(federationContextID, id)
	if err := c.kubernetes.Delete(context.TODO(), &opgv1beta1.ApplicationInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appIns,
			Namespace: c.getNamespace(),
		},
	}, &k8scli.DeleteOptions{}); err != nil {
		return errors.Wrapf(err, "unable to remove application instance")
	}
	return nil
}

func (c *k8sClient) RemoveArtefact(ctx context.Context, federationContextID, id string) error {
	appIns := k8sCustomResourceNameFromArtefactID(federationContextID, id)
	if err := c.kubernetes.Delete(context.TODO(), &opgv1beta1.Artefact{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appIns,
			Namespace: c.getNamespace(),
		},
	}, &k8scli.DeleteOptions{}); err != nil {
		return errors.Wrapf(err, "unable to remove artefact")
	}
	return nil
}

func (c *k8sClient) RemoveFederation(ctx context.Context, federationContextID string) error {
	obj, err := c.getFederation(federationContextID)
	if err != nil {
		return err
	}
	if err := c.kubernetes.Delete(context.TODO(), obj, &k8scli.DeleteOptions{}); err != nil {
		return errors.Wrapf(err, "unable to remove federation")
	}
	return nil
}

func (c *k8sClient) RemoveFile(ctx context.Context, federationContextID, id string) error {
	fileID := k8sCustomResourceNameFromFileID(federationContextID, id)
	if err := c.kubernetes.Delete(context.TODO(), &opgv1beta1.File{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fileID,
			Namespace: c.getNamespace(),
		},
	}, &k8scli.DeleteOptions{}); err != nil {
		return errors.Wrapf(err, "unable to remove file")
	}
	return nil
}

func (c *k8sClient) UpdateFileStatus(ctx context.Context, federationCallbackID string, updates *models.FileStatusCallbackLinkJSONRequestBody) error {
	id := updates.FileId
	obj, err := c.getKubernetesCallbackObject(id, &opgv1beta1.FileList{}, federationCallbackID)
	if err != nil {
		return err
	}
	res, ok := obj.(*opgv1beta1.File)
	if !ok {
		return missMatchErr("file", id, federationCallbackID, &opgv1beta1.File{}, obj)
	}
	state := string(updates.UpdateStatus)
	if isValidFileStatus(state) {
		return c.updateK8sObjectStatus(res, state)
	}
	return nil
}

func (c *k8sClient) UpdateArtefactStatus(ctx context.Context, federationCallbackID string, updates *models.ArtefactStatusCallbackLinkJSONRequestBody) error {
	id := updates.ArtefactId
	obj, err := c.getKubernetesCallbackObject(id, &opgv1beta1.ArtefactList{}, federationCallbackID)
	if err != nil {
		return err
	}
	res, ok := obj.(*opgv1beta1.Artefact)
	if !ok {
		return missMatchErr("artefact", id, federationCallbackID, &opgv1beta1.Artefact{}, obj)
	}
	state := string(updates.UpdateStatus)
	if isValidArtefactStatus(state) {
		return c.updateK8sObjectStatus(res, state)
	}
	return nil
}

func (c *k8sClient) UpdateApplicationStatus(ctx context.Context, federationCallbackID string, updates *models.AppStatusCallbackLinkJSONRequestBody) error {
	id := updates.AppId
	obj, err := c.getKubernetesCallbackObject(id, &opgv1beta1.ApplicationList{}, federationCallbackID)
	if err != nil {
		return err
	}
	res, ok := obj.(*opgv1beta1.Application)
	if !ok {
		return missMatchErr("application", id, federationCallbackID, &opgv1beta1.ApplicationInstance{}, obj)
	}
	if len(updates.StatusInfo) > 0 {
		state := string(updates.StatusInfo[0].OnboardStatusInfo)
		if isValidApplicationStatus(state) {
			return c.updateK8sObjectStatus(res, state)
		}
	}
	return nil
}

func (c *k8sClient) UpdateApplicationInstanceStatus(ctx context.Context, federationCallbackID string, updates *models.AppInstCallbackLinkJSONRequestBody) error {
	id := updates.AppInstanceId
	obj, err := c.getKubernetesCallbackObject(id, &opgv1beta1.ApplicationInstanceList{}, federationCallbackID)
	if err != nil {
		return err
	}
	res, ok := obj.(*opgv1beta1.ApplicationInstance)
	if !ok {
		return missMatchErr("application instance", id, federationCallbackID, &opgv1beta1.ApplicationInstance{}, obj)
	}
	if updates.AppInstanceInfo.AppInstanceState != nil {
		state := string(*updates.AppInstanceInfo.AppInstanceState)
		if isValidApplicationInstanceStatus(state) {
			return c.updateK8sObjectAppInstStatus(res, updates)
		}
	}
	return nil
}

func (c *k8sClient) UpdateFederationStatus(ctx context.Context, federationCallbackID string, status models.Status) error {
	obj, err := c.searchKubernetesObject(&opgv1beta1.FederationList{}, labels.Set{
		opgLabel(federationCallbackIDLabel): federationCallbackID,
		opgLabel(federationRelation):        guest,
	})
	if err != nil {
		return err
	}
	res, ok := obj.(*opgv1beta1.Federation)
	if !ok {
		return missMatchErr("federation", federationCallbackID, federationCallbackID, &opgv1beta1.Federation{}, obj)
	}

	state := string(status)
	if isValidFederationStatus(state) {
		return c.updateK8sObjectStatus(res, state)
	}
	return nil
}

func (c *k8sClient) UploadArtefact(ctx context.Context, artefact *UploadArtefact) (*opgv1beta1.Artefact, error) {
	for _, file := range artefact.files() {
		if _, err := c.GetFile(ctx, artefact.FederationContextId, file); err != nil {
			if IsNotFoundError(err) {
				return nil, errors.Wrap(ErrBadRequest, err.Error())
			}
		}
	}
	opt, err := c.buildOwnerReferenceOption(artefact.FederationContextId)
	if err != nil {
		return nil, err
	}
	obj, err := artefact.k8sCustomResource(c.getNamespace(), opt)
	if err != nil {
		return nil, err
	}
	err = c.createK8sObject(obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (c *k8sClient) UploadFile(ctx context.Context, file *UploadFile) (*opgv1beta1.File, error) {
	opt, err := c.buildOwnerReferenceOption(file.FederationContextId)
	if err != nil {
		return nil, err
	}
	obj, err := file.k8sCustomResource(c.getNamespace(), opt)
	if err != nil {
		return nil, err
	}
	err = c.createK8sObject(obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (c *k8sClient) getNamespace() string {
	return c.namespace
}

func (c *k8sClient) getScheme() *runtime.Scheme {
	return c.kubernetes.Scheme()
}

func (c *k8sClient) GetApplicationInstanceDetails(ctx context.Context, federationContextID, id string) (*ApplicationInstanceDetails, error) {
	//return nil, errors.Errorf("method not implemented")
	application, err := c.getKubernetesObject(id, &opgv1beta1.ApplicationInstanceList{}, federationContextID)
	if err != nil {
		return nil, err
	}
	res, ok := application.(*opgv1beta1.ApplicationInstance)
	if !ok {
		return nil, missMatchErr("application instance", id, federationContextID, &opgv1beta1.ApplicationInstance{}, application)
	}
	return applicationInstanceFromK8sCustomResource(id, *res)
}
