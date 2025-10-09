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
	opgewbiv1beta1 "github.com/nbycomp/neonephos-opg-ewbi-operator/api/v1beta1"
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

	// Artefact
	testArtefactName       = "artefact001"
	testArtefactExternalId = "art-00000000-0000-0000-0000-000000000001"
)

func TestArtefactReconciler(t *testing.T) {
	feder := makeTestFederation(testFederationName, withFederationContextId(testFederationContextId),
		federationWithFederationRelation(v1beta1.FederationRelationGuest),
	)
	file := makeTestFile(testFederationContextId)

	type fields struct {
		resources          []client.Object
		mockOpgFederations []*v1beta1.Federation
		mockOpgArtefacts   []*v1beta1.Artefact
	}
	type args struct {
		req ctrl.Request
	}
	type response struct {
		wantResult       ctrl.Result
		wantReconcileErr bool
		wantGetErr       func(err error) bool
		wantStatusPhase  v1beta1.ArtefactPhase
		wantFinalizer    string
		wantAPIArtefacts []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		resp   response
	}{
		{
			name: "New Artefact without Finalizer will get it and return",
			fields: fields{
				resources: []client.Object{feder, file, makeTestArtefact(testFederationContextId)},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testArtefactName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  "",
				wantFinalizer:    v1beta1.ArtefactFinalizer,
			},
		},
		{
			name: "A Host Artefact is ignored, phase is set to Ready",
			fields: fields{
				resources: []client.Object{feder, file, makeTestArtefact(testFederationContextId, artefactWithFinalizer())},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testArtefactName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  v1beta1.ArtefactPhaseReady,
				wantFinalizer:    v1beta1.ArtefactFinalizer,
			},
		},
		{
			name: "A New Guest Artefact is created at federation partner Operator",
			fields: fields{
				resources: []client.Object{
					feder,
					file,
					makeTestArtefact(testFederationContextId, artefactWithFinalizer()),
				},
				mockOpgFederations: []*v1beta1.Federation{feder},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testArtefactName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  v1beta1.ArtefactPhaseReady,
				wantFinalizer:    v1beta1.ArtefactFinalizer,
				wantAPIArtefacts: []string{testArtefactExternalId},
			},
		},
		{
			name: "An existing Guest Artefact is synced at federation partner if already exists",
			fields: fields{
				resources: []client.Object{
					feder,
					file,
					makeTestArtefact(testFederationContextId, artefactWithFinalizer(), artefactWithPhase(v1beta1.ArtefactPhaseReady))},
				mockOpgFederations: []*v1beta1.Federation{feder},
				mockOpgArtefacts:   []*v1beta1.Artefact{makeTestArtefact(testFederationContextId)},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Name: testArtefactName, Namespace: testNamespace}},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  v1beta1.ArtefactPhaseReady,
				wantFinalizer:    v1beta1.ArtefactFinalizer,
				wantAPIArtefacts: []string{testArtefactExternalId},
			},
		},
		{
			name: "Delete Artefact is synced at federation partner and its finalizer removed in a single reconcile",
			fields: fields{
				resources: []client.Object{feder, file,
					makeTestArtefact(testFederationContextId, artefactWithFinalizer(), artefactWithDeletedAt(time.Now()))},
				mockOpgFederations: []*v1beta1.Federation{feder},
				mockOpgArtefacts:   []*v1beta1.Artefact{makeTestArtefact(testFederationContextId)},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Name: testArtefactName, Namespace: testNamespace}},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantGetErr:       errors.IsNotFound,
				wantAPIArtefacts: []string{},
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
			r := makeTestArtefactReconciler(cl, sch, opgcmap)

			gotResult, err := r.Reconcile(ctx, tt.args.req)

			if tt.resp.wantReconcileErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.resp.wantResult, gotResult)

			for _, apiArtefact := range tt.resp.wantAPIArtefacts {
				assert.Contains(t, mockedOpgAPI.Artefacts, apiArtefact)
			}

			var reqArt v1beta1.Artefact
			err = r.Client.Get(ctx, tt.args.req.NamespacedName, &reqArt)
			if tt.resp.wantGetErr != nil {
				assert.True(t, tt.resp.wantGetErr(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.resp.wantStatusPhase, reqArt.Status.Phase)
			assert.Contains(t, reqArt.Finalizers, tt.resp.wantFinalizer)

		})
	}
}

type artefactOpt func(*opgewbiv1beta1.Artefact)

func artefactWithFinalizer() artefactOpt {
	return func(f *v1beta1.Artefact) {
		controllerutil.AddFinalizer(f, v1beta1.ArtefactFinalizer)
	}
}

func artefactWithDeletedAt(now time.Time) artefactOpt {
	return func(a *opgewbiv1beta1.Artefact) {
		wrapped := metav1.NewTime(now)
		a.ObjectMeta.DeletionTimestamp = &wrapped
		a.Finalizers = []string{opgewbiv1beta1.ArtefactFinalizer}
	}
}

func artefactWithPhase(ph v1beta1.ArtefactPhase) artefactOpt {
	return func(f *v1beta1.Artefact) {
		f.Status.Phase = ph
	}
}

func makeTestArtefact(fedCtxId string, opts ...artefactOpt) *opgewbiv1beta1.Artefact {
	a := &opgewbiv1beta1.Artefact{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testArtefactName,
			Namespace: testNamespace,
			Labels: map[string]string{
				opgewbiv1beta1.FederationContextIdLabel: fedCtxId,
				opgewbiv1beta1.ExternalIdLabel:          testArtefactExternalId,
				v1beta1.FederationRelationLabel:         string(defaultTestFederationRelation),
			},
		},
		Spec: opgewbiv1beta1.ArtefactSpec{
			AppProviderId:   testAppProvider,
			ArtefactName:    "ContainerDeploy001",
			ArtefactVersion: "14",
			DescriptorType:  "COMPONENTSPEC",
			VirtType:        "CONTAINER_TYPE",
			ComponentSpec: []opgewbiv1beta1.ComponentSpec{{
				Name: "test-pod",
				CommandLineParams: opgewbiv1beta1.CommandLine{
					Command: []string{"nginx-debug"},
					Args:    []string{"-g", "daemon off;"},
				},
				Images:         []string{testFileName},
				NumOfInstances: 0,
				RestartPolicy:  "RESTART_POLICY_ALWAYS",
				ComputeResourceProfile: opgewbiv1beta1.ComputeResourceProfile{
					CPUArchType:    "ISA_X86_64",
					CPUExclusivity: false,
					Memory:         512,
					NumCPU:         "1",
				},
				ExposedInterfaces: []opgewbiv1beta1.ExposedInterface{{
					Port:           30013,
					Protocol:       "TCP",
					InterfaceId:    "interfaceid",
					VisibilityType: "VISIBILITY_EXTERNAL",
				}},
			}},
		},
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

func makeTestArtefactReconciler(
	client client.Client,
	sch *runtime.Scheme,
	opgClients opg.OPGClientsMapInterface,
) *ArtefactReconciler {
	r := &ArtefactReconciler{
		Client:                 client,
		Scheme:                 sch,
		OPGClientsMapInterface: opgClients,
	}
	return r
}
