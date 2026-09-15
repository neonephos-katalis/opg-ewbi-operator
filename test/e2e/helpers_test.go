package e2e

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/google/uuid"
)

const (
	pollInterval = 3 * time.Second

	minPartneropUpdateDelay = 5 * time.Second
	maxPartneropUpdateDelay = 10 * time.Second
)

func loadSample[T any](t *testing.T, fileName string) *T {
	t.Helper()

	path := filepath.Join("..", "..", "config", "samples", "k8s", fileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading sample %s: %v", path, err)
	}

	var obj T
	if err := yaml.Unmarshal(raw, &obj); err != nil {
		t.Fatalf("unmarshalling sample %s: %v", path, err)
	}
	return &obj
}

func createOrGet(t *testing.T, ctx context.Context, c client.Client, obj client.Object) bool {
	t.Helper()

	err := c.Create(ctx, obj)
	if err == nil {
		return false
	}
	if !apierrors.IsAlreadyExists(err) {
		t.Fatalf("creating %s/%s: %v", obj.GetNamespace(), obj.GetName(), err)
	}

	key := client.ObjectKeyFromObject(obj)
	if err := c.Get(ctx, key, obj); err != nil {
		t.Fatalf("getting existing object %s/%s: %v", obj.GetNamespace(), obj.GetName(), err)
	}
	return true
}

func findObject(
	t *testing.T,
	ctx context.Context,
	c client.Client,
	list client.ObjectList,
	namespace string,
	label string,
	predicate func(obj client.Object) bool,
	timeout time.Duration,
) client.Object {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		if err := c.List(ctx, list, client.InNamespace(namespace)); err != nil {
			t.Fatalf("listing %s in namespace %s: %v", label, namespace, err)
		}

		items, err := apimeta.ExtractList(list)
		if err != nil {
			t.Fatalf("extracting list items for %s: %v", label, err)
		}

		for _, item := range items {
			obj, ok := item.(client.Object)
			if !ok {
				t.Fatalf("list item for %s does not implement client.Object (%T)", label, item)
			}
			if predicate(obj) {
				return obj
			}
		}

		if time.Now().After(deadline) {
			t.Fatalf("timed out after %s waiting for a matching %s in namespace %s", timeout, label, namespace)
		}

		t.Logf("polling for %s in namespace %s...", label, namespace)
		time.Sleep(pollInterval)
	}
}

func waitForCondition(
	t *testing.T,
	ctx context.Context,
	c client.Client,
	obj client.Object,
	label string,
	predicate func() bool,
	timeout time.Duration,
) {
	t.Helper()

	key := client.ObjectKeyFromObject(obj)
	deadline := time.Now().Add(timeout)
	for {
		if err := c.Get(ctx, key, obj); err != nil {
			t.Fatalf("getting %s: %v", label, err)
		}

		if predicate() {
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("timed out after %s waiting for %s", timeout, label)
		}

		t.Logf("polling %s...", label)
		time.Sleep(pollInterval)
	}
}

func waitBeforePartneropUpdate(t *testing.T) {
	t.Helper()
	d := minPartneropUpdateDelay + time.Duration(rand.Int63n(int64(maxPartneropUpdateDelay-minPartneropUpdateDelay+1)))
	t.Logf("waiting %s before applying partnerop-side status update(s)...", d.Round(time.Millisecond))
	time.Sleep(d)
}

func reusedLabel(reused bool) string {
	if reused {
		return "already existed, reusing it"
	}
	return "created"
}

func genUUID() string {
	return uuid.NewString()
}

const alnumAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

func randomAlnum(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alnumAlphabet[rand.Intn(len(alnumAlphabet))]
	}
	return string(b)
}

func genAppID() string {
	return "app" + randomAlnum(16)
}

func genAppInstanceID() string {
	return randomAlnum(40)
}
