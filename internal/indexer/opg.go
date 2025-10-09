package indexer

import (
	"context"

	opgewbiv1beta1 "github.com/nbycomp/neonephos-opg-ewbi-operator/api/v1beta1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetFederationIndexers(ctx context.Context, mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(ctx, &opgewbiv1beta1.Federation{},
		opgewbiv1beta1.FederationStatusContextIDField, FedContextIdIndexer)
}

func FedContextIdIndexer(rawObj client.Object) []string {
	f := rawObj.(*opgewbiv1beta1.Federation)
	if f.Status.FederationContextId == "" {
		v, ok := f.Labels[opgewbiv1beta1.FederationContextIdLabel]
		if !ok || v == "" {
			return nil
		}
		return []string{f.Labels[opgewbiv1beta1.FederationContextIdLabel]}
	}
	return []string{f.Status.FederationContextId}
}
