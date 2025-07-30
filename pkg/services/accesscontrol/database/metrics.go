package database

import (
	"context"
	"maps"

	"github.com/grafana/grafana/pkg/infra/db"
)

// countZanzanaTuples returns the number of tuples stored with zanzana
func (ac *AccessControlStore) countZanzanaTuples(ctx context.Context) (map[string]interface{}, error) {
	// TODO figure out where these metrics are exposed... don't seem to be on http://localhost:3000/metrics
	// TODO test this with standalone mode as well (or embedded, idk what i'm running now)
	query := `SELECT COUNT(*) as count FROM tuple` // TODO maybe don't need the alias

	var result struct {
		Count int64
	}

	err := ac.sql.WithDbSession(ctx, func(sess *db.Session) error {
		if _, err := sess.SQL(query).Get(&result); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"stats.enterprise.accesscontrol.zanzana.tuples.count": result.Count,
	}, nil
}

func (ac *AccessControlStore) GetUsageStats(ctx context.Context) map[string]interface{} {
	metricsMap := make(map[string]interface{})
	collectFuncs := map[string]func(context.Context) (map[string]any, error){
		"countZanzanaTuples": ac.countZanzanaTuples,
	}

	for name, fn := range collectFuncs {
		stats, err := fn(ctx)
		if err != nil {
			ac.logger.Error("error in func %s: %e", name, err)
			continue
		}
		maps.Copy(metricsMap, stats)
	}
	return metricsMap
}
