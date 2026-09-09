package metastore

import (
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8scli "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	v1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
)

type k8sClient struct {
	kubernetes k8scli.Client
	namespace  string
}

func NewK8sClient(c k8scli.Client, namespace string) *k8sClient {
	return &k8sClient{c, namespace}
}

func (c *k8sClient) getNamespace() string {
	return c.namespace
}

func (c *k8sClient) getScheme() *runtime.Scheme {
	return c.kubernetes.Scheme()
}

func (c *k8sClient) AddApplicationDeployment(ctx context.Context, dep *ApplicationInstance) (*v1beta1.ApplicationDeployment, error) {
	if _, err := c.GetApplication(ctx, dep.FederationContextId, dep.AppId); err != nil {
		if IsNotFoundError(err) {
			return nil, errors.Wrap(ErrBadRequest, err.Error())
		}
	}
	if _, err := c.searchFederation(ctx, dep.FederationContextId, "HOST"); err != nil {
		return nil, err
	}
	obj, err := dep.k8sCustomResource(c.getNamespace())
	if err != nil {
		return nil, err
	}
	err = c.createK8sObject(obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (c *k8sClient) getFederation(federationContextID string) (*v1beta1.Federation, error) {
	obj, err := c.getKubernetesObject(federationContextID, &v1beta1.FederationList{}, federationContextID)
	if err != nil {
		return nil, err
	}
	fed, ok := obj.(*v1beta1.Federation)
	if !ok {
		return nil, missMatchErr("federation", federationContextID, federationContextID, &v1beta1.Federation{}, obj)
	}
	return fed, nil
}

func (c *k8sClient) GetApplication(ctx context.Context, federationContextID, id string) (*Application, error) {
	if _, err := c.searchFederation(ctx, federationContextID, "HOST"); err != nil {
		return nil, err
	}
	res, err := c.searchApplication(ctx, federationContextID, id, "HOST")
	if err != nil {
		return nil, err
	}
	return applicationFromK8sCustomResource(*res)
}

func (c *k8sClient) GetAvailabilityZone(ctx context.Context, federationContextID, id string) (*PartnerAvailabilityZone, error) {
	obj := &v1beta1.AvailabilityZone{}
	if err := c.kubernetes.Get(context.TODO(), types.NamespacedName{Name: id, Namespace: c.getNamespace()}, obj, &k8scli.GetOptions{}); err != nil {
		return nil, errors.Wrapf(err, "unable to find the requested az")
	}
	paz, err := partnerAvailabilityZoneFromK8sAvailabilityZone(obj)
	if err != nil {
		return nil, err
	}
	return paz, err
}

func (c *k8sClient) ListAvailabilityZones(ctx context.Context) ([]*PartnerAvailabilityZone, error) {
	azList := &v1beta1.AvailabilityZoneList{}

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

func (c *k8sClient) OnboardApplication(ctx context.Context, app *OnboardApplication) (*v1beta1.ApplicationOnboarding, error) {
	for _, artefact := range app.artefacts() {
		if _, err := c.GetArtefact(ctx, app.FederationContextId, artefact); err != nil {
			if IsNotFoundError(err) {
				return nil, errors.Wrap(ErrBadRequest, err.Error())
			}
		}
	}
	if _, err := c.searchFederation(ctx, app.FederationContextId, "HOST"); err != nil {
		return nil, err
	}
	obj, err := app.k8sCustomResource(c.getNamespace())
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
	if err := c.kubernetes.Delete(context.TODO(), &v1beta1.ApplicationOnboarding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appId,
			Namespace: c.getNamespace(),
		},
	}, &k8scli.DeleteOptions{}); err != nil {
		return errors.Wrapf(err, "unable to remove application")
	}
	return nil
}

func (c *k8sClient) RemoveApplicationDeployment(ctx context.Context, federationContextID, appInstanceId, appId string) error {
	appIns := "appdeploy-" + uuidV5Fn(federationContextID+appId+appInstanceId)
	if err := c.kubernetes.Delete(context.TODO(), &v1beta1.ApplicationDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appIns,
			Namespace: c.getNamespace(),
		},
	}, &k8scli.DeleteOptions{}); err != nil {
		return errors.Wrapf(err, "unable to remove application instance")
	}
	return nil
}

func (c *k8sClient) UpdateApplicationStatus(ctx context.Context, federationCallbackID string, updates *models.AppStatusCallbackLinkJSONRequestBody) error {
	res, err := c.searchApplication(ctx, federationCallbackID, updates.AppId, "GUEST")
	if err != nil {
		return err
	}
	if len(updates.StatusInfo) > 0 {
		state := string(updates.StatusInfo[0].OnboardStatusInfo)
		//if isValidApplicationStatus(state) {
		return c.updateK8sObjectStatus(res, state)
		//}
	}
	return nil
}

func (c *k8sClient) UpdateApplicationDeploymentStatus(ctx context.Context, federationCallbackID string, updates *models.AppInstCallbackLinkJSONRequestBody) error {
	res, err := c.searchApplicationDeployment(ctx, federationCallbackID, updates.AppInstanceId, "GUEST")
	if err != nil {
		return err
	}
	if updates.AppInstanceInfo.AppInstanceState != nil {
		//state := string(*updates.AppInstanceInfo.AppInstanceState)
		//if isValidApplicationDeploymentStatus(state) {
		return c.updateK8sObjectAppInstStatus(res, updates)
		//}
	}
	return nil
}

func (c *k8sClient) GetApplicationDeploymentDetails(ctx context.Context, federationContextID, id string) (*ApplicationInstanceDetails, error) {
	if _, err := c.searchFederation(ctx, federationContextID, "HOST"); err != nil {
		return nil, err
	}
	res, err := c.searchApplicationDeployment(ctx, federationContextID, id, "HOST")
	if err != nil {
		return nil, err
	}
	return applicationDeploymentFromK8sCustomResource(id, *res)
}
func (c *k8sClient) GetApplicationDeployment(ctx context.Context, federationContextID, id string) (*ApplicationInstance, error) {
	return nil, errors.Errorf("method not implemented")
}
