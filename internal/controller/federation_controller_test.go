/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"testing"
	"time"

	"github.com/nbycomp/neonephos-opg-ewbi-operator/api/v1beta1"
	"github.com/nbycomp/neonephos-opg-ewbi-operator/internal/indexer"
	"github.com/nbycomp/neonephos-opg-ewbi-operator/internal/opg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	testNamespace = "opg"

	testFederationName        = "federation001"
	testFederationCountryCode = "JN"
	testFederationMCC         = "999"
	testFederationMNC         = "99"
	testFederationClientId    = "08151875-6c4e-40a7-a9ff-dbe85cecfc71"
	testFederationTokenUrl    = "https://company.com/federation/token"
	testFederationLink        = "https://company.com/federation"

	testFederationExternalId = "00710df0-ede0-4f0a-bde2-1601aac93e7a"
	testFederationContextId  = "00000000-0000-0000-0000-000000000000"
	testFederationContextId2 = "00000000-0000-0000-0000-222222222222"

	defaultTestFederationRelation = v1beta1.FederationRelationGuest
	testFederationDomain          = "https://oscar.envs.nearbycomputing.com"

	testFederationUrlBasePath = "/apicatalog/unauthorized/operatorplatform/federation/v1"
	testFederationUrl         = testFederationDomain + testFederationUrlBasePath
)

func TestFederationReconciler(t *testing.T) {

	type fields struct {
		resources          []client.Object
		mockOpgFederations []*v1beta1.Federation
	}
	type args struct {
		req ctrl.Request
	}
	type response struct {
		wantResult         ctrl.Result
		wantReconcileErr   bool
		wantGetErr         func(err error) bool
		wantStatusPhase    v1beta1.FederationPhase
		wantStatusState    v1beta1.FederationState
		wantFinalizer      string
		wantOfferedAZs     []v1beta1.ZoneDetails
		wantAcceptedAZs    []string
		wantAPIFederations []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		resp   response
	}{
		{
			name: "New Federation without Finalizer will get it and return",
			fields: fields{
				resources: []client.Object{makeTestFederation(testFederationName)},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testFederationName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  "",
				wantFinalizer:    v1beta1.FederationFinalizer,
			},
		},
		{
			name: "A Host Federation is ignored, phase is set to Ready",
			fields: fields{
				resources: []client.Object{
					makeTestFederation(
						testFederationName,
						federationWithFinalizer(),
						federationWithFederationRelation(v1beta1.FederationRelationHost),
					),
				},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testFederationName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  v1beta1.FederationPhaseReady,
				wantFinalizer:    v1beta1.FederationFinalizer},
		},
		{
			name: "A Guest Federation requests OfferedAvailabilityZones",
			fields: fields{
				resources: []client.Object{makeTestFederation(testFederationName, federationWithFinalizer())},
				mockOpgFederations: []*v1beta1.Federation{
					makeTestFederation(testFederationName,
						federationWithAvailableAZ(testAZName),
						federationWithAvailableAZ("secondAZ"),
						withFederationContextId(testFederationExternalId),
					),
				},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testFederationName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:         ctrl.Result{Requeue: false},
				wantReconcileErr:   false,
				wantStatusPhase:    "", // Phase is not updated
				wantStatusState:    v1beta1.FederationStateAvailable,
				wantFinalizer:      v1beta1.FederationFinalizer,
				wantOfferedAZs:     []v1beta1.ZoneDetails{{ZoneId: testAZName}},
				wantAcceptedAZs:    nil, // none yet, these would be assigned next reconcile iter
				wantAPIFederations: []string{testFederationExternalId},
			},
		},
		{
			name: "A Guest Federation with OfferedAvailabilityZones will accept the first one",
			fields: fields{
				resources: []client.Object{
					makeTestFederation(testFederationName,
						federationWithFinalizer(),
						federationWithAvailableAZ(testAZName),
						federationWithFederationState(v1beta1.FederationStateAvailable),
					),
				},
				mockOpgFederations: []*v1beta1.Federation{
					makeTestFederation(
						testFederationName,
						federationWithAvailableAZ(testAZName),
						withFederationContextId(testFederationExternalId),
					),
				},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testFederationName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:         ctrl.Result{Requeue: false},
				wantReconcileErr:   false,
				wantStatusPhase:    "", // Phase is not updated
				wantStatusState:    v1beta1.FederationStateAvailable,
				wantFinalizer:      v1beta1.FederationFinalizer,
				wantOfferedAZs:     []v1beta1.ZoneDetails{{ZoneId: testAZName}},
				wantAcceptedAZs:    []string{testAZName},
				wantAPIFederations: []string{testFederationExternalId},
			},
		},
		{
			name: "Delete Federation with DeletionTimestamp",
			fields: fields{
				resources: []client.Object{
					makeTestFederation(testFederationName,
						federationWithFinalizer(),
						federationWithAvailableAZ(testAZName),
						federationWithFederationState(v1beta1.FederationStateAvailable),
						federationDeletedAt(time.Now()),
						withFederationContextId(testFederationExternalId),
					),
				},
				mockOpgFederations: []*v1beta1.Federation{
					makeTestFederation(testFederationName,
						federationWithAvailableAZ(testAZName),
						withFederationContextId(testFederationExternalId)),
				},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testFederationName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:         ctrl.Result{Requeue: false},
				wantReconcileErr:   false,
				wantGetErr:         errors.IsNotFound,
				wantAPIFederations: []string{}, // empty it should have been deleted
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			apiObjs := &ApiObjects{
				Federations: tt.fields.mockOpgFederations}
			cl, opgcmap, mockedOpgAPI, sch := prepareEnv(tt.fields.resources, apiObjs)
			r := makeTestFederationReconciler(cl, sch, opgcmap)

			gotResult, err := r.Reconcile(ctx, tt.args.req)

			if tt.resp.wantReconcileErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.resp.wantResult, gotResult)

			for _, apiFed := range tt.resp.wantAPIFederations {
				assert.Contains(t, mockedOpgAPI.Federations, apiFed)
			}

			var reqFeder v1beta1.Federation
			err = r.Client.Get(ctx, tt.args.req.NamespacedName, &reqFeder)
			if tt.resp.wantGetErr != nil {
				assert.True(t, tt.resp.wantGetErr(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.resp.wantStatusPhase, reqFeder.Status.Phase)
			assert.Equal(t, tt.resp.wantStatusState, reqFeder.Status.State)
			assert.Contains(t, reqFeder.Finalizers, tt.resp.wantFinalizer)
			assert.Equal(t, tt.resp.wantOfferedAZs, reqFeder.Status.OfferedAvailabilityZones)
			assert.Equal(t, tt.resp.wantAcceptedAZs, reqFeder.Spec.AcceptedAvailabilityZones)

		})
	}
}

type federationOpt func(*v1beta1.Federation)

func federationDeletedAt(now time.Time) federationOpt {
	return func(a *v1beta1.Federation) {
		wrapped := metav1.NewTime(now)
		a.ObjectMeta.DeletionTimestamp = &wrapped
		a.Finalizers = []string{v1beta1.FederationFinalizer}
	}
}

//nolint:unparam
func federationWithAvailableAZ(azId string) federationOpt {
	return func(f *v1beta1.Federation) {
		if f.Status.OfferedAvailabilityZones == nil {
			f.Status.OfferedAvailabilityZones = []v1beta1.ZoneDetails{}
		}
		f.Status.OfferedAvailabilityZones = append(f.Status.OfferedAvailabilityZones, v1beta1.ZoneDetails{ZoneId: azId})
	}
}

func federationWithFederationRelation(rel v1beta1.FederationRelation) federationOpt {
	return func(f *v1beta1.Federation) {
		f.Labels[v1beta1.FederationRelationLabel] = string(rel)
	}
}

func federationWithFederationState(state v1beta1.FederationState) federationOpt {
	return func(f *v1beta1.Federation) {
		f.Status.State = state
	}
}

func federationWithFinalizer() federationOpt {
	return func(f *v1beta1.Federation) {
		controllerutil.AddFinalizer(f, v1beta1.FederationFinalizer)
	}
}

func makeTestFederation(name string, opts ...federationOpt) *v1beta1.Federation {
	a := &v1beta1.Federation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
			Labels: map[string]string{
				// the FederationContextIdLabel points to itself
				// opgewbiv1beta1.FederationContextIdLabel: testFederationExternalId,
				v1beta1.ExternalIdLabel:         testFederationExternalId,
				v1beta1.FederationRelationLabel: string(defaultTestFederationRelation),
				v1beta1.FederationGuestUrlLabel: testFederationUrl,
			},
		},
		Spec: v1beta1.FederationSpec{
			InitialDate: metav1.NewTime(time.Now()),
			OriginOP: v1beta1.Origin{
				CountryCode:       testFederationCountryCode,
				FixedNetworkCodes: []string{"123", "456"},
				MobileNetworkCodes: v1beta1.MobileNetworkCodes{
					MCC: testFederationMCC,
					MNC: []string{testFederationMNC},
				},
			},
			Partner: v1beta1.Partner{
				CallbackCredentials: v1beta1.FederationCredentials{
					ClientId: testFederationClientId,
					TokenUrl: testFederationTokenUrl,
				},
				StatusLink: testFederationLink,
			},
			AcceptedAvailabilityZones: []string{},
		},
	}
	for _, o := range opts {
		o(a)
	}

	return a
}

func makeTestFederationReconciler(
	client client.Client,
	sch *runtime.Scheme,
	opgClients opg.OPGClientsMapInterface,
) *FederationReconciler {
	r := &FederationReconciler{
		Client:                 client,
		Scheme:                 sch,
		OPGClientsMapInterface: opgClients,
	}
	return r
}

func makeTestReconcilerClient(sch *runtime.Scheme, resObjs, subresObjs []client.Object,
	runtimeObj []runtime.Object) client.Client {

	client := fake.NewClientBuilder().WithScheme(sch).WithIndex(&v1beta1.Federation{},
		v1beta1.FederationStatusContextIDField, indexer.FedContextIdIndexer)

	if len(resObjs) > 0 {
		client = client.WithObjects(resObjs...)
	}
	if len(subresObjs) > 0 {
		client = client.WithStatusSubresource(subresObjs...)
	}
	if len(runtimeObj) > 0 {
		client = client.WithRuntimeObjects(runtimeObj...)
	}
	return client.Build()
}

type SchemeOpt func(*runtime.Scheme) error

func makeTestReconcilerScheme(sOpts ...SchemeOpt) *runtime.Scheme {
	s := scheme.Scheme
	for _, opt := range sOpts {
		_ = opt(s)
	}

	return s
}

func withFederationContextId(fcid string) federationOpt {
	return func(f *v1beta1.Federation) {
		f.Status.FederationContextId = fcid
	}
}

func withFederationContextIdAsLabel(fcid string) federationOpt {
	return func(f *v1beta1.Federation) {
		f.Labels[v1beta1.FederationContextIdLabel] = fcid
	}
}
