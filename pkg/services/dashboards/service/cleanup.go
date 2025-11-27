package service

import (
	"context"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/folder"
)

type (
	logicalCleanup struct {
		foldersByTitle      map[string][]dashboards.DashboardSearchProjection
		dashboardsByTitle   map[string][]dashboards.DashboardSearchProjection
		foldersToDashboards map[string][]dashboards.DashboardSearchProjection
	}

	cleanupStep interface {
		Run(ctx context.Context, orgID int64, service *DashboardServiceImpl) error
	}

	provisionedDashboardMove struct {
		dashboardID  int64
		newFolderUID string
	}

	provisionedFolderDelete struct {
		folderUID string
	}

	provisionedDashboardDelete struct {
		dashboardID int64
	}
)

func logicalCleanupSteps(resources []dashboards.DashboardSearchProjection) []cleanupStep {
	foldersByTitle := map[string][]dashboards.DashboardSearchProjection{}
	dashboardsByTitle := map[string][]dashboards.DashboardSearchProjection{}
	foldersToDashboards := map[string][]dashboards.DashboardSearchProjection{}

	for _, r := range resources {
		if r.IsFolder {
			foldersByTitle[r.Title] = append(foldersByTitle[r.Title], r)
		} else {
			dashboardsByTitle[r.Title] = append(dashboardsByTitle[r.Title], r)
			foldersToDashboards[r.FolderUID] = append(foldersToDashboards[r.FolderUID], r)
		}
	}

	var steps []cleanupStep
	for _, folders := range foldersByTitle {
		keepFolder := folders[0]
		for _, duplicateFolder := range folders[1:] {
			for _, dashboard := range foldersToDashboards[duplicateFolder.UID] {
				steps = append(steps, provisionedDashboardMove{
					dashboardID:  dashboard.ID,
					newFolderUID: keepFolder.UID,
				})
			}
		}

		for _, duplicateFolder := range folders[1:] {
			steps = append(steps, provisionedFolderDelete{
				folderUID: duplicateFolder.UID,
			})
		}
	}

	for _, dashboards := range dashboardsByTitle {
		for _, dashboard := range dashboards[1:] {
			steps = append(steps, provisionedDashboardDelete{
				dashboardID: dashboard.ID,
			})
		}
	}

	return steps
}

func (p provisionedDashboardMove) Run(ctx context.Context, orgID int64, service *DashboardServiceImpl) error {
	if true {
		return nil
	}
	dashboard, err := service.GetDashboard(ctx, &dashboards.GetDashboardQuery{
		ID: p.dashboardID,
	})
	if err != nil {
		return err
	}

	ctx, ident := identity.WithServiceIdentity(ctx, orgID)
	dashboard.FolderUID = p.newFolderUID
	dto := &dashboards.SaveDashboardDTO{
		OrgID:     orgID,
		User:      ident,
		Dashboard: dashboard,
	}

	cmd, err := service.BuildSaveDashboardCommand(ctx, dto, true)
	if err != nil {
		return err
	}

	_, err = service.saveDashboard(ctx, cmd)
	return err
}

func (p provisionedFolderDelete) Run(ctx context.Context, orgID int64, service *DashboardServiceImpl) error {
	if true {
		return nil
	}
	ctx, ident := identity.WithServiceIdentity(ctx, orgID)
	return service.folderService.Delete(ctx, &folder.DeleteFolderCommand{
		UID:          p.folderUID,
		OrgID:        orgID,
		SignedInUser: ident,
	})
}

func (p provisionedDashboardDelete) Run(ctx context.Context, orgID int64, service *DashboardServiceImpl) error {
	if true {
		return nil
	}
	return service.DeleteProvisionedDashboard(ctx, p.dashboardID, orgID)
}
