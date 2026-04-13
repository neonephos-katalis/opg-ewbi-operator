package mock

import (
	"context"

	"github.com/neonephos-katalis/opg-ewbi-api/pkg/metastore"
)

// Mock MetaStoreClient
type FakeMetaStoreClient struct {
	metastore.Client
	ListAvailabilityZonesFunc func() ([]*metastore.PartnerAvailabilityZone, error)
	CreateFederationFunc      func(federation *metastore.Federation) (*metastore.Federation, error)
	GetClientCredentialsFunc  func(clientID string) (metastore.ClientCredentials, error)
}

func (f *FakeMetaStoreClient) ListAvailabilityZones(ctx context.Context) ([]*metastore.PartnerAvailabilityZone, error) {
	return f.ListAvailabilityZonesFunc()
}

func (f *FakeMetaStoreClient) CreateFederation(ctx context.Context, federation *metastore.Federation) (*metastore.Federation, error) {
	return f.CreateFederationFunc(federation)
}

func (f *FakeMetaStoreClient) GetClientCredentials(ctx context.Context, clientID string) (metastore.ClientCredentials, error) {
	return f.GetClientCredentialsFunc(clientID)
}
