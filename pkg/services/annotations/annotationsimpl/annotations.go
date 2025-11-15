package annotationsimpl

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/grafana/pkg/services/annotations/accesscontrol"
	"github.com/grafana/grafana/pkg/services/annotations/annotationsimpl/loki"
	"github.com/grafana/grafana/pkg/services/annotations/annotationsimpl/loki_annotations"
	"github.com/grafana/grafana/pkg/services/dashboards"
	alertingStore "github.com/grafana/grafana/pkg/services/ngalert/store"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/infra/tracing"
	"github.com/grafana/grafana/pkg/services/annotations"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/tag"
	"github.com/grafana/grafana/pkg/setting"
)

type RepositoryImpl struct {
	db       db.DB
	authZ    *accesscontrol.AuthService
	features featuremgmt.FeatureToggles
	reader   readStore
	writer   writeStore
}

func ProvideService(
	db db.DB,
	cfg *setting.Cfg,
	features featuremgmt.FeatureToggles,
	tagService tag.Service,
	tracer tracing.Tracer,
	ruleStore *alertingStore.DBstore,
	dashSvc dashboards.DashboardService,
	reg prometheus.Registerer,
) *RepositoryImpl {
	l := log.New("annotations")
	l.Debug("Initializing annotations service")

	var write writeStore
	var read readStore

	// Try to create Loki store first (default)
	lokiStore, err := loki_annotations.NewLokiAnnotationsStore(cfg.AnnotationsLoki, log.New("annotations.loki"), tracer, reg)
	if err != nil {
		l.Warn("Failed to initialize Loki annotations store", "error", err)
	}
	if lokiStore != nil {
		l.Debug("Using Loki annotations store")
		write = lokiStore
		read = lokiStore
	} else {
		// Fallback to SQL store
		l.Debug("Using SQL annotations store")
		xormStore := NewXormStore(cfg, log.New("annotations.sql"), db, tagService, reg)
		write = xormStore
		read = xormStore

		// Check if we should also use historian store for reading
		historianStore := loki.NewLokiHistorianStore(cfg.UnifiedAlerting.StateHistory, db, ruleStore, log.New("annotations.loki"), tracer, reg)
		if historianStore != nil {
			l.Debug("Using composite read store (SQL + Historian)")
			read = NewCompositeStore(log.New("annotations.composite"), xormStore, historianStore)
		}
	}

	// Create repository first (we need it for AuthService)
	repo := &RepositoryImpl{
		db:       db,
		features: features,
		reader:   read,
		writer:   write,
	}

	// Create AuthService with reader reference (to avoid circular dependency)
	// AuthService needs to read annotations directly without going through Repository.Find
	repo.authZ = accesscontrol.NewAuthService(db, features, dashSvc, cfg, read)

	return repo
}

func (r *RepositoryImpl) Save(ctx context.Context, item *annotations.Item) error {
	return r.writer.Add(ctx, item)
}

// SaveMany inserts multiple annotations at once.
// It does not return IDs associated with created annotations. If you need this functionality, use the single-item Save instead.
func (r *RepositoryImpl) SaveMany(ctx context.Context, items []annotations.Item) error {
	return r.writer.AddMany(ctx, items)
}

func (r *RepositoryImpl) Update(ctx context.Context, item *annotations.Item) error {
	return r.writer.Update(ctx, item)
}

func (r *RepositoryImpl) Find(ctx context.Context, query *annotations.ItemQuery) ([]*annotations.ItemDTO, error) {
	if query.Limit == 0 {
		query.Limit = 100
	}

	// Search without dashboard UID filter is expensive, so check without access control first
	// nolint: staticcheck
	if query.DashboardID == 0 && query.DashboardUID == "" {
		// Return early if no annotations found, it's not necessary to perform expensive access control filtering
		res, err := r.reader.Get(ctx, *query, &accesscontrol.AccessResources{
			SkipAccessControlFilter: true,
		})
		if err != nil || len(res) == 0 {
			return []*annotations.ItemDTO{}, err
		}
		// If number of resources is less than limit, it makes sense to set query limit to this
		// value, otherwise query will be iterating over all user's dashboards since original
		// query limit is never reached.
		query.Limit = int64(len(res))
	}

	results := make([]*annotations.ItemDTO, 0, query.Limit)
	query.Page = 1

	// Iterate over available annotations until query limit is reached
	// or all available dashboards are checked
	for len(results) < int(query.Limit) {
		resources, err := r.authZ.Authorize(ctx, *query)
		if err != nil {
			return nil, err
		}

		res, err := r.reader.Get(ctx, *query, resources)
		if err != nil {
			return nil, err
		}

		results = append(results, res...)
		query.Page++
		// All user's dashboards are fetched
		if len(resources.Dashboards) < int(query.Limit) {
			break
		}
	}

	return results, nil
}

func (r *RepositoryImpl) Delete(ctx context.Context, params *annotations.DeleteParams) error {
	return r.writer.Delete(ctx, params)
}

func (r *RepositoryImpl) FindTags(ctx context.Context, query *annotations.TagsQuery) (annotations.FindTagsResult, error) {
	return r.reader.GetTags(ctx, *query)
}
