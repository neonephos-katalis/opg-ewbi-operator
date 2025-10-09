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
	"github.com/nbycomp/neonephos-opg-ewbi-operator/internal/opg"
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

	// Application
	testAppName            = "app001"
	testAppExternalId      = "app-00000000-0000-0000-0000-000000000001"
	testAppProvider        = "nearbycomputing"
	testAppMetaDataName    = "final-user-application-name"
	testAppMetaDataVersion = "37.6"
)

func TestApplicationReconciler(t *testing.T) {
	feder := makeTestFederation(testFederationName, withFederationContextId(testFederationContextId))
	file := makeTestFile(testFederationContextId)

	type fields struct {
		resources          []client.Object
		mockOpgFederations []*v1beta1.Federation
		mockOpgApps        []*v1beta1.Application
	}
	type args struct {
		req ctrl.Request
	}
	type response struct {
		wantResult       ctrl.Result
		wantReconcileErr bool
		wantGetErr       func(err error) bool
		wantStatusPhase  v1beta1.ApplicationPhase
		wantStatusState  v1beta1.ApplicationState
		wantFinalizer    string
		wantAPIApps      []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		resp   response
	}{
		{
			name: "New Application without Finalizer will get it and return",
			fields: fields{
				resources: []client.Object{feder, file, makeTestApplication(testFederationContextId)},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testAppName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  "",
				wantFinalizer:    v1beta1.AppFinalizer,
			},
		},
		{
			name: "A Host Application is ignored, phase is set to Ready",
			fields: fields{
				resources: []client.Object{feder, file, makeTestApplication(testFederationContextId, appWithFinalizer())},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testAppName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  v1beta1.ApplicationPhaseReady,
				wantFinalizer:    v1beta1.AppFinalizer,
			},
		},
		{
			name: "A New Guest Application is created at federation partner Operator",
			fields: fields{
				resources:          []client.Object{feder, file, makeTestApplication(testFederationContextId, appWithFinalizer())},
				mockOpgFederations: []*v1beta1.Federation{feder},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testAppName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  v1beta1.ApplicationPhaseReady,
				wantFinalizer:    v1beta1.AppFinalizer,
				wantAPIApps:      []string{testAppExternalId},
			},
		},
		{
			name: "An existing Guest Application is synced at federation partner if already exists",
			fields: fields{
				resources: []client.Object{
					feder,
					file,
					makeTestApplication(testFederationContextId, appWithFinalizer(), appWithPhase(v1beta1.ApplicationPhaseReady))},
				mockOpgFederations: []*v1beta1.Federation{feder},
				mockOpgApps:        []*v1beta1.Application{makeTestApplication(testFederationContextId)},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Name: testAppName, Namespace: testNamespace}},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  v1beta1.ApplicationPhaseReady,
				wantFinalizer:    v1beta1.AppFinalizer,
				wantAPIApps:      []string{testAppExternalId},
			},
		},
		{
			name: "Delete Application is synced at federation partner and its finalizer removed in a single reconcile",
			fields: fields{
				resources: []client.Object{feder, file,
					makeTestApplication(testFederationContextId, appWithFinalizer(), appWithDeletedAt(time.Now()))},
				mockOpgFederations: []*v1beta1.Federation{feder},
				mockOpgApps:        []*v1beta1.Application{makeTestApplication(testFederationContextId)},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Name: testAppName, Namespace: testNamespace}},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantGetErr:       errors.IsNotFound,
				wantAPIApps:      []string{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			apiObjs := &ApiObjects{
				Federations: tt.fields.mockOpgFederations,
			}
			cl, opgcmap, mockedOpgAPI, sch := prepareEnv(tt.fields.resources, apiObjs)
			r := makeTestAppReconciler(cl, sch, opgcmap)

			gotResult, err := r.Reconcile(ctx, tt.args.req)

			if tt.resp.wantReconcileErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.resp.wantResult, gotResult)

			for _, apiApp := range tt.resp.wantAPIApps {
				assert.Contains(t, mockedOpgAPI.Apps, apiApp)
			}

			var reqApp v1beta1.Application
			err = r.Client.Get(ctx, tt.args.req.NamespacedName, &reqApp)
			if tt.resp.wantGetErr != nil {
				assert.True(t, tt.resp.wantGetErr(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.resp.wantStatusPhase, reqApp.Status.Phase)
			assert.Equal(t, tt.resp.wantStatusState, reqApp.Status.State)
			assert.Contains(t, reqApp.Finalizers, tt.resp.wantFinalizer)

		})
	}
}

func appWithFinalizer() appOpt {
	return func(f *v1beta1.Application) {
		controllerutil.AddFinalizer(f, v1beta1.AppFinalizer)
	}
}

func appWithDeletedAt(now time.Time) appOpt {
	return func(a *v1beta1.Application) {
		wrapped := metav1.NewTime(now)
		a.ObjectMeta.DeletionTimestamp = &wrapped
		a.Finalizers = []string{v1beta1.AppFinalizer}
	}
}

func appWithPhase(ph v1beta1.ApplicationPhase) appOpt {
	return func(f *v1beta1.Application) {
		f.Status.Phase = ph
	}
}

type appOpt func(*v1beta1.Application)

func makeTestApplication(fedCtxId string, opts ...appOpt) *v1beta1.Application {
	a := &v1beta1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAppName,
			Namespace: testNamespace,
			Labels: map[string]string{
				v1beta1.FederationContextIdLabel: fedCtxId,
				v1beta1.ExternalIdLabel:          testAppExternalId,
				v1beta1.FederationRelationLabel:  string(defaultTestFederationRelation),
			},
		},
		Spec: v1beta1.ApplicationSpec{
			AppProviderId: testAppProvider,
			ComponentSpecs: []v1beta1.ComponentSpecRef{{
				ArtefactId: testArtefactName,
			}},
			MetaData: v1beta1.AppMetaData{
				AccessToken:     "a1234567890123456789012345678901234567890123456789012345678901",
				Name:            testAppMetaDataName,
				MobilitySupport: false,
				Version:         testAppMetaDataVersion,
			},
			QoSProfile: v1beta1.QoSProfile{
				Provisioning:       false,
				LatencyConstraints: "LOW",
			},
			StatusLink: "https://onboard.app/callback",
		},
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

func makeTestAppReconciler(
	client client.Client,
	sch *runtime.Scheme,
	opgClients opg.OPGClientsMapInterface,
) *ApplicationReconciler {
	r := &ApplicationReconciler{
		Client:                 client,
		Scheme:                 sch,
		OPGClientsMapInterface: opgClients,
	}
	return r
}
