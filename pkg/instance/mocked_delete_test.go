package instance

import (
	"context"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/testhelper"
	th "github.com/gophercloud/gophercloud/testhelper"
	"github.com/gophercloud/gophercloud/testhelper/client"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
)

func TestDeleteInstanceSuccess(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	testhelper.Mux.HandleFunc("/servers/mock-id", func(w http.ResponseWriter, r *http.Request) {
		testhelper.TestMethod(t, r, "DELETE")
		w.WriteHeader(204)
	})

	providerClient := client.ServiceClient()
	provider := NewProvider(providerClient, "test-cluster")

	ctx := context.Background()
	err := provider.Delete(ctx, "openstack:///mock-id")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestDeleteInstanceNotFound(t *testing.T) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	testhelper.Mux.HandleFunc("/servers/missing-id", func(w http.ResponseWriter, r *http.Request) {
		testhelper.TestMethod(t, r, "DELETE")
		w.WriteHeader(404)
	})

	providerClient := client.ServiceClient()
	provider := NewProvider(providerClient, "test-cluster")

	ctx := context.Background()
	err := provider.Delete(ctx, "openstack:///missing-id")
	if err == nil {
		t.Fatalf("expected error but got none")
	}

	_, ok := err.(*cloudprovider.NodeClaimNotFoundError)
	if !ok {
		t.Fatalf("expected NodeClaimNotFoundError, got: %T (%v)", err, err)
	}
}

func TestDeleteInvalidProviderID(t *testing.T) {
	provider := NewProvider(nil, "test-cluster")

	ctx := context.Background()
	err := provider.Delete(ctx, "wrong-format")

	if err == nil {
		t.Fatalf("expected parsing error but got nil")
	}
}
