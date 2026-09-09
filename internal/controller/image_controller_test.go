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

	// File
	testFileName        = "file001"
	testFileFileName    = "busybox"
	testFileFileVersion = "latest"
	testFileExternalId  = "file-00000000-0000-0000-0000-000000000001"
)

func TestFileReconciler(t *testing.T) {
	feder := makeTestFederation(testFederationName, withFederationContextId(testFederationContextId))
	federHost := makeTestFederation(
		"hostFeder",
	)

	type fields struct {
		resources          []client.Object
		mockOpgFederations []*v1beta1.Federation
		mockOpgImages      []*v1beta1.Image
	}
	type args struct {
		req ctrl.Request
	}
	type response struct {
		wantResult       ctrl.Result
		wantReconcileErr bool
		wantGetErr       func(err error) bool
		wantStatusState  v1beta1.ImageState
		wantFinalizer    string
		wantAPIImages    []string
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
				resources: []client.Object{feder, makeTestImage(testFederationContextId)},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testFileName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusState:  "",
				wantFinalizer:    v1beta1.ImageFinalizer,
			},
		},
		{
			name: "A Host Image is ignored, state is set to Ready",
			fields: fields{
				resources: []client.Object{federHost, makeTestImage(
					testFederationContextId,
					imageWithFinalizer(),
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
				wantStatusState:  v1beta1.ImageStateReady,
				wantFinalizer:    v1beta1.ImageFinalizer,
			},
		},
		{
			name: "A Host Image is ignored, state is set to Ready even if same image in Guest Mode exists",
			fields: fields{
				resources: []client.Object{feder, federHost,
					makeTestImage(
						testFederationContextId,
						imageWithFinalizer(),
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
				wantStatusState:  v1beta1.ImageStateReady,
				wantFinalizer:    v1beta1.ImageFinalizer,
			},
		},
		{
			name: "A New Guest Image is created at federation partner Operator",
			fields: fields{
				resources:          []client.Object{feder, makeTestImage(testFederationContextId, imageWithFinalizer())},
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
				wantStatusState:  v1beta1.ImageStateReady,
				wantFinalizer:    v1beta1.ImageFinalizer,
				wantAPIImages:    []string{testFileExternalId},
			},
		},
		{
			name: "An existing Guest Image is synced at federation partner if already exists",
			fields: fields{
				resources: []client.Object{
					feder,
					makeTestImage(testFederationContextId, imageWithFinalizer(), imageWithState(v1beta1.ImageStateReady))},
				mockOpgFederations: []*v1beta1.Federation{feder},
				mockOpgImages:      []*v1beta1.Image{makeTestImage(testFederationContextId)},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Name: testFileName, Namespace: testNamespace}},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusState:  v1beta1.ImageStateReady,
				wantFinalizer:    v1beta1.ImageFinalizer,
				wantAPIImages:    []string{testFileExternalId},
			},
		},
		{
			name: "Delete Image is synced at federation partner and its finalizer removed in a single reconcile",
			fields: fields{
				resources: []client.Object{feder,
					makeTestImage(testFederationContextId, imageWithFinalizer(), imageWithDeletedAt(time.Now()))},
				mockOpgFederations: []*v1beta1.Federation{feder},
				mockOpgImages:      []*v1beta1.Image{makeTestImage(testFederationContextId)},
			},
			args: args{
				req: ctrl.Request{NamespacedName: types.NamespacedName{Name: testFileName, Namespace: testNamespace}},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantGetErr:       errors.IsNotFound,
				wantAPIImages:    []string{},
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

			for _, apiFile := range tt.resp.wantAPIImages {
				assert.Contains(t, mockedOpgAPI.Images, apiFile)
			}

			var reqFile v1beta1.Image
			err = r.Client.Get(ctx, tt.args.req.NamespacedName, &reqFile)
			if tt.resp.wantGetErr != nil {
				assert.True(t, tt.resp.wantGetErr(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.resp.wantStatusState, reqFile.Status.State)
			assert.Contains(t, reqFile.Finalizers, tt.resp.wantFinalizer)

		})
	}
}

type imageOpt func(*v1beta1.Image)

func imageWithFinalizer() imageOpt {
	return func(f *v1beta1.Image) {
		controllerutil.AddFinalizer(f, v1beta1.ImageFinalizer)
	}
}

func imageWithDeletedAt(now time.Time) imageOpt {
	return func(a *v1beta1.Image) {
		wrapped := metav1.NewTime(now)
		a.ObjectMeta.DeletionTimestamp = &wrapped
		a.Finalizers = []string{v1beta1.ImageFinalizer}
	}
}

func imageWithState(state v1beta1.ImageState) imageOpt {
	return func(f *v1beta1.Image) {
		f.Status.State = state
	}
}

func makeTestImage(fedCtxId string, opts ...imageOpt) *v1beta1.Image {
	a := &v1beta1.Image{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testFileName,
			Namespace: testNamespace,
		},
		Spec: v1beta1.ImageSpec{
			FederationContextId: fedCtxId,
			ImageBody: &v1beta1.ImageBody{
				AppProviderId:    testAppProvider,
				ImageName:        testFileFileName,
				ImageVersionInfo: testFileFileVersion,
				ImageType:        "CONTAINER",
				RepoType:         "private",
				ImageRepoLocation: &v1beta1.ImageRepoLocation{
					RepoURL:  "https://harbor.example.com/repo",
					Password: "pass",
					Token:    "token",
					UserName: "foo",
				},
				ImgInsSetArch: "ISA_X86_64",
				ImgOSType: &v1beta1.ImgOSType{
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
) *ImageReconciler {
	r := &ImageReconciler{
		Client:                 client,
		Scheme:                 sch,
		OPGClientsMapInterface: opgClients,
	}
	return r
}
