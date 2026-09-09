package metastore

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scli "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pkg/errors"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	camara "github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/server"
	v1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	uu "github.com/neonephos-katalis/opg-ewbi-operator/pkg/uuid"
)

type ApplicationInstanceDetails struct {
	*camara.GetAppInstanceDetails200JSONResponse
}

type ApplicationInstance struct {
	*models.InstallAppJSONBody
	FederationContextId models.FederationContextId `json:"-"`
}

// func isValidApplicationDeploymentStatus(status string) bool {
// 	switch v1beta1.ApplicationDeploymentState(status) {
// 	case v1beta1.ApplicationDeploymentStatePending, v1beta1.ApplicationDeploymentStateReady, v1beta1.ApplicationDeploymentStateFailed, v1beta1.ApplicationDeploymentStateTerminating:
// 		return true
// 	}
// 	return false
// }

func (d *ApplicationInstance) k8sCustomResource(namespace string, opts ...Opt) (*v1beta1.ApplicationDeployment, error) {
	appId := "appdeploy-" + uu.V5(d.FederationContextId+string(d.AppId[:])+*d.AppInstanceId)
	obj := &v1beta1.ApplicationDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appId,
			Namespace: namespace,
		},
		Spec: v1beta1.ApplicationDeploymentSpec{
			FederationContextId: d.FederationContextId,
			RelationType:        string(v1beta1.FederationRelationHost),
			AppProviderId:       d.AppProviderId,
			AppId:               d.AppId,
			AppInstanceId:       *d.AppInstanceId,
			ZoneId:              d.ZoneInfo.ZoneId,
			AppDetails: &v1beta1.AppDetails{
				AppVersion: d.AppVersion,
				ZoneInfo: &v1beta1.ZoneInfo{
					FlavourId:           d.ZoneInfo.FlavourId,
					ResourceConsumption: defaultIfNil((*string)(d.ZoneInfo.ResourceConsumption)),
					ResPool:             defaultIfNil(d.ZoneInfo.ResPool),
				},
				AppInstCallbackLink: d.AppInstCallbackLink,
			},
		},
	}
	for _, opt := range opts {
		if err := opt(&obj.ObjectMeta); err != nil {
			return nil, err
		}
	}

	return obj, nil
}

func applicationDeploymentFromK8sCustomResource(appInstanceID string, appInstance v1beta1.ApplicationDeployment) (*ApplicationInstanceDetails, error) {
	return &ApplicationInstanceDetails{}, nil
}

func (c *k8sClient) searchApplicationDeployment(ctx context.Context, federationContextId string, appInstanceId string, role string) (*v1beta1.ApplicationDeployment, error) {
	var depList v1beta1.ApplicationDeploymentList
	if err := c.kubernetes.List(ctx, &depList, &k8scli.ListOptions{Namespace: c.getNamespace()}); err != nil {
		return nil, err
	}
	if len(depList.Items) == 0 {
		return nil, errors.Errorf("ApplicationDeployment not found for federationContextId: %s and appInstanceId: %s and role: %s", federationContextId, appInstanceId, role)
	}
	for i := range depList.Items {
		dep := depList.Items[i]
		if dep.Spec.AppInstanceId == appInstanceId && dep.Spec.FederationContextId == federationContextId && dep.Spec.RelationType == role {
			return &dep, nil
		}
	}
	return nil, errors.Errorf("ApplicationDeployment not found for federationContextId: %s and appInstanceId: %s and role: %s", federationContextId, appInstanceId, role)
}
