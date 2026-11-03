package controller

import (
	"context"
	"testing"
	"time"

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	k8s "github.com/neonephos-katalis/opg-ewbi-operator/internal/k8s"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
	rest "github.com/neonephos-katalis/opg-ewbi-operator/internal/rest"
	"github.com/neonephos-katalis/opg-ewbi-operator/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	testRcmName    = "rcm001"
	testRcmZoneId  = "zone-001"
	testRcmMonType = "edge_resource"
)

func TestConsumptionMonitoringReconciler(t *testing.T) {
	federK8s := makeTestFederation(testFederationName,
		withFederationContextId(testFederationContextId),
		rcmFederationWithTechnology(v1beta1.FederationTechnologyK8s),
		rcmFederationWithState(v1beta1.FederationStateAvailable),
	)
	federRest := makeTestFederation(testFederationName,
		withFederationContextId(testFederationContextId),
		rcmFederationWithTechnology(v1beta1.FederationTechnologyRest),
		rcmFederationWithState(v1beta1.FederationStateAvailable),
	)
	federLocked := makeTestFederation(testFederationName,
		withFederationContextId(testFederationContextId),
		rcmFederationWithTechnology(v1beta1.FederationTechnologyK8s),
		rcmFederationWithState(v1beta1.FederationStateLocked),
	)
	wantLabel := "rcm-" + uuid.V5(testFederationContextId+testRcmMonType+testRcmZoneId)

	type response struct {
		wantReconcileErr bool
		wantGetErr       func(err error) bool
		wantStatusState  v1beta1.ConsumptionMonitoringState
		wantFinalizer    bool
		wantLabel        string
	}
	tests := []struct {
		name      string
		resources []client.Object
		resp      response
	}{
		{
			name:      "New RCM without Finalizer will get it and return",
			resources: []client.Object{federK8s, makeTestRcm(v1beta1.FederationRelationHost)},
			resp: response{
				wantStatusState: "",
				wantFinalizer:   true,
			},
		},
		{
			name:      "New Host RCM is set to Pending and labeled",
			resources: []client.Object{federK8s, makeTestRcm(v1beta1.FederationRelationHost, rcmWithFinalizer())},
			resp: response{
				wantStatusState: v1beta1.ConsumptionMonitoringStatePending,
				wantFinalizer:   true,
				wantLabel:       wantLabel,
			},
		},
		{
			name: "Existing Host RCM on K8s federation keeps its state",
			resources: []client.Object{federK8s, makeTestRcm(v1beta1.FederationRelationHost,
				rcmWithFinalizer(),
				rcmWithState(v1beta1.ConsumptionMonitoringStateAvailable),
			)},
			resp: response{
				wantStatusState: v1beta1.ConsumptionMonitoringStateAvailable,
				wantFinalizer:   true,
			},
		},
		{
			name:      "New Guest RCM on REST federation is set to Pending and labeled",
			resources: []client.Object{federRest, makeTestRcm(v1beta1.FederationRelationGuest, rcmWithFinalizer())},
			resp: response{
				wantStatusState: v1beta1.ConsumptionMonitoringStatePending,
				wantFinalizer:   true,
				wantLabel:       wantLabel,
			},
		},
		{
			name: "Existing Guest RCM on REST federation keeps its state",
			resources: []client.Object{federRest, makeTestRcm(v1beta1.FederationRelationGuest,
				rcmWithFinalizer(),
				rcmWithState(v1beta1.ConsumptionMonitoringStatePending),
			)},
			resp: response{
				wantStatusState: v1beta1.ConsumptionMonitoringStatePending,
				wantFinalizer:   true,
			},
		},
		{
			name: "Deleted Guest RCM on REST federation has its finalizer removed",
			resources: []client.Object{federRest, makeTestRcm(v1beta1.FederationRelationGuest,
				rcmWithState(v1beta1.ConsumptionMonitoringStatePending),
				rcmWithDeletedAt(time.Now()),
			)},
			resp: response{
				wantGetErr: errors.IsNotFound,
			},
		},
		{
			name: "Deleted Host RCM has its finalizer removed",
			resources: []client.Object{federK8s, makeTestRcm(v1beta1.FederationRelationHost,
				rcmWithState(v1beta1.ConsumptionMonitoringStatePending),
				rcmWithDeletedAt(time.Now()),
			)},
			resp: response{
				wantGetErr: errors.IsNotFound,
			},
		},
		{
			name:      "RCM without parent federation is set to Failed",
			resources: []client.Object{makeTestRcm(v1beta1.FederationRelationHost, rcmWithFinalizer())},
			resp: response{
				wantReconcileErr: true,
				wantStatusState:  v1beta1.ConsumptionMonitoringStateFailed,
				wantFinalizer:    true,
			},
		},
		{
			name:      "RCM with locked federation is not reconciled",
			resources: []client.Object{federLocked, makeTestRcm(v1beta1.FederationRelationHost, rcmWithFinalizer())},
			resp: response{
				wantStatusState: "",
				wantFinalizer:   true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			cl, opgcmap, _, sch := prepareEnv(tt.resources, &ApiObjects{})
			r := makeTestRcmReconciler(cl, sch, opgcmap)
			req := ctrl.Request{NamespacedName: types.NamespacedName{Name: testRcmName, Namespace: testNamespace}}

			gotResult, err := r.Reconcile(ctx, req)

			if tt.resp.wantReconcileErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, ctrl.Result{}, gotResult)

			var got v1beta1.ConsumptionMonitoring
			err = r.Client.Get(ctx, req.NamespacedName, &got)
			if tt.resp.wantGetErr != nil {
				assert.True(t, tt.resp.wantGetErr(err))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.resp.wantStatusState, got.Status.State)
			assert.Equal(t, tt.resp.wantFinalizer, controllerutil.ContainsFinalizer(&got, v1beta1.ConsumptionMonitoringFinalizer))
			if tt.resp.wantLabel != "" {
				assert.Equal(t, tt.resp.wantLabel, got.Labels[v1beta1.ResourceIdLabel])
			}
		})
	}
}

func rcmFederationWithTechnology(tech v1beta1.FederationTechnology) federationOpt {
	return func(f *v1beta1.Federation) {
		f.Spec.FederationData.TechnologyType = string(tech)
	}
}

func rcmFederationWithState(state v1beta1.FederationState) federationOpt {
	return func(f *v1beta1.Federation) {
		f.Status.State = state
	}
}

type rcmOpt func(*v1beta1.ConsumptionMonitoring)

func rcmWithFinalizer() rcmOpt {
	return func(r *v1beta1.ConsumptionMonitoring) {
		controllerutil.AddFinalizer(r, v1beta1.ConsumptionMonitoringFinalizer)
	}
}

func rcmWithDeletedAt(now time.Time) rcmOpt {
	return func(r *v1beta1.ConsumptionMonitoring) {
		wrapped := metav1.NewTime(now)
		r.DeletionTimestamp = &wrapped
		r.Finalizers = []string{v1beta1.ConsumptionMonitoringFinalizer}
	}
}

func rcmWithState(state v1beta1.ConsumptionMonitoringState) rcmOpt {
	return func(r *v1beta1.ConsumptionMonitoring) {
		r.Status.State = state
	}
}

func makeTestRcm(relation v1beta1.FederationRelation, opts ...rcmOpt) *v1beta1.ConsumptionMonitoring {
	r := &v1beta1.ConsumptionMonitoring{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRcmName,
			Namespace: testNamespace,
		},
		Spec: v1beta1.ConsumptionMonitoringSpec{
			RelationType:        string(relation),
			FederationContextId: testFederationContextId,
			ZoneId:              testRcmZoneId,
			MonType:             testRcmMonType,
		},
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

func makeTestRcmReconciler(
	client client.Client,
	sch *runtime.Scheme,
	opgClients opg.OPGClientsMapInterface,
) *ConsumptionMonitoringReconciler {
	return &ConsumptionMonitoringReconciler{
		Client:                 client,
		Scheme:                 sch,
		OPGClientsMapInterface: opgClients,
		K8sClient: &k8s.ConsumptionMonitoringReconciler{
			Client:                 client,
			Scheme:                 sch,
			OPGClientsMapInterface: opgClients,
		},
		RestClient: &rest.ConsumptionMonitoringReconciler{
			Client:                 client,
			Scheme:                 sch,
			OPGClientsMapInterface: opgClients,
		},
	}
}
