package metastore

import (
	"context"

	"github.com/pkg/errors"
)

func (c *k8sClient) AddAvailabilityZone(ctx context.Context, az *PartnerAvailabilityZone) error {
	return errors.Errorf("method not implemented")
}

func (c *k8sClient) RemoveAvailabilityZone(ctx context.Context, federationContextID, id string) error {
	return errors.Errorf("method not implemented")
}
