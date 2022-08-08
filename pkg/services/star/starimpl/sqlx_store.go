package starimpl

import (
	"context"
	"database/sql"
	"errors"

	"github.com/grafana/grafana/pkg/services/sqlstore/db"
	"github.com/grafana/grafana/pkg/services/star"
	"github.com/jmoiron/sqlx"
)

type sqlxStore struct {
	sqlxdb *sqlx.DB
	db     db.DB
}

func (s *sqlxStore) Get(ctx context.Context, query *star.IsStarredByUserQuery) (bool, error) {
	var star_res star.Star
	err := s.sqlxdb.GetContext(ctx, &star_res, s.sqlxdb.Rebind("SELECT * from star where user_id=? and dashboard_id=?"), query.UserID, query.DashboardID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *sqlxStore) Insert(ctx context.Context, cmd *star.StarDashboardCommand) error {
	entity := star.Star{
		UserID:      cmd.UserID,
		DashboardID: cmd.DashboardID,
	}
	_, err := s.sqlxdb.NamedExecContext(ctx, `INSERT INTO star (user_id, dashboard_id) VALUES (:user_id, :dashboard_id)`, entity)
	if err != nil {
		return err
	}
	return err
}

func (s *sqlxStore) Delete(ctx context.Context, cmd *star.UnstarDashboardCommand) error {
	_, err := s.sqlxdb.ExecContext(ctx, s.sqlxdb.Rebind("DELETE FROM star WHERE user_id=? and dashboard_id=?"), cmd.UserID, cmd.DashboardID)
	return err
}

func (s *sqlxStore) DeleteByUser(ctx context.Context, userID int64) error {
	_, err := s.sqlxdb.ExecContext(ctx, s.sqlxdb.Rebind("DELETE FROM star WHERE user_id = ?"), userID)
	return err
}

func (s *sqlxStore) List(ctx context.Context, query *star.GetUserStarsQuery) (*star.GetUserStarsResult, error) {
	userStars := make(map[int64]bool)
	var stars = make([]star.Star, 0)
	err := s.sqlxdb.SelectContext(ctx, &stars, s.sqlxdb.Rebind("SELECT * FROM star WHERE user_id=?"), query.UserID)
	if err != nil {
		return nil, err
	}
	for _, star := range stars {
		userStars[star.DashboardID] = true
	}

	return &star.GetUserStarsResult{UserStars: userStars}, err
}
