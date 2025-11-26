package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	claims "github.com/grafana/authlib/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/codes"

	"github.com/grafana/grafana-app-sdk/logging"
	secretv1beta1 "github.com/grafana/grafana/apps/secret/pkg/apis/secret/v1beta1"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/grafana/pkg/registry/apis/secret/contracts"
	"github.com/grafana/grafana/pkg/registry/apis/secret/service/metrics"
	"github.com/grafana/grafana/pkg/registry/apis/secret/xkube"
)

var _ contracts.SecureValueService = (*SecureValueService)(nil)

type SecureValueService struct {
	tracer                     trace.Tracer
	accessClient               claims.AccessClient
	database                   contracts.Database
	secureValueMetadataStorage contracts.SecureValueMetadataStorage
	secureValueValidator       contracts.SecureValueValidator
	secureValueMutator         contracts.SecureValueMutator
	keeperConfigReader         contracts.KeeperConfigReader
	keeperService              contracts.KeeperService
	metrics                    *metrics.SecureValueServiceMetrics
}

func ProvideSecureValueService(
	tracer trace.Tracer,
	accessClient claims.AccessClient,
	database contracts.Database,
	secureValueMetadataStorage contracts.SecureValueMetadataStorage,
	secureValueValidator contracts.SecureValueValidator,
	secureValueMutator contracts.SecureValueMutator,
	keeperConfigReader contracts.KeeperConfigReader,
	keeperService contracts.KeeperService,
	reg prometheus.Registerer,
) contracts.SecureValueService {
	return &SecureValueService{
		tracer:                     tracer,
		accessClient:               accessClient,
		database:                   database,
		secureValueMetadataStorage: secureValueMetadataStorage,
		secureValueValidator:       secureValueValidator,
		secureValueMutator:         secureValueMutator,
		keeperConfigReader:         keeperConfigReader,
		keeperService:              keeperService,
		metrics:                    metrics.NewSecureValueServiceMetrics(reg),
	}
}

func (s *SecureValueService) Create(ctx context.Context, sv *secretv1beta1.SecureValue, actorUID string) (createdSv *secretv1beta1.SecureValue, createErr error) {
	start := time.Now()

	ctx, span := s.tracer.Start(ctx, "SecureValueService.Create", trace.WithAttributes(
		attribute.String("namespace", sv.GetNamespace()),
		attribute.String("actor", actorUID),
	))
	defer span.End()

	defer func() {
		args := []any{
			"namespace", sv.GetNamespace(),
			"actorUID", actorUID,
		}

		if createdSv != nil {
			args = append(args, "name", createdSv.GetName())
			span.SetAttributes(attribute.String("name", createdSv.GetName()))
		}

		success := createErr == nil
		args = append(args, "success", success)
		if !success {
			span.SetStatus(codes.Error, "SecureValueService.Create failed")
			span.RecordError(createErr)
			args = append(args, "error", createErr)
		}

		logging.FromContext(ctx).Info("SecureValueService.Create finished", args...)

		s.metrics.SecureValueCreateDuration.WithLabelValues(strconv.FormatBool(success)).Observe(time.Since(start).Seconds())
	}()

	if err := s.CreateV2(ctx, sv, actorUID); err != nil {
		return nil, err
	}

	read, err := s.Read(ctx, xkube.Namespace(sv.GetNamespace()), sv.GetName())
	if err != nil {
		return nil, fmt.Errorf("reading secure value after create: %w", err)
	}

	return read, nil

	/*
		// Secure value creation uses the active keeper
		keeperName, keeperCfg, err := s.keeperConfigReader.GetActiveKeeperConfig(ctx, sv.Namespace)
		if err != nil {
			return nil, fmt.Errorf("fetching active keeper config: namespace=%+v %w", sv.Namespace, err)
		}

		return s.createNewVersion(ctx, keeperName, keeperCfg, sv, actorUID)
	*/
}

func (s *SecureValueService) Update(ctx context.Context, newSecureValue *secretv1beta1.SecureValue, actorUID string) (_ *secretv1beta1.SecureValue, sync bool, updateErr error) {
	start := time.Now()
	name, namespace := newSecureValue.GetName(), newSecureValue.GetNamespace()

	ctx, span := s.tracer.Start(ctx, "SecureValueService.Update", trace.WithAttributes(
		attribute.String("name", name),
		attribute.String("namespace", namespace),
		attribute.String("actor", actorUID),
	))
	defer span.End()

	defer func() {
		args := []any{
			"name", name,
			"namespace", namespace,
			"actorUID", actorUID,
			"sync", sync,
		}

		success := updateErr == nil
		args = append(args, "success", success)
		if !success {
			span.SetStatus(codes.Error, "SecureValueService.Update failed")
			span.RecordError(updateErr)
			args = append(args, "error", updateErr)
		}

		logging.FromContext(ctx).Info("SecureValueService.Update finished", args...)

		s.metrics.SecureValueUpdateDuration.WithLabelValues(strconv.FormatBool(success)).Observe(time.Since(start).Seconds())
	}()

	sync, err := s.UpdateV2(ctx, newSecureValue, actorUID)
	if err != nil {
		return nil, sync, err
	}

	read, err := s.Read(ctx, xkube.Namespace(namespace), name)
	if err != nil {
		return nil, sync, fmt.Errorf("reading secure value after update: %w", err)
	}

	return read, true, nil

	/*
		currentVersion, err := s.secureValueMetadataStorage.Read(ctx, xkube.Namespace(newSecureValue.Namespace), newSecureValue.Name, contracts.ReadOpts{})
		if err != nil {
			return nil, false, fmt.Errorf("reading secure value secret: %+w", err)
		}

		keeperCfg, err := s.keeperConfigReader.GetKeeperConfig(ctx, currentVersion.Namespace, currentVersion.Status.Keeper, contracts.ReadOpts{})
		if err != nil {
			return nil, false, fmt.Errorf("fetching keeper config: namespace=%+v keeper: %q %w", newSecureValue.Namespace, currentVersion.Status.Keeper, err)
		}

		if newSecureValue.Spec.Value == nil {
			keeper, err := s.keeperService.KeeperForConfig(keeperCfg)
			if err != nil {
				return nil, false, fmt.Errorf("getting keeper for config: namespace=%+v keeperName=%+v %w", newSecureValue.Namespace, newSecureValue.Status.Keeper, err)
			}
			logging.FromContext(ctx).Debug("retrieved keeper", "namespace", newSecureValue.Namespace, "type", keeperCfg.Type())

			secret, err := keeper.Expose(ctx, keeperCfg, xkube.Namespace(newSecureValue.Namespace), newSecureValue.Name, currentVersion.Status.Version)
			if err != nil {
				return nil, false, fmt.Errorf("reading secret value from keeper: %w", err)
			}

			newSecureValue.Spec.Value = &secret
		}

		// Secure value updates use the keeper used to create the secure value
		const updateIsSync = true
		createdSv, err := s.createNewVersion(ctx, currentVersion.Status.Keeper, keeperCfg, newSecureValue, actorUID)
		return createdSv, updateIsSync, err
	*/
}

func (s *SecureValueService) createNewVersion(ctx context.Context, keeperName string, keeperCfg secretv1beta1.KeeperConfig, sv *secretv1beta1.SecureValue, actorUID string) (*secretv1beta1.SecureValue, error) {
	if keeperName == "" {
		return nil, fmt.Errorf("keeper name is required, got empty string")
	}
	if err := s.secureValueMutator.Mutate(sv, admission.Create); err != nil {
		return nil, err
	}

	if errorList := s.secureValueValidator.Validate(sv, nil, admission.Create); len(errorList) > 0 {
		return nil, contracts.NewErrValidateSecureValue(errorList)
	}

	createdSv, err := s.secureValueMetadataStorage.Create(ctx, keeperName, sv, actorUID)
	if err != nil {
		return nil, fmt.Errorf("creating secure value: %w", err)
	}

	createdSv.Status = secretv1beta1.SecureValueStatus{
		Version: createdSv.Status.Version,
		Keeper:  keeperName,
	}

	keeper, err := s.keeperService.KeeperForConfig(keeperCfg)
	if err != nil {
		return nil, fmt.Errorf("getting keeper for config: namespace=%+v keeperName=%+v %w", createdSv.Namespace, keeperName, err)
	}
	logging.FromContext(ctx).Debug("retrieved keeper", "namespace", createdSv.Namespace, "type", keeperCfg.Type())

	// TODO: can we stop using external id?
	// TODO: store uses only the namespace and returns and id. It could be a kv instead.
	// TODO: check that the encrypted store works with multiple versions
	externalID, err := keeper.Store(ctx, keeperCfg, xkube.Namespace(createdSv.Namespace), createdSv.Name, createdSv.Status.Version, sv.Spec.Value.DangerouslyExposeAndConsumeValue())
	if err != nil {
		return nil, fmt.Errorf("storing secure value in keeper: %w", err)
	}
	createdSv.Status.ExternalID = string(externalID)

	if err := s.secureValueMetadataStorage.SetExternalID(ctx, xkube.Namespace(createdSv.Namespace), createdSv.Name, createdSv.Status.Version, externalID); err != nil {
		return nil, fmt.Errorf("setting secure value external id: %w", err)
	}

	if err := s.secureValueMetadataStorage.SetVersionToActive(ctx, xkube.Namespace(createdSv.Namespace), createdSv.Name, createdSv.Status.Version); err != nil {
		return nil, fmt.Errorf("marking secure value version as active: %w", err)
	}

	// In a single query:
	// TODO: set external id
	// TODO: set to active

	return createdSv, nil
}

func (s *SecureValueService) Read(ctx context.Context, namespace xkube.Namespace, name string) (_ *secretv1beta1.SecureValue, readErr error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace cannot be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	start := time.Now()

	ctx, span := s.tracer.Start(ctx, "SecureValueService.Read", trace.WithAttributes(
		attribute.String("name", name),
		attribute.String("namespace", namespace.String()),
	))

	defer func() {
		args := []any{
			"name", name,
			"namespace", namespace.String(),
		}

		success := readErr == nil
		args = append(args, "success", success)
		if !success {
			span.SetStatus(codes.Error, "SecureValueService.Read failed")
			span.RecordError(readErr)
			args = append(args, "error", readErr)
		}

		logging.FromContext(ctx).Info("SecureValueService.Read finished", args...)

		s.metrics.SecureValueReadDuration.WithLabelValues(strconv.FormatBool(success)).Observe(time.Since(start).Seconds())
	}()

	defer span.End()

	return s.secureValueMetadataStorage.Read(ctx, namespace, name, contracts.ReadOpts{ForUpdate: false})
}

func (s *SecureValueService) List(ctx context.Context, namespace xkube.Namespace) (_ *secretv1beta1.SecureValueList, listErr error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace cannot be empty")
	}

	start := time.Now()

	ctx, span := s.tracer.Start(ctx, "SecureValueService.List", trace.WithAttributes(
		attribute.String("namespace", namespace.String()),
	))
	defer span.End()

	defer func() {
		args := []any{
			"namespace", namespace,
		}

		success := listErr == nil
		args = append(args, "success", success)
		if !success {
			span.SetStatus(codes.Error, "SecureValueService.List failed")
			span.RecordError(listErr)
			args = append(args, "error", listErr)
		}

		logging.FromContext(ctx).Info("SecureValueService.List finished", args...)

		s.metrics.SecureValueListDuration.WithLabelValues(strconv.FormatBool(success)).Observe(time.Since(start).Seconds())
	}()

	user, ok := claims.AuthInfoFrom(ctx)
	if !ok {
		return nil, fmt.Errorf("missing auth info in context")
	}

	hasPermissionFor, _, err := s.accessClient.Compile(ctx, user, claims.ListRequest{
		Group:     secretv1beta1.APIGroup,
		Resource:  secretv1beta1.SecureValuesResourceInfo.GetName(),
		Namespace: namespace.String(),
		Verb:      utils.VerbGet, // Why not VerbList?
	})
	if err != nil {
		return nil, fmt.Errorf("failed to compile checker: %w", err)
	}

	secureValuesMetadata, err := s.secureValueMetadataStorage.List(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("fetching secure values from storage: %+w", err)
	}

	out := make([]secretv1beta1.SecureValue, 0)

	for _, metadata := range secureValuesMetadata {
		// Check whether the user has permission to access this specific SecureValue in the namespace.
		if !hasPermissionFor(metadata.Name, "") {
			continue
		}

		out = append(out, metadata)
	}

	return &secretv1beta1.SecureValueList{
		Items: out,
	}, nil
}

func (s *SecureValueService) Delete(ctx context.Context, namespace xkube.Namespace, name string) (_ *secretv1beta1.SecureValue, deleteErr error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace cannot be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	start := time.Now()

	ctx, span := s.tracer.Start(ctx, "SecureValueService.Delete", trace.WithAttributes(
		attribute.String("name", name),
		attribute.String("namespace", namespace.String()),
	))
	defer span.End()

	defer func() {
		args := []any{
			"name", name,
			"namespace", namespace,
		}

		success := deleteErr == nil
		args = append(args, "success", success)
		if !success {
			span.SetStatus(codes.Error, "SecureValueService.Delete failed")
			span.RecordError(deleteErr)
			args = append(args, "error", deleteErr)
		}

		logging.FromContext(ctx).Info("SecureValueService.Delete finished", args...)

		s.metrics.SecureValueDeleteDuration.WithLabelValues(strconv.FormatBool(success)).Observe(time.Since(start).Seconds())
	}()

	// TODO: does this need to be for update?
	sv, err := s.secureValueMetadataStorage.Read(ctx, namespace, name, contracts.ReadOpts{ForUpdate: true})
	if err != nil {
		return nil, fmt.Errorf("fetching secure value: %+w", err)
	}

	/*
		if err := s.secureValueMetadataStorage.SetVersionToInactive(ctx, namespace, name, sv.Status.Version); err != nil {
			return nil, fmt.Errorf("setting secure value version to inactive: %+w", err)
		}
	*/

	if err := s.DeleteV2(ctx, namespace, name); err != nil {
		return nil, fmt.Errorf("deleting secure value from keeper: %+w", err)
	}

	return sv, nil
}

// V2 stuff
func (s *SecureValueService) CreateV2(ctx context.Context, sv *secretv1beta1.SecureValue, actorUID string) error {
	keeperName, keeperCfg, err := s.keeperConfigReader.GetActiveKeeperConfig(ctx, sv.Namespace)
	if err != nil {
		return fmt.Errorf("fetching active keeper config: namespace=%+v %w", sv.Namespace, err)
	}

	return s.createNewVersionV2(ctx, keeperName, keeperCfg, sv, actorUID)
}

func (s *SecureValueService) UpdateV2(ctx context.Context, newSecureValue *secretv1beta1.SecureValue, actorUID string) (bool, error) {
	currentVersion, err := s.secureValueMetadataStorage.Read(ctx, xkube.Namespace(newSecureValue.Namespace), newSecureValue.Name, contracts.ReadOpts{})
	if err != nil {
		return false, fmt.Errorf("reading secure value secret: %+w", err)
	}

	keeperCfg, err := s.keeperConfigReader.GetKeeperConfig(ctx, currentVersion.Namespace, currentVersion.Status.Keeper, contracts.ReadOpts{})
	if err != nil {
		return false, fmt.Errorf("fetching keeper config: namespace=%+v keeper: %q %w", newSecureValue.Namespace, currentVersion.Status.Keeper, err)
	}

	if newSecureValue.Spec.Value == nil {
		keeper, err := s.keeperService.KeeperForConfig(keeperCfg)
		if err != nil {
			return false, fmt.Errorf("getting keeper for config: namespace=%+v keeperName=%+v %w", newSecureValue.Namespace, newSecureValue.Status.Keeper, err)
		}
		logging.FromContext(ctx).Debug("retrieved keeper", "namespace", newSecureValue.Namespace, "type", keeperCfg.Type())

		secret, err := keeper.Expose(ctx, keeperCfg, xkube.Namespace(newSecureValue.Namespace), newSecureValue.Name, currentVersion.Status.Version)
		if err != nil {
			return false, fmt.Errorf("reading secret value from keeper: %w", err)
		}

		newSecureValue.Spec.Value = &secret
	}

	// Secure value updates use the keeper used to create the secure value
	return true, s.createNewVersionV2(ctx, currentVersion.Status.Keeper, keeperCfg, newSecureValue, actorUID)
}

func (s *SecureValueService) createNewVersionV2(ctx context.Context, keeperName string, keeperCfg secretv1beta1.KeeperConfig, sv *secretv1beta1.SecureValue, actorUID string) error {
	if keeperName == "" {
		return fmt.Errorf("keeper name is required, got empty string")
	}
	if err := s.secureValueMutator.Mutate(sv, admission.Create); err != nil {
		return err
	}

	if errorList := s.secureValueValidator.Validate(sv, nil, admission.Create); len(errorList) > 0 {
		return contracts.NewErrValidateSecureValue(errorList)
	}

	version := int64(1)
	created := time.Now().UTC().Unix()

	read, err := s.secureValueMetadataStorage.Read(ctx, xkube.Namespace(sv.Namespace), sv.Name, contracts.ReadOpts{})
	if err != nil && !errors.Is(err, contracts.ErrSecureValueNotFound) {
		return fmt.Errorf("read secure value for created: %w", err)
	}
	if read != nil {
		version = read.Status.Version + 1
		created = read.GetCreationTimestamp().UTC().Unix()
	}

	meta, err := utils.MetaAccessor(sv)
	if err != nil {
		return fmt.Errorf("failed to get meta accessor: %w", err)
	}
	if meta.GetFolder() != "" {
		return fmt.Errorf("folders are not supported")
	}

	var (
		ownerReferenceAPIGroup   *string
		ownerReferenceAPIVersion *string
		ownerReferenceKind       *string
		ownerReferenceName       *string
	)

	ownerReferences := meta.GetOwnerReferences()
	if len(ownerReferences) > 1 {
		return fmt.Errorf("only one owner reference is supported, found %d", len(ownerReferences))
	}
	if len(ownerReferences) == 1 {
		ownerReference := ownerReferences[0]

		gv, err := schema.ParseGroupVersion(ownerReference.APIVersion)
		if err != nil {
			return fmt.Errorf("failed to parse owner reference API version %s: %w", ownerReference.APIVersion, err)
		}
		if gv.Group == "" {
			return fmt.Errorf("malformed api version %s requires <group>/<version> format", ownerReference.APIVersion)
		}

		ownerReferenceAPIGroup = &gv.Group
		ownerReferenceAPIVersion = &gv.Version
		ownerReferenceKind = &ownerReference.Kind
		ownerReferenceName = &ownerReference.Name
	}

	sv2 := &contracts.SecureValueMetadataModel{
		GUID:                     uuid.New().String(),
		Name:                     sv.Name,
		Namespace:                sv.Namespace,
		Annotations:              sv.Annotations,
		Labels:                   sv.Labels,
		Created:                  created,
		CreatedBy:                actorUID,
		Updated:                  time.Now().UTC().Unix(),
		UpdatedBy:                actorUID,
		OwnerReferenceAPIGroup:   ownerReferenceAPIGroup,
		OwnerReferenceAPIVersion: ownerReferenceAPIVersion,
		OwnerReferenceKind:       ownerReferenceKind,
		OwnerReferenceName:       ownerReferenceName,
		Active:                   false,
		Version:                  version,
		Keeper:                   &keeperName,
		Ref:                      sv.Spec.Ref,
		Decrypters:               sv.Spec.Decrypters,
		Description:              sv.Spec.Description,
		ExternalID:               "",
	}

	createdVersion, err := s.secureValueMetadataStorage.CreateV2(ctx, sv2)
	if err != nil {
		return fmt.Errorf("creating secure value metadata: %w", err)
	}

	keeper, err := s.keeperService.KeeperForConfig(keeperCfg)
	if err != nil {
		return fmt.Errorf("getting keeper for config: namespace=%+v keeperName=%+v %w", sv.Namespace, keeperName, err)
	}
	logging.FromContext(ctx).Debug("retrieved keeper", "namespace", sv.Namespace, "type", keeperCfg.Type())

	externalID, err := keeper.Store(ctx, keeperCfg, xkube.Namespace(sv.Namespace), sv.Name, createdVersion, sv.Spec.Value.DangerouslyExposeAndConsumeValue())
	if err != nil {
		return fmt.Errorf("storing secure value in keeper: %w", err)
	}

	sv2.ExternalID = string(externalID)
	sv2.Active = true

	if err := s.secureValueMetadataStorage.UpdateV2(ctx, sv2); err != nil {
		return fmt.Errorf("updating v2 secure value metadata: %w", err)
	}

	return nil
}

func (s *SecureValueService) DeleteV2(ctx context.Context, namespace xkube.Namespace, name string) error {
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	sv, err := s.secureValueMetadataStorage.Read(ctx, namespace, name, contracts.ReadOpts{ForUpdate: true})
	if err != nil {
		return fmt.Errorf("fetching secure value: %+w", err)
	}

	sv2 := &contracts.SecureValueMetadataModel{
		GUID:        string(sv.UID),
		Name:        sv.Name,
		Namespace:   sv.Namespace,
		Annotations: sv.Annotations,
		Labels:      sv.Labels,
		Created:     sv.GetCreationTimestamp().Unix(),
		CreatedBy:   sv.GetCreatedBy(),
		Updated:     time.Now().UTC().Unix(),
		UpdatedBy:   sv.GetUpdatedBy(),
		Active:      false, // set to active=false
		Version:     sv.Status.Version,
		Keeper:      &sv.Status.Keeper,
		Ref:         sv.Spec.Ref,
		Decrypters:  sv.Spec.Decrypters,
		Description: sv.Spec.Description,
		ExternalID:  sv.Status.ExternalID,
	}

	if err := s.secureValueMetadataStorage.UpdateV2(ctx, sv2); err != nil {
		return fmt.Errorf("setting secure value version to inactive: %+w", err)
	}

	return nil
}
