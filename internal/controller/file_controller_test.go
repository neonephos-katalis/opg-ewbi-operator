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

	// File
	testFileName        = "file001"
	testFileFileName    = "busybox"
	testFileFileVersion = "latest"
	testFileExternalId  = "file-00000000-0000-0000-0000-000000000001"
)

func TestFileReconciler(t *testing.T) {
	feder := makeTestFederation(testFederationName, withFederationContextId(testFederationContextId),
		federationWithFederationRelation(v1beta1.FederationRelationGuest),
	)
	federHost := makeTestFederation(
		"hostFeder",
		withFederationContextIdAsLabel(testFederationContextId),
		federationWithFederationRelation(v1beta1.FederationRelationHost),
	)

	type fields struct {
		resources          []client.Object
		mockOpgFederations []*v1beta1.Federation
		mockOpgFiles       []*v1beta1.File
	}
	type args struct {
		req ctrl.Request
	}
	type response struct {
		wantResult       ctrl.Result
		wantReconcileErr bool
		wantGetErr       func(err error) bool
		wantStatusPhase  v1beta1.FilePhase
		wantFinalizer    string
		wantAPIFiles     []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		resp   response
	}{
		{
			name: "New File without Finalizer will get it and return",
			fields: fields{
				resources: []client.Object{feder, makeTestFile(testFederationContextId)},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testFileName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  "",
				wantFinalizer:    v1beta1.FileFinalizer,
			},
		},
		{
			name: "A Host File is ignored, phase is set to Ready",
			fields: fields{
				resources: []client.Object{federHost, makeTestFile(
					testFederationContextId,
					fileWithFederationRelationLabel(v1beta1.FederationRelationHost),
					fileWithFinalizer(),
				)},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testFileName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  v1beta1.FilePhaseReady,
				wantFinalizer:    v1beta1.FileFinalizer,
			},
		},
		{
			name: "A Host File is ignored, phase is set to Ready even if same file in Guest Mode exists",
			fields: fields{
				resources: []client.Object{feder, federHost,
					makeTestFile(
						testFederationContextId,
						fileWithFinalizer(),
						fileWithFederationRelationLabel(v1beta1.FederationRelationHost),
					),
				},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testFileName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  v1beta1.FilePhaseReady,
				wantFinalizer:    v1beta1.FileFinalizer,
			},
		},
		{
			name: "A New Guest File is created at federation partner Operator",
			fields: fields{
				resources:          []client.Object{feder, makeTestFile(testFederationContextId, fileWithFinalizer())},
				mockOpgFederations: []*v1beta1.Federation{feder},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testFileName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  v1beta1.FilePhaseReady,
				wantFinalizer:    v1beta1.FileFinalizer,
				wantAPIFiles:     []string{testFileExternalId},
			},
		},
		{
			name: "An existing Guest File is synced at federation partner if already exists",
			fields: fields{
				resources: []client.Object{
					feder,
					makeTestFile(testFederationContextId, fileWithFinalizer(), fileWithPhase(v1beta1.FilePhaseReady))},
				mockOpgFederations: []*v1beta1.Federation{feder},
				mockOpgFiles:       []*v1beta1.File{makeTestFile(testFederationContextId)},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Name: testFileName, Namespace: testNamespace}},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  v1beta1.FilePhaseReady,
				wantFinalizer:    v1beta1.FileFinalizer,
				wantAPIFiles:     []string{testFileExternalId},
			},
		},
		{
			name: "Delete File is synced at federation partner and its finalizer removed in a single reconcile",
			fields: fields{
				resources: []client.Object{feder,
					makeTestFile(testFederationContextId, fileWithFinalizer(), fileWithDeletedAt(time.Now()))},
				mockOpgFederations: []*v1beta1.Federation{feder},
				mockOpgFiles:       []*v1beta1.File{makeTestFile(testFederationContextId)},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Name: testFileName, Namespace: testNamespace}},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantGetErr:       errors.IsNotFound,
				wantAPIFiles:     []string{},
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
			r := makeTestFileReconciler(cl, sch, opgcmap)

			gotResult, err := r.Reconcile(ctx, tt.args.req)

			if tt.resp.wantReconcileErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.resp.wantResult, gotResult)

			for _, apiFile := range tt.resp.wantAPIFiles {
				assert.Contains(t, mockedOpgAPI.Files, apiFile)
			}

			var reqFile v1beta1.File
			err = r.Client.Get(ctx, tt.args.req.NamespacedName, &reqFile)
			if tt.resp.wantGetErr != nil {
				assert.True(t, tt.resp.wantGetErr(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.resp.wantStatusPhase, reqFile.Status.Phase)
			assert.Contains(t, reqFile.Finalizers, tt.resp.wantFinalizer)

		})
	}
}

type fileOpt func(*v1beta1.File)

func fileWithFinalizer() fileOpt {
	return func(f *v1beta1.File) {
		controllerutil.AddFinalizer(f, v1beta1.FileFinalizer)
	}
}

func fileWithFederationRelationLabel(rel v1beta1.FederationRelation) fileOpt {
	return func(f *v1beta1.File) {
		f.Labels[v1beta1.FederationRelationLabel] = string(rel)
	}
}

func fileWithDeletedAt(now time.Time) fileOpt {
	return func(a *v1beta1.File) {
		wrapped := metav1.NewTime(now)
		a.ObjectMeta.DeletionTimestamp = &wrapped
		a.Finalizers = []string{v1beta1.FileFinalizer}
	}
}

func fileWithPhase(ph v1beta1.FilePhase) fileOpt {
	return func(f *v1beta1.File) {
		f.Status.Phase = ph
	}
}

func makeTestFile(fedCtxId string, opts ...fileOpt) *v1beta1.File {
	a := &v1beta1.File{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testFileName,
			Namespace: testNamespace,
			Labels: map[string]string{
				v1beta1.FederationContextIdLabel: fedCtxId,
				v1beta1.ExternalIdLabel:          testFileExternalId,
				v1beta1.FederationRelationLabel:  string(defaultTestFederationRelation),
			},
		},
		Spec: v1beta1.FileSpec{
			AppProviderId: testAppProvider,
			FileName:      testFileFileName,
			FileVersion:   testFileFileVersion,
			FileType:      "CONTAINER",
			Repo: v1beta1.Repo{
				Type:     "private",
				URL:      "https://harbor.example.com/repo",
				Password: "pass",
				Token:    "token",
				UserName: "foo",
			},
			Image: v1beta1.Image{
				InstructionSetArchitecture: "ISA_X86_64",
				OS: v1beta1.OS{
					Architecture: "x86_64",
					Distribution: "UBUNTU",
					License:      "OS_LICENSE_TYPE_FREE",
					Version:      "OS_VERSION_UBUNTU_2204_LTS",
				},
			},
		},
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

func makeTestFileReconciler(
	client client.Client,
	sch *runtime.Scheme,
	opgClients opg.OPGClientsMapInterface,
) *FileReconciler {
	r := &FileReconciler{
		Client:                 client,
		Scheme:                 sch,
		OPGClientsMapInterface: opgClients,
	}
	return r
}
