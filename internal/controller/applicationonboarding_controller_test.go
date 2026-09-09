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

	"github.com/neonephos-katalis/opg-ewbi-operator/api/operator/v1beta1"
	"github.com/neonephos-katalis/opg-ewbi-operator/internal/opg"
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
	file := makeTestImage(testFederationContextId)

	type fields struct {
		resources          []client.Object
		mockOpgFederations []*v1beta1.Federation
		mockOpgApps        []*v1beta1.ApplicationOnboarding
	}
	type args struct {
		req ctrl.Request
	}
	type response struct {
		wantResult       ctrl.Result
		wantReconcileErr bool
		wantGetErr       func(err error) bool
		wantStatusState  v1beta1.ApplicationOnboardingState
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
				resources: []client.Object{feder, file, makeTestApplicationOnboarding(testFederationContextId)},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testAppName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusState:  "",
				wantFinalizer:    v1beta1.ApplicationOnboardingFinalizer,
			},
		},
		{
			name: "A Host Application is ignored, state is set to Onboarded",
			fields: fields{
				resources: []client.Object{feder, file, makeTestApplicationOnboarding(testFederationContextId, appWithFinalizer())},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testAppName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusState:  v1beta1.ApplicationOnboardingStateOnboarded,
				wantFinalizer:    v1beta1.ApplicationOnboardingFinalizer,
			},
		},
		{
			name: "A New Guest Application is created at federation partner Operator",
			fields: fields{
				resources:          []client.Object{feder, file, makeTestApplicationOnboarding(testFederationContextId, appWithFinalizer())},
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
				wantStatusState:  v1beta1.ApplicationOnboardingStateOnboarded,
				wantFinalizer:    v1beta1.ApplicationOnboardingFinalizer,
				wantAPIApps:      []string{testAppExternalId},
			},
		},
		{
			name: "An existing Guest Application is synced at federation partner if already exists",
			fields: fields{
				resources: []client.Object{
					feder,
					file,
					makeTestApplicationOnboarding(testFederationContextId, appWithFinalizer(), appWithState(v1beta1.ApplicationOnboardingStateOnboarded))},
				mockOpgFederations: []*v1beta1.Federation{feder},
				mockOpgApps:        []*v1beta1.ApplicationOnboarding{makeTestApplicationOnboarding(testFederationContextId)},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Name: testAppName, Namespace: testNamespace}},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusState:  v1beta1.ApplicationOnboardingStateOnboarded,
				wantFinalizer:    v1beta1.ApplicationOnboardingFinalizer,
				wantAPIApps:      []string{testAppExternalId},
			},
		},
		{
			name: "Delete ApplicationOnboarding is synced at federation partner and its finalizer removed in a single reconcile",
			fields: fields{
				resources: []client.Object{feder, file,
					makeTestApplicationOnboarding(testFederationContextId, appWithFinalizer(), appWithDeletedAt(time.Now()))},
				mockOpgFederations: []*v1beta1.Federation{feder},
				mockOpgApps:        []*v1beta1.ApplicationOnboarding{makeTestApplicationOnboarding(testFederationContextId)},
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
				assert.Contains(t, mockedOpgAPI.AppOnboards, apiApp)
			}

			var reqApp v1beta1.ApplicationOnboarding
			err = r.Client.Get(ctx, tt.args.req.NamespacedName, &reqApp)
			if tt.resp.wantGetErr != nil {
				assert.True(t, tt.resp.wantGetErr(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.resp.wantStatusState, reqApp.Status.State)
			assert.Contains(t, reqApp.Finalizers, tt.resp.wantFinalizer)

		})
	}
}

func appWithFinalizer() appOpt {
	return func(f *v1beta1.ApplicationOnboarding) {
		controllerutil.AddFinalizer(f, v1beta1.ApplicationOnboardingFinalizer)
	}
}

func appWithDeletedAt(now time.Time) appOpt {
	return func(a *v1beta1.ApplicationOnboarding) {
		wrapped := metav1.NewTime(now)
		a.ObjectMeta.DeletionTimestamp = &wrapped
		a.Finalizers = []string{v1beta1.ApplicationOnboardingFinalizer}
	}
}

func appWithState(state v1beta1.ApplicationOnboardingState) appOpt {
	return func(f *v1beta1.ApplicationOnboarding) {
		f.Status.State = state
	}
}

type appOpt func(*v1beta1.ApplicationOnboarding)

func makeTestApplicationOnboarding(fedCtxId string, opts ...appOpt) *v1beta1.ApplicationOnboarding {
	a := &v1beta1.ApplicationOnboarding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAppName,
			Namespace: testNamespace,
		},
		Spec: v1beta1.ApplicationOnboardingSpec{
			FederationContextId: fedCtxId,
			AppInfo: &v1beta1.AppInfo{
				AppProviderId: testAppProvider,
				AppComponentSpecs: []v1beta1.AppComponentSpec{{
					ArtefactId: testArtefactName,
				}},
				AppMetaData: &v1beta1.AppMetaData{
					AccessToken:     "a1234567890123456789012345678901234567890123456789012345678901",
					AppName:         testAppMetaDataName,
					MobilitySupport: false,
					Version:         testAppMetaDataVersion,
				},
				AppQoSProfile: &v1beta1.AppQoSProfile{
					AppProvisioning:    false,
					LatencyConstraints: "LOW",
				},
				AppStatusCallbackLink: "https://onboard.app/callback",
			},
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
) *ApplicationOnboardingReconciler {
	r := &ApplicationOnboardingReconciler{
		Client:                 client,
		Scheme:                 sch,
		OPGClientsMapInterface: opgClients,
	}
	return r
}
