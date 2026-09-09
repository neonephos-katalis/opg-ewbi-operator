package metastore

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scli "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/icza/gog"
	"github.com/pkg/errors"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/models"
	camara "github.com/neonephos-katalis/opg-ewbi-operator/api/ewbi/server"
	v1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	uu "github.com/neonephos-katalis/opg-ewbi-operator/pkg/uuid"
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

func (a *OnboardApplication) k8sCustomResource(namespace string, opts ...Opt) (*v1beta1.ApplicationOnboarding, error) {
	appId := "apponboard-" + uu.V5(a.FederationContextId+a.AppProviderId+string(a.AppId[:]))
	obj := &v1beta1.ApplicationOnboarding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appId,
			Namespace: namespace,
		},
		Spec: v1beta1.ApplicationOnboardingSpec{
			FederationContextId: a.FederationContextId,
			RelationType:        string(v1beta1.FederationRelationHost),
			AppInfo: &v1beta1.AppInfo{
				AppId:                 a.AppId,
				AppProviderId:         a.AppProviderId,
				AppComponentSpecs:     a.componentSpecs(),
				AppMetaData:           a.metaData(),
				AppQoSProfile:         a.qosProfile(),
				AppStatusCallbackLink: string(*a.AppStatusCallbackLink),
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

func (c *k8sClient) searchApplication(ctx context.Context, federationContextId string, appId string, role string) (*v1beta1.ApplicationOnboarding, error) {
	var appList v1beta1.ApplicationOnboardingList
	if err := c.kubernetes.List(ctx, &appList, &k8scli.ListOptions{Namespace: c.getNamespace()}); err != nil {
		return nil, err
	}
	if len(appList.Items) == 0 {
		return nil, errors.Errorf("No applications found for federationContextId: %s and appId: %s and role: %s. Empty list", federationContextId, appId, role)
	}
	for i := range appList.Items {
		app := appList.Items[i]
		if app.Spec.AppInfo.AppId == appId && app.Spec.FederationContextId == federationContextId && app.Spec.RelationType == role {
			return &app, nil
		}
	}
	return nil, errors.Errorf("Application not found for federationContextId: %s and appId: %s and role: %s", federationContextId, appId, role)
}

func (a *OnboardApplication) artefacts() []string {
	out := make([]string, len(a.AppComponentSpecs))
	for i, componentSpec := range a.AppComponentSpecs {
		// Usa .String() per formattare l'array UUID come stringa
		out[i] = componentSpec.ArtefactId
	}
	return out
}

func (a *OnboardApplication) componentSpecs() []v1beta1.AppComponentSpec {
	out := make([]v1beta1.AppComponentSpec, len(a.AppComponentSpecs))
	for i, componentSpec := range a.AppComponentSpecs {
		out[i] = v1beta1.AppComponentSpec{
			ArtefactId: componentSpec.ArtefactId,
		}
	}
	return out
}

func (a *OnboardApplication) metaData() *v1beta1.AppMetaData {
	return &v1beta1.AppMetaData{
		AccessToken:     a.AppMetaData.AccessToken,
		AppName:         a.AppMetaData.AppName,
		MobilitySupport: defaultIfNil(a.AppMetaData.MobilitySupport),
		Version:         a.AppMetaData.Version,
	}
}

func (a *OnboardApplication) qosProfile() *v1beta1.AppQoSProfile {
	return &v1beta1.AppQoSProfile{
		AppProvisioning:     defaultIfNil(a.AppQoSProfile.AppProvisioning),
		LatencyConstraints:  string(a.AppQoSProfile.LatencyConstraints),
		MultiUserClients:    defaultIfNil((*string)(a.AppQoSProfile.MultiUserClients)),
		NoOfUsersPerAppInst: int(*a.AppQoSProfile.NoOfUsersPerAppInst),
	}
}

func k8sCustomResourceNameFromApplicationID(federationContextID, appID string) string {
	return fmt.Sprintf("%s-%s", applicationOnboardingKind, uuidV5Fn(federationContextID+"/"+appID))
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

func applicationFromK8sCustomResource(app v1beta1.ApplicationOnboarding) (*Application, error) {
	componentSpec := make(models.AppComponentSpecs, len(app.Spec.AppInfo.AppComponentSpecs))
	for i, cs := range app.Spec.AppInfo.AppComponentSpecs {
		componentSpec[i] = appComponentSpec{
			ArtefactId: cs.ArtefactId,
		}
	}
	return &Application{
		ViewApplication200JSONResponse: &camara.ViewApplication200JSONResponse{
			AppProviderId:     app.Spec.AppInfo.AppProviderId,
			AppComponentSpecs: componentSpec,
			AppMetaData: models.AppMetaData{
				AccessToken:     app.Spec.AppInfo.AppMetaData.AccessToken,
				AppName:         app.Spec.AppInfo.AppMetaData.AppName,
				Version:         app.Spec.AppInfo.AppMetaData.Version,
				MobilitySupport: &app.Spec.AppInfo.AppMetaData.MobilitySupport,
			},
			AppQoSProfile: models.AppQoSProfile{
				AppProvisioning:     &app.Spec.AppInfo.AppQoSProfile.AppProvisioning,
				LatencyConstraints:  models.LatencyConstraints(app.Spec.AppInfo.AppQoSProfile.LatencyConstraints),
				MultiUserClients:    (*models.MultiUserClients)(&app.Spec.AppInfo.AppQoSProfile.MultiUserClients),
				NoOfUsersPerAppInst: (*int)(gog.Ptr(int(app.Spec.AppInfo.AppQoSProfile.NoOfUsersPerAppInst))),
			},
		},
		FederationContextId: app.Labels[opgLabel(federationContextIDLabel)],
	}, nil
}

// func isValidApplicationStatus(status string) bool {
// 	switch v1beta1.ApplicationOnboardingState(status) {
// 	case v1beta1.ApplicationOnboardingStatePending, v1beta1.ApplicationOnboardingStateOnboarded, v1beta1.ApplicationOnboardingStateDeboarding, v1beta1.ApplicationOnboardingStateFailed, v1beta1.ApplicationOnboardingStateRemoved:
// 		return true
// 	}
// 	return false
// }
