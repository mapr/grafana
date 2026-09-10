package foldermetadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	provisioning "github.com/grafana/grafana/apps/provisioning/pkg/apis/provisioning/v0alpha1"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/grafana/pkg/tests/apis/provisioning/common"
)

func TestIntegrationProvisioning_DeleteRepositoryPreservesSharedFolder(t *testing.T) {
	helper := sharedHelper(t)
	const repo = "delete-shared-folder-repo"
	folder, managed, unmanaged := createMixedOwnership(t, helper, repo)

	require.NoError(t, helper.Repositories.Resource.Delete(t.Context(), repo, metav1.DeleteOptions{}))
	helper.WaitForRepositoryDeleted(t, repo)

	requireSharedFolderPreserved(t, helper, folder, managed, unmanaged)
}

func TestIntegrationProvisioning_DeleteResourcesPreservesSharedFolder(t *testing.T) {
	helper := sharedHelper(t)
	const repo = "delete-shared-folder-job-repo"
	folder, managed, unmanaged := createMixedOwnership(t, helper, repo)

	_, err := helper.Repositories.Resource.Patch(t.Context(), repo, types.JSONPatchType, []byte(`[
		{"op":"replace","path":"/metadata/finalizers","value":["cleanup"]}
	]`), metav1.PatchOptions{})
	require.NoError(t, err)
	require.NoError(t, helper.Repositories.Resource.Delete(t.Context(), repo, metav1.DeleteOptions{}))
	helper.WaitForRepositoryDeleted(t, repo)

	job := helper.TriggerJobAndWaitForComplete(t, repo, provisioning.JobSpec{
		Action:     provisioning.JobActionDeleteResources,
		Repository: repo,
	})
	common.HasNoErrors()(t, job)
	common.HasState(provisioning.JobStateWarning)(t, job)

	requireSharedFolderPreserved(t, helper, folder, managed, unmanaged)
}

func createMixedOwnership(t *testing.T, helper *common.ProvisioningTestHelper, repo string) (string, string, string) {
	t.Helper()
	folder := helper.CreateUnmanagedFolder(t, "Shared migration folder", "")
	managed := helper.CreateUnmanagedDashboard(t, "Managed dashboard", folder)
	unmanaged := helper.CreateUnmanagedDashboard(t, "Unmanaged dashboard", folder)

	helper.CreateLocalRepo(t, common.TestRepo{
		Name:       repo,
		SyncTarget: "folderless",
		Workflows:  []string{"write"},
		Copies:     map[string]string{},
	})
	helper.TriggerJobAndWaitForSuccess(t, repo, provisioning.JobSpec{
		Action: provisioning.JobActionMigrate,
		Migrate: &provisioning.MigrateJobOptions{
			Resources: []provisioning.ResourceRef{{
				Name: managed, Kind: "Dashboard", Group: "dashboard.grafana.app",
			}},
		},
	})

	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		folderObject, err := helper.Folders.Resource.Get(t.Context(), folder, metav1.GetOptions{})
		if !assert.NoError(collect, err) {
			return
		}
		assert.Equal(collect, string(utils.ManagerKindRepo), folderObject.GetAnnotations()[utils.AnnoKeyManagerKind])
		assert.Equal(collect, repo, folderObject.GetAnnotations()[utils.AnnoKeyManagerIdentity])

		managedObject, err := helper.DashboardsV1.Resource.Get(t.Context(), managed, metav1.GetOptions{})
		if !assert.NoError(collect, err) {
			return
		}
		assert.Equal(collect, string(utils.ManagerKindRepo), managedObject.GetAnnotations()[utils.AnnoKeyManagerKind])
		assert.Equal(collect, repo, managedObject.GetAnnotations()[utils.AnnoKeyManagerIdentity])

		unmanagedObject, err := helper.DashboardsV1.Resource.Get(t.Context(), unmanaged, metav1.GetOptions{})
		if !assert.NoError(collect, err) {
			return
		}
		assert.Empty(collect, unmanagedObject.GetAnnotations()[utils.AnnoKeyManagerKind])
		assert.Empty(collect, unmanagedObject.GetAnnotations()[utils.AnnoKeyManagerIdentity])
	}, common.WaitTimeoutDefault, common.WaitIntervalDefault)

	return folder, managed, unmanaged
}

func requireSharedFolderPreserved(t *testing.T, helper *common.ProvisioningTestHelper, folder, managed, unmanaged string) {
	t.Helper()
	helper.RequireDashboardsNotFound(t, managed)

	folderObject, err := helper.Folders.Resource.Get(t.Context(), folder, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotContains(t, folderObject.GetAnnotations(), utils.AnnoKeyManagerKind)
	require.NotContains(t, folderObject.GetAnnotations(), utils.AnnoKeyManagerIdentity)
	require.NotContains(t, folderObject.GetAnnotations(), utils.AnnoKeySourcePath)
	require.NotContains(t, folderObject.GetAnnotations(), utils.AnnoKeySourceChecksum)

	unmanagedObject, err := helper.DashboardsV1.Resource.Get(t.Context(), unmanaged, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, folder, unmanagedObject.GetAnnotations()[utils.AnnoKeyFolder])
	require.Empty(t, unmanagedObject.GetAnnotations()[utils.AnnoKeyManagerKind])
	require.Empty(t, unmanagedObject.GetAnnotations()[utils.AnnoKeyManagerIdentity])
}
