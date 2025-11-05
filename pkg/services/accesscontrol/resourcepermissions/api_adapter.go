package resourcepermissions

import (
	"context"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"

	iamv0 "github.com/grafana/grafana/apps/iam/pkg/apis/iam/v0alpha1"
	"github.com/grafana/grafana/pkg/api/dtos"
	"github.com/grafana/grafana/pkg/services/user"
)

func (a *api) getResourcePermissionsFromK8s(ctx context.Context, namespace string, resourceID string) (getResourcePermissionsResponse, error) {
	if a.restConfigProvider == nil {
		return nil, fmt.Errorf("k8s rest config provider not available")
	}

	restConfig, err := a.restConfigProvider.GetRestConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get rest config: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	resourcePermName := a.buildResourcePermissionName(resourceID)

	resourcePermResource := dynamicClient.Resource(iamv0.ResourcePermissionInfo.GroupVersionResource()).Namespace(namespace)
	resourcePerm, err := resourcePermResource.Get(ctx, resourcePermName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Return empty list if not found (no permissions set)
			return getResourcePermissionsResponse{}, nil
		}
		return nil, fmt.Errorf("failed to get resource permission from k8s: %w", err)
	}

	return a.convertK8sResourcePermissionToDTO(resourcePerm)
}

func (a *api) convertK8sResourcePermissionToDTO(resourcePerm *unstructured.Unstructured) (getResourcePermissionsResponse, error) {
	permissions, found, err := unstructured.NestedSlice(resourcePerm.Object, "spec", "permissions")
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions from spec: %w", err)
	}
	if !found {
		return getResourcePermissionsResponse{}, nil
	}

	dto := make(getResourcePermissionsResponse, 0, len(permissions))

	for _, permRaw := range permissions {
		permMap, ok := permRaw.(map[string]interface{})
		if !ok {
			continue
		}

		kind, _, _ := unstructured.NestedString(permMap, "kind")
		name, _, _ := unstructured.NestedString(permMap, "name")
		verb, _, _ := unstructured.NestedString(permMap, "verb")

		if name == "" || verb == "" {
			continue
		}

		permission := verb

		actions, exists := a.service.options.PermissionsToActions[verb]
		if !exists {
			actions = []string{verb}
		}

		permDTO := resourcePermissionDTO{
			Permission: permission,
			Actions:    actions,
		}

		switch iamv0.ResourcePermissionSpecPermissionKind(kind) {
		case iamv0.ResourcePermissionSpecPermissionKindUser, iamv0.ResourcePermissionSpecPermissionKindServiceAccount:
			// Fetch user details
			userDetails, err := a.service.userService.GetByUID(context.Background(), &user.GetUserByUIDQuery{UID: name})
			if err == nil {
				permDTO.UserID = userDetails.ID
				permDTO.UserUID = userDetails.UID
				permDTO.UserLogin = userDetails.Login
				permDTO.UserAvatarUrl = dtos.GetGravatarUrl(a.cfg, userDetails.Email)
				permDTO.IsServiceAccount = userDetails.IsServiceAccount
			}
		case iamv0.ResourcePermissionSpecPermissionKindTeam:
			permDTO.TeamUID = name
			permDTO.Team = name
		case iamv0.ResourcePermissionSpecPermissionKindBasicRole:
			permDTO.BuiltInRole = name
		}

		permDTO.RoleName = fmt.Sprintf("managed:%s:%s:permissions", a.service.options.Resource, name)

		dto = append(dto, permDTO)
	}

	return dto, nil
}

// buildResourcePermissionName constructs the resource permission name for k8s API
// Format: <api-group>.<resource>.<resource-uid>
// Example: folder.grafana.app.folders.folder-abc123
func (a *api) buildResourcePermissionName(resourceID string) string {
	apiGroup := ""

	switch a.service.options.Resource {
	case "folders":
		apiGroup = "folder.grafana.app"
	case "dashboards":
		apiGroup = "dashboard.grafana.app"
	default:
		apiGroup = fmt.Sprintf("%s.grafana.app", a.service.options.Resource)
	}

	return fmt.Sprintf("%s-%s-%s", apiGroup, a.service.options.Resource, resourceID)
}
