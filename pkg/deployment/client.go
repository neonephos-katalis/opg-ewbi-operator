package deployment

import (
	"context"
	"errors"

	k8scl "sigs.k8s.io/controller-runtime/pkg/client"

	v1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/metastore"
	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/uuid"
)

var _ Client = &client{}

type Client interface {
	Install(ctx context.Context, app *InstallDeployment) (*v1beta1.ApplicationDeployment, string, error)
	Uninstall(ctx context.Context, federationContextID, appId, appInstanceId string) error
}

func NewClient(k8sClient k8scl.Client, namespace string) *client {
	return &client{
		appMetaClient: metastore.NewK8sClient(k8sClient, namespace),
	}
}

type client struct {
	appMetaClient metastore.Client
}

func (c *client) Install(ctx context.Context, dep *InstallDeployment) (*v1beta1.ApplicationDeployment, string, error) {
	var obj *v1beta1.ApplicationDeployment
	var err error
	if obj, err = c.appMetaClient.AddApplicationDeployment(ctx, &metastore.ApplicationInstance{
		InstallAppJSONBody:  dep.InstallAppJSONBody,
		FederationContextId: dep.FederationContextID,
	}); err != nil {
		return nil, "", err
	}

	return obj, uuid.V5(dep.AppId + dep.AppProviderId), nil
}

func (c *client) Uninstall(ctx context.Context, federationContextID, appId, appInstanceId string) error {
	if err := c.appMetaClient.RemoveApplicationDeployment(ctx, federationContextID, appId, appInstanceId); err != nil && !errors.Is(err, metastore.ErrNotFound) {
		return err
	}

	return nil
}
