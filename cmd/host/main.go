package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	opgv1beta1 "github.com/nbycomp/neonephos-opg-ewbi-operator/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	k8sClient client.Client
	scheme    = runtime.NewScheme()
	namespace = "default"
)

func init() {
	utilruntime.Must(opgv1beta1.AddToScheme(scheme))

	config := ctrl.GetConfigOrDie()
	var err error

	k8sClient, err = client.New(config, client.Options{
		Scheme: scheme,
	})

	if err != nil {
		log.Fatalf("Error creating k8sClient: %v", err)
	}
}

func listApps(w http.ResponseWriter, r *http.Request) {

	appList := &opgv1beta1.ApplicationInstanceList{}

	if err := k8sClient.List(context.Background(), appList, client.InNamespace(namespace)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(appList); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func createApp(w http.ResponseWriter, r *http.Request) {

	app := &opgv1beta1.ApplicationInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: namespace,
		},
		// not providing any Spec field
	}
	// this is the equivalent to a kubectl create, it doesn't allow to update the status, only the spec.
	if err := k8sClient.Create(context.Background(), app); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(app); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/apps", listApps).Methods("GET")
	r.HandleFunc("/apps", createApp).Methods("POST")
	log.Fatal(http.ListenAndServe(":8080", r))
}
