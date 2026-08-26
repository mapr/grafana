package informer

import (
	"context"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/storage/unified/resource"
	"github.com/grafana/grafana/pkg/storage/unified/resourcepb"
)

// keysListerPageLimit is the max keys_only page size (server cap).
const keysListerPageLimit = 10000

// KeysLister lists a kind's keys — identity only, no body — for an informer
// re-list; the controller re-fetches the body on demand.
type KeysLister interface {
	ListKeys(ctx context.Context) ([]runtime.Object, int64, error)
}

// grpcKeysLister reads keys from unified storage over gRPC. newKey builds the
// kind's minimal object from a listed key.
type grpcKeysLister struct {
	store  resourcepb.ResourceStoreClient
	gvr    schema.GroupVersionResource
	newKey func(namespace, name, rv string) runtime.Object
}

// NewGRPCKeysLister returns a KeysLister backed by the unified-storage gRPC client.
func NewGRPCKeysLister(store resourcepb.ResourceStoreClient, gvr schema.GroupVersionResource, newKey func(namespace, name, rv string) runtime.Object) KeysLister {
	return grpcKeysLister{store: store, gvr: gvr, newKey: newKey}
}

func (l grpcKeysLister) ListKeys(ctx context.Context) ([]runtime.Object, int64, error) {
	// The in-process client reads identity from ctx; the background re-list has none, so set the service identity ("*").
	ctx = identity.WithServiceIdentityContext(ctx, 1)
	var objs []runtime.Object
	var listRV int64
	var token string
	for {
		resp, err := l.store.List(ctx, &resourcepb.ListRequest{
			KeysOnly:      true,
			Limit:         keysListerPageLimit,
			NextPageToken: token,
			Options: &resourcepb.ListOptions{
				Key: &resourcepb.ResourceKey{Group: l.gvr.Group, Resource: l.gvr.Resource},
			},
		})
		if err := resource.ErrorFromResponse(resp.GetError(), err); err != nil {
			return nil, 0, err
		}
		listRV = resp.GetResourceVersion()
		for _, it := range resp.GetItems() {
			objs = append(objs, l.newKey(it.GetNamespace(), it.GetName(), strconv.FormatInt(it.GetResourceVersion(), 10)))
		}
		if token = resp.GetNextPageToken(); token == "" {
			return objs, listRV, nil
		}
	}
}
