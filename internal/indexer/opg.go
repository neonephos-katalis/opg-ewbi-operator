package indexer

import (
	"context"

	opgewbiv1beta1 "github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetFederationIndexers(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(ctx, &opgewbiv1beta1.Federation{},
		opgewbiv1beta1.FederationStatusContextIDField, FedContextIdIndexer)
}

func FedContextIdIndexer(rawObj client.Object) []string {
	return []string{rawObj.(*opgewbiv1beta1.Federation).Status.FederationContextId}
}
