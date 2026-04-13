package metastore

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/neonephos-katalis/opg-ewbi-api/api/federation/models"
	camara "github.com/neonephos-katalis/opg-ewbi-api/api/federation/server"
	opgv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/v1beta1"
)

type ApplicationInstanceDetails struct {
	*camara.GetAppInstanceDetails200JSONResponse
}

type ApplicationInstance struct {
	*models.InstallAppJSONBody
	FederationContextId models.FederationContextId `json:"-"`
}

func (d *ApplicationInstance) k8sCustomResource(namespace string, opts ...Opt) (*opgv1beta1.ApplicationInstance, error) {
	obj := &opgv1beta1.ApplicationInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sCustomResourceNameFromApplicationInstance(d.FederationContextId, d.AppInstanceId),
			Namespace: namespace,
			Labels: map[string]string{
				opgLabel(federationContextIDLabel): d.FederationContextId,
				opgLabel(idLabel):                  d.AppInstanceId,
				opgLabel(federationRelation):       host,
			},
		},
		Spec: opgv1beta1.ApplicationInstanceSpec{
			AppProviderId: d.AppProviderId,
			AppId:         d.AppId,
			AppVersion:    d.AppVersion,
			ZoneInfo: opgv1beta1.Zone{
				ZoneId:              d.ZoneInfo.ZoneId,
				FlavourId:           d.ZoneInfo.FlavourId,
				ResourceConsumption: defaultIfNil((*string)(d.ZoneInfo.ResourceConsumption)),
				ResPool:             defaultIfNil(d.ZoneInfo.ResPool),
			},
			CallbBackLink: d.AppInstCallbackLink,
		},
	}
	for _, opt := range opts {
		if err := opt(&obj.ObjectMeta); err != nil {
			return nil, err
		}
	}

	return obj, nil
}

func isValidApplicationInstanceStatus(status string) bool {
	switch opgv1beta1.ApplicationInstanceState(status) {
	case opgv1beta1.ApplicationInstanceStatePending, opgv1beta1.ApplicationInstanceStateReady, opgv1beta1.ApplicationInstanceStateFailed, opgv1beta1.ApplicationInstanceStateTerminating:
		return true
	}
	return false
}

func k8sCustomResourceNameFromApplicationInstance(federationContextID, appID string) string {
	return fmt.Sprintf("%s-%s", applicationInstancePrefix, uuidV5Fn(federationContextID+"/"+appID))
}

func applicationInstanceFromK8sCustomResource(appInstanceID string, appInstance opgv1beta1.ApplicationInstance) (*ApplicationInstanceDetails, error) {
	return &ApplicationInstanceDetails{}, nil
}
