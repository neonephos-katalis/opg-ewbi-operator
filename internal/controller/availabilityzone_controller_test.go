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

	"github.com/nbycomp/neonephos-opg-ewbi-operator/api/v1beta1"
	opgewbiv1beta1 "github.com/nbycomp/neonephos-opg-ewbi-operator/api/v1beta1"
	"github.com/nbycomp/neonephos-opg-ewbi-operator/internal/opg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (

	// AvailabilityZone
	testAZName = "az001"
)

func TestAvailabilityZoneReconciler(t *testing.T) {
	feder := makeTestFederation(testFederationName, withFederationContextId(testFederationContextId))

	type fields struct {
		resources          []client.Object
		mockOpgFederations []*v1beta1.Federation
	}
	type args struct {
		req ctrl.Request
	}
	type response struct {
		wantResult       ctrl.Result
		wantReconcileErr bool
		wantGetErr       func(err error) bool
		wantStatusPhase  v1beta1.AvailabilityZonePhase
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		resp   response
	}{
		{
			name: "AvailabilityZone's phase is set to Ready",
			fields: fields{
				resources: []client.Object{feder, makeTestAvailabilityZone()},
			},
			args: args{
				req: ctrl.Request{
					NamespacedName: types.NamespacedName{Name: testAZName, Namespace: testNamespace},
				},
			},
			resp: response{
				wantResult:       ctrl.Result{Requeue: false},
				wantReconcileErr: false,
				wantStatusPhase:  v1beta1.AvailabilityZonePhaseReady,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			apiObjs := &ApiObjects{
				Federations: tt.fields.mockOpgFederations,
			}
			cl, opgcmap, _, sch := prepareEnv(tt.fields.resources, apiObjs)
			r := makeTestAvailabilityZoneReconciler(cl, sch, opgcmap)

			gotResult, err := r.Reconcile(ctx, tt.args.req)

			if tt.resp.wantReconcileErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.resp.wantResult, gotResult)

			var reqAvailabilityZone v1beta1.AvailabilityZone
			err = r.Client.Get(ctx, tt.args.req.NamespacedName, &reqAvailabilityZone)
			if tt.resp.wantGetErr != nil {
				assert.True(t, tt.resp.wantGetErr(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.resp.wantStatusPhase, reqAvailabilityZone.Status.Phase)

		})
	}
}

type azOpt func(*opgewbiv1beta1.AvailabilityZone)

func makeTestAvailabilityZone(opts ...azOpt) *opgewbiv1beta1.AvailabilityZone {
	a := &opgewbiv1beta1.AvailabilityZone{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testAZName,
			Namespace: testNamespace,
		},
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

func makeTestAvailabilityZoneReconciler(
	client client.Client,
	sch *runtime.Scheme,
	opgClients opg.OPGClientsMapInterface,
) *AvailabilityZoneReconciler {
	r := &AvailabilityZoneReconciler{
		Client:                 client,
		Scheme:                 sch,
		OPGClientsMapInterface: opgClients,
	}
	return r
}
