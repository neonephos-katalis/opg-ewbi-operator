package metastore

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/icza/gog"

	"github.com/neonephos-katalis/opg-ewbi-api/api/federation/models"
	camara "github.com/neonephos-katalis/opg-ewbi-api/api/federation/server"
	opgv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/v1beta1"
)

type Application struct {
	*camara.ViewApplication200JSONResponse
	FederationContextId models.FederationContextId
}

type OnboardApplication struct {
	*models.OnboardApplicationJSONBody
	FederationContextId models.FederationContextId
}

func (a *OnboardApplication) MarshalJSON() ([]byte, error) {
	cp := *a.OnboardApplicationJSONBody
	return json.Marshal(&cp)
}

func (a *OnboardApplication) k8sCustomResource(namespace string, opts ...Opt) (*opgv1beta1.Application, error) {
	obj := &opgv1beta1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sCustomResourceNameFromApplicationID(a.FederationContextId, a.AppId),
			Namespace: namespace,
			Labels: map[string]string{
				opgLabel(federationContextIDLabel): a.FederationContextId,
				opgLabel(idLabel):                  a.AppId,
				opgLabel(federationRelation):       host,
			},
		},
		Spec: opgv1beta1.ApplicationSpec{
			AppProviderId:  a.AppProviderId,
			ComponentSpecs: a.componentSpecs(),
			MetaData:       a.metaData(),
			QoSProfile:     a.qosProfile(),
			StatusLink:     a.AppStatusCallbackLink,
		},
	}
	for _, opt := range opts {
		if err := opt(&obj.ObjectMeta); err != nil {
			return nil, err
		}
	}

	return obj, nil
}

func (a *OnboardApplication) artefacts() []string {
	out := make([]string, len(a.AppComponentSpecs))
	for i, componentSpec := range a.AppComponentSpecs {
		out[i] = componentSpec.ArtefactId
	}
	return out
}

func (a *OnboardApplication) componentSpecs() []opgv1beta1.ComponentSpecRef {
	out := make([]opgv1beta1.ComponentSpecRef, len(a.AppComponentSpecs))
	for i, componentSpec := range a.AppComponentSpecs {
		out[i] = opgv1beta1.ComponentSpecRef{
			ArtefactId: componentSpec.ArtefactId,
		}
	}
	return out
}

func (a *OnboardApplication) metaData() opgv1beta1.AppMetaData {
	return opgv1beta1.AppMetaData{
		AccessToken:     a.AppMetaData.AccessToken,
		Name:            a.AppMetaData.AppName,
		MobilitySupport: defaultIfNil(a.AppMetaData.MobilitySupport),
		Version:         a.AppMetaData.Version,
	}
}

func (a *OnboardApplication) qosProfile() opgv1beta1.QoSProfile {
	return opgv1beta1.QoSProfile{
		Provisioning:       defaultIfNil(a.AppQoSProfile.AppProvisioning),
		LatencyConstraints: string(a.AppQoSProfile.LatencyConstraints),
		MultiUserClients:   defaultIfNil((*string)(a.AppQoSProfile.MultiUserClients)),
		UsersPerAppInst:    int64(*a.AppQoSProfile.NoOfUsersPerAppInst),
	}
}

func k8sCustomResourceNameFromApplicationID(federationContextID, appID string) string {
	return fmt.Sprintf("%s-%s", applicationKind, uuidV5Fn(federationContextID+"/"+appID))
}

type appComponentSpec struct {
	// ArtefactId A globally unique identifier associated with the artefact. Originating OP generates this identifier when artefact is submitted over NBI.
	ArtefactId models.ArtefactId `json:"artefactId"`

	// ComponentName Must be a valid RFC 1123 label name. Component name must be unique with an application
	ComponentName *string `json:"componentName,omitempty"`

	// ServiceNameEW Must be a valid RFC 1123 label name. This defines the DNS name via which the component can be accessed via peer components. Access via serviceNameEW is open on all ports. Platform shall not expose serviceNameEW externally outside edge.
	ServiceNameEW *string `json:"serviceNameEW,omitempty"`

	// ServiceNameNB Must be a valid RFC 1123 label name. This defines the DNS name via which the component can be accessed over NBI. Access via serviceNameNB is restricted on specific ports. Platform shall expose component access externally via this DNS name
	ServiceNameNB *string `json:"serviceNameNB,omitempty"`
}

func applicationFromK8sCustomResource(app opgv1beta1.Application) (*Application, error) {
	componentSpec := make(models.AppComponentSpecs, len(app.Spec.ComponentSpecs))
	for i, cs := range app.Spec.ComponentSpecs {
		componentSpec[i] = appComponentSpec{
			ArtefactId: cs.ArtefactId,
		}
	}
	return &Application{
		ViewApplication200JSONResponse: &camara.ViewApplication200JSONResponse{
			AppProviderId:     app.Spec.AppProviderId,
			AppComponentSpecs: componentSpec,
			AppMetaData: models.AppMetaData{
				AccessToken:     app.Spec.MetaData.AccessToken,
				AppName:         app.Spec.MetaData.Name,
				Version:         app.Spec.MetaData.Version,
				MobilitySupport: &app.Spec.MetaData.MobilitySupport,
			},
			AppQoSProfile: models.AppQoSProfile{
				AppProvisioning:     &app.Spec.QoSProfile.Provisioning,
				LatencyConstraints:  models.AppQoSProfileLatencyConstraints(app.Spec.QoSProfile.LatencyConstraints),
				MultiUserClients:    (*models.AppQoSProfileMultiUserClients)(&app.Spec.QoSProfile.MultiUserClients),
				NoOfUsersPerAppInst: (*int)(gog.Ptr(int(app.Spec.QoSProfile.UsersPerAppInst))),
			},
		},
		FederationContextId: app.Labels[opgLabel(federationContextIDLabel)],
	}, nil
}

func isValidApplicationStatus(status string) bool {
	switch opgv1beta1.ApplicationState(status) {
	case opgv1beta1.ApplicationStatePending, opgv1beta1.ApplicationStateOnboarded, opgv1beta1.ApplicationStateDeboarding, opgv1beta1.ApplicationStateFailed, opgv1beta1.ApplicationStateRemoved:
		return true
	}
	return false
}
