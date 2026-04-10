package metastore

import (
	"context"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"

	opgv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/v1beta1"
)

type ClientCredentials struct {
	ClientID string
}

func (c *k8sClient) GetClientCredentials(ctx context.Context, ClientID string) (ClientCredentials, error) {
	obj, err := c.searchKubernetesObject(&opgv1beta1.FederationList{}, labels.Set{
		opgLabel(clientIDLabel): ClientID,
	})
	if err != nil {
		return ClientCredentials{}, errors.Wrapf(ErrNotFound, "unkown client ID")
	}
	res, ok := obj.(*opgv1beta1.Federation)
	if !ok {
		log.Errorf("failed to get federation with %s label '%s': type missmatch, expected %T got %T", clientIDLabel, ClientID, &opgv1beta1.Federation{}, obj)
		return ClientCredentials{}, ErrInternal
	}
	return ClientCredentials{
		ClientID: res.Spec.GuestPartnerCredentials.ClientId,
	}, nil
}
