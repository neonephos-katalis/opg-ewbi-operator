package deployment

import (
	"context"
	"errors"

	k8scl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/neonephos-katalis/opg-ewbi-api/pkg/metastore"
	opgv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/v1beta1"
)

var _ Client = &client{}

type Client interface {
	Install(ctx context.Context, app *InstallDeployment) (*opgv1beta1.ApplicationInstance, string, error)
	Uninstall(ctx context.Context, federationContextID, id string) error
}

func NewClient(k8sClient k8scl.Client, namespace string) *client {
	return &client{
		appMetaClient: metastore.NewK8sClient(k8sClient, namespace),
	}
}

type client struct {
	appMetaClient metastore.Client
}

func (c *client) Install(ctx context.Context, dep *InstallDeployment) (*opgv1beta1.ApplicationInstance, string, error) {
	var obj *opgv1beta1.ApplicationInstance
	var err error
	if obj, err = c.appMetaClient.AddApplicationInstance(ctx, &metastore.ApplicationInstance{
		InstallAppJSONBody:  dep.InstallAppJSONBody,
		FederationContextId: dep.FederationContextID,
	}); err != nil {
		return nil, "", err
	}

	return obj, dep.AppInstanceId, nil
}

func (c *client) Uninstall(ctx context.Context, federationContextID, id string) error {
	if err := c.appMetaClient.RemoveApplicationInstance(ctx, federationContextID, id); err != nil && !errors.Is(err, metastore.ErrNotFound) {
		return err
	}

	return nil
}
