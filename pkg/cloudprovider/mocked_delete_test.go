package cloudprovider

import (
	"context"
	"testing"

	"github.com/samber/lo"

	v1openstack "github.com/ianzx15/karpenter-provider-openstack/pkg/apis/v1openstack"
	"github.com/ianzx15/karpenter-provider-openstack/pkg/instance"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	karpcloud "sigs.k8s.io/karpenter/pkg/cloudprovider"
)

// FAKE INSTANCE PROVIDER
type FakeInstanceProvider struct {
	DeletedIDs []string
}

func (f *FakeInstanceProvider) Create(
	ctx context.Context,
	nc *v1openstack.OpenStackNodeClass,
	claim *karpv1.NodeClaim,
	types []*karpcloud.InstanceType,
) (*instance.Instance, error) {
	return nil, nil
}

func (f *FakeInstanceProvider) Delete(ctx context.Context, providerID string) error {
	f.DeletedIDs = append(f.DeletedIDs, providerID)
	return nil
}

func TestCloudProviderDelete_RemovesUnusedNode(t *testing.T) {
	ctx := context.Background()

	scheme := runtime.NewScheme()
	lo.Must0(corev1.AddToScheme(scheme))
	lo.Must0(karpv1.AddToScheme(scheme))

	client := fakeclient.NewClientBuilder().WithScheme(scheme).Build()

	fakeInstanceProvider := &FakeInstanceProvider{}

	cp := New(
		client,
		nil,
		fakeInstanceProvider,
		nil,
	)

	nc := &karpv1.NodeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "unused-node"},
		Status: karpv1.NodeClaimStatus{
			ProviderID: "openstack:///instance-123",
		},
	}
	lo.Must0(client.Create(ctx, nc))

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "unused-node"},
		Spec: corev1.NodeSpec{
			ProviderID: "openstack:///instance-123",
		},
	}
	lo.Must0(client.Create(ctx, node))

	lo.Must0(cp.Delete(ctx, nc))

	if len(fakeInstanceProvider.DeletedIDs) != 1 {
		t.Fatalf("expected 1 delete call, got %d", len(fakeInstanceProvider.DeletedIDs))
	}

	if fakeInstanceProvider.DeletedIDs[0] != "openstack:///instance-123" {
		t.Fatalf("wrong id: %s", fakeInstanceProvider.DeletedIDs[0])
	}
}
