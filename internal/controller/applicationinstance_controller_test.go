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
	"github.com/nbycomp/neonephos-opg-ewbi-operator/test/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (

	// ApplicationInstance
	testAppInstName       = "appinst001"
	testAppInstExternalId = "appinst-00000000-0000-0000-0000-000000000001"
)

func TestApplicationInstanceReconciler(t *testing.T) {
	feder := makeTestFederation(testFederationName, withFederationContextId(testFederationContextId))
	file := makeTestFile(testFederationContextId)

	type fields struct {
		resources          []client.Object
		mockOpgFederations []*v1beta1.Federation
		mockOpgAppInsts    []*v1beta1.ApplicationInstance
	}
	type args struct {
		req ctrl.Request
	}
	type response struct {
		wantResult       ctrl.Result
		wantReconcileErr bool
		wantGetErr       func(err error) bool
		wantStatusPhase  v1beta1.ApplicationInstancePhase
		wantStatusState  v1beta1.ApplicationInstanceState
		wantFinalizer    string
		wantAPIAppInsts  []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		resp   response
	}{
		{
			name: "New ApplicationInstance without Finalizer will get it and return",
			fields: fields{
				resources: []client.Object{feder, file, makeTestAppInst(testFederationContextId)},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testAppInstName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  "",
				wantFinalizer:    v1beta1.ApplicationInstanceFinalizer,
			},
		},
		{
			name: "A Host ApplicationInstance is ignored, phase is set to Ready",
			fields: fields{
				resources: []client.Object{feder, file, makeTestAppInst(testFederationContextId, appInstWithFinalizer())},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testAppInstName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  v1beta1.ApplicationInstancePhaseReady,
				wantFinalizer:    v1beta1.ApplicationInstanceFinalizer,
			},
		},
		{
			name: "A New Guest ApplicationInstance is created at federation partner Operator",
			fields: fields{
				resources:          []client.Object{feder, file, makeTestAppInst(testFederationContextId, appInstWithFinalizer())},
				mockOpgFederations: []*v1beta1.Federation{feder},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testAppInstName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  v1beta1.ApplicationInstancePhaseReady,
				wantFinalizer:    v1beta1.ApplicationInstanceFinalizer,
				wantAPIAppInsts:  []string{testAppInstExternalId},
			},
		},
		{
			name: "An existing Guest ApplicationInstance is synced at federation partner if already exists",
			fields: fields{
				resources: []client.Object{
					feder,
					file,
					makeTestAppInst(testFederationContextId,
						appInstWithFinalizer(),
						appInstWithPhase(v1beta1.ApplicationInstancePhaseReady),
					)},
				mockOpgFederations: []*v1beta1.Federation{feder},
				mockOpgAppInsts:    []*v1beta1.ApplicationInstance{makeTestAppInst(testFederationContextId)},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Name: testAppInstName, Namespace: testNamespace}},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  v1beta1.ApplicationInstancePhaseReady,
				wantFinalizer:    v1beta1.ApplicationInstanceFinalizer,
				wantAPIAppInsts:  []string{testAppInstExternalId},
			},
		},
		{
			name: "Delete ApplicationInstance is synced at federation partner and its finalizer removed in a single reconcile",
			fields: fields{
				resources: []client.Object{feder, file,
					makeTestAppInst(testFederationContextId, appInstWithFinalizer(), appInstWithDeletedAt(time.Now()))},
				mockOpgFederations: []*v1beta1.Federation{feder},
				mockOpgAppInsts:    []*v1beta1.ApplicationInstance{makeTestAppInst(testFederationContextId)},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Name: testAppInstName, Namespace: testNamespace}},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantGetErr:       errors.IsNotFound,
				wantAPIAppInsts:  []string{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			apiObjs := &ApiObjects{
				Federations: tt.fields.mockOpgFederations,
				AppInsts:    tt.fields.mockOpgAppInsts,
			}
			cl, opgcmap, mockedOpgAPI, sch := prepareEnv(tt.fields.resources, apiObjs)

			r := makeTestAppInstReconciler(cl, sch, opgcmap)

			gotResult, err := r.Reconcile(ctx, tt.args.req)

			if tt.resp.wantReconcileErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.resp.wantResult, gotResult)

			for _, apiAppInst := range tt.resp.wantAPIAppInsts {
				assert.Contains(t, mockedOpgAPI.AppInsts, apiAppInst)
			}

			var reqAppInst v1beta1.ApplicationInstance
			err = r.Client.Get(ctx, tt.args.req.NamespacedName, &reqAppInst)
			if tt.resp.wantGetErr != nil {
				assert.True(t, tt.resp.wantGetErr(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.resp.wantStatusPhase, reqAppInst.Status.Phase)
			assert.Equal(t, tt.resp.wantStatusState, reqAppInst.Status.State)
			assert.Contains(t, reqAppInst.Finalizers, tt.resp.wantFinalizer)

		})
	}
}

type appInstOpt func(*v1beta1.ApplicationInstance)

func appInstWithDeletedAt(now time.Time) appInstOpt {
	return func(a *v1beta1.ApplicationInstance) {
		wrapped := metav1.NewTime(now)
		a.ObjectMeta.DeletionTimestamp = &wrapped
		a.Finalizers = []string{v1beta1.ApplicationInstanceFinalizer}
	}
}

func appInstWithFinalizer() appInstOpt {
	return func(f *v1beta1.ApplicationInstance) {
		controllerutil.AddFinalizer(f, v1beta1.ApplicationInstanceFinalizer)
	}
}

func appInstWithPhase(ph v1beta1.ApplicationInstancePhase) appInstOpt {
	return func(f *v1beta1.ApplicationInstance) {
		f.Status.Phase = ph
	}
}

func makeTestAppInst(fedCtxId string, opts ...appInstOpt) *v1beta1.ApplicationInstance {
	a := &v1beta1.ApplicationInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAppInstName,
			Namespace: testNamespace,
			Labels: map[string]string{
				v1beta1.FederationContextIdLabel: fedCtxId,
				v1beta1.ExternalIdLabel:          testAppInstExternalId,
				v1beta1.FederationRelationLabel:  string(defaultTestFederationRelation),
			},
		},
		Spec: v1beta1.ApplicationInstanceSpec{
			AppProviderId: testAppProvider,
			AppId:         testAppName,
			AppVersion:    testAppMetaDataVersion,
			ZoneInfo: v1beta1.Zone{
				ZoneId:              testAZName,
				FlavourId:           "NOT_SPECIFIED",
				ResourceConsumption: "RESERVED_RES_AVOID",
				ResPool:             "mock-res-pool",
			},
			CallbBackLink: "https://onboard.app/callback",
		},
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

func makeTestAppInstReconciler(
	client client.Client,
	sch *runtime.Scheme,
	opgClients opg.OPGClientsMapInterface,
) *ApplicationInstanceReconciler {
	r := &ApplicationInstanceReconciler{
		Client:                 client,
		Scheme:                 sch,
		OPGClientsMapInterface: opgClients,
	}
	return r
}

type ApiObjects struct {
	Federations []*v1beta1.Federation
	Files       []*v1beta1.File
	Artefacts   []*v1beta1.Artefact
	Apps        []*v1beta1.Application
	AppInsts    []*v1beta1.ApplicationInstance
	AZs         []*v1beta1.AvailabilityZone
}

func prepareEnv(clientObjs []client.Object, apiObjs *ApiObjects,
) (client.Client, opg.OPGClientsMapInterface, *mock.MockedOpgAPI, *runtime.Scheme) {
	log.SetLogger(zap.New(zap.UseDevMode(true)))
	sch := makeTestReconcilerScheme(v1beta1.AddToScheme)
	cl := makeTestReconcilerClient(sch, clientObjs, clientObjs, []runtime.Object{})

	mockedOpgAPI := mock.MakeMokedOpgAPI()
	mockedOpgAPI.WithFederations(apiObjs.Federations)
	mockedOpgAPI.WithFiles(apiObjs.Files)
	mockedOpgAPI.WithArtefacts(apiObjs.Artefacts)
	mockedOpgAPI.WithApplications(apiObjs.Apps)
	mockedOpgAPI.WithApplicationInstances(apiObjs.AppInsts)
	mockedOpgAPI.WithAZs(apiObjs.AZs)

	opgcmap := opg.NewOPGClientsMap()
	opgcmap.SetOPGClient(testFederationExternalId, mockedOpgAPI)

	return cl, opgcmap, mockedOpgAPI, sch
}
