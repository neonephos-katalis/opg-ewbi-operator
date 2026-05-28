package metastore

import (
	"context"

	"github.com/pkg/errors"
)

func (c *k8sClient) AddAvailabilityZone(ctx context.Context, az *PartnerAvailabilityZone) error {
	return errors.Errorf("method not implemented")
}
func (c *k8sClient) GetApplicationInstance(ctx context.Context, federationContextID, id string) (*ApplicationInstance, error) {
	return nil, errors.Errorf("method not implemented")
}

//	func (c *k8sClient) GetApplicationInstanceDetails(ctx context.Context, federationContextID, id string) (*ApplicationInstanceDetails, error) {
//		return nil, errors.Errorf("method not implemented")
//	}
func (c *k8sClient) RemoveAvailabilityZone(ctx context.Context, federationContextID, id string) error {
	return errors.Errorf("method not implemented")
}
