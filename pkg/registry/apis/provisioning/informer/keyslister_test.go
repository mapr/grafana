package informer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	provisioningapis "github.com/grafana/grafana/apps/provisioning/pkg/apis/provisioning/v0alpha1"
	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/storage/unified/resourcepb"
)

// fakeStoreClient serves canned List pages and records the requests, so the gRPC
// keys lister can be exercised without a storage backend. Embedding the
// interface supplies the other (unused) methods.
type fakeStoreClient struct {
	resourcepb.ResourceStoreClient
	pages              []*resourcepb.ListResponse
	reqs               []*resourcepb.ListRequest
	sawServiceIdentity bool
}

func (f *fakeStoreClient) List(ctx context.Context, in *resourcepb.ListRequest, _ ...grpc.CallOption) (*resourcepb.ListResponse, error) {
	if len(f.reqs) == 0 {
		f.sawServiceIdentity = identity.IsServiceIdentity(ctx) // else the in-process client rejects it: "no claims found"
	}
	f.reqs = append(f.reqs, in)
	return f.pages[len(f.reqs)-1], nil
}

func TestGRPCConnectionKeysLister(t *testing.T) {
	fake := &fakeStoreClient{pages: []*resourcepb.ListResponse{
		{
			Items: []*resourcepb.ResourceWrapper{
				{Namespace: "ns1", Name: "a", ResourceVersion: 10},
				{Namespace: "ns2", Name: "b", ResourceVersion: 11},
			},
			NextPageToken:   "tok",
			ResourceVersion: 100,
		},
		{
			Items:           []*resourcepb.ResourceWrapper{{Namespace: "ns1", Name: "c", ResourceVersion: 12}},
			NextPageToken:   "",
			ResourceVersion: 100,
		},
	}}

	objs, listRV, err := NewGRPCConnectionKeysLister(fake).ListKeys(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int64(100), listRV, "the snapshot resourceVersion is returned")

	assert.True(t, fake.sawServiceIdentity, "the re-list must authenticate as the service identity, or the in-process client rejects it with \"no claims found\"")

	require.Len(t, fake.reqs, 2)
	first := fake.reqs[0]
	assert.True(t, first.KeysOnly, "keys_only is set")
	assert.Empty(t, first.Options.Key.Namespace, "keys_only lists cluster-wide")
	assert.Equal(t, "connections", first.Options.Key.Resource)
	assert.Equal(t, "", first.NextPageToken, "first page carries no token")
	assert.Equal(t, "tok", fake.reqs[1].NextPageToken, "the continue token is forwarded")

	require.Len(t, objs, 3, "items accumulate across pages")

	// Each object is a minimal, label-less Connection carrying only identity —
	// the property the controller's re-fetch depends on.
	first0, ok := objs[0].(*provisioningapis.Connection)
	require.True(t, ok, "expected a *Connection, got %T", objs[0])
	assert.Equal(t, "ns1", first0.Namespace)
	assert.Equal(t, "a", first0.Name)
	assert.Equal(t, "10", first0.ResourceVersion)
	assert.Nil(t, first0.Labels, "keys-only objects carry no labels")
}
