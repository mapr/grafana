import { useEffect } from 'react';

import { Trans, t } from '@grafana/i18n';
import { config } from '@grafana/runtime';
import { Alert, Button, EmptyState, LinkButton, LoadingPlaceholder, Stack, Text, TextLink } from '@grafana/ui';

import { logError } from '../../Analytics';
import { AlertingPageWrapper } from '../../components/AlertingPageWrapper';
import { WithReturnButton } from '../../components/WithReturnButton';
import { AutoSyncConfiguration } from '../../components/settings/AutoSyncConfiguration';
import { useAutoSyncConfiguration } from '../../components/settings/useAutoSyncConfiguration';
import { AlertmanagerProvider } from '../../state/AlertmanagerContext';
import { hasAutoSyncSource } from '../../utils/autoSync';
import { GRAFANA_RULES_SOURCE_NAME } from '../../utils/datasource';
import { DOCS_URL_ALERTING_MIGRATION } from '../../utils/docs';
import { stringifyErrorLike } from '../../utils/misc';
import { withPageErrorBoundary } from '../../withPageErrorBoundary';
import { useSettingsPageNav } from '../navigation';

import { AutoSyncStagedCard } from './AutoSyncStagedCard';
import { StagedConfiguration } from './StagedConfiguration';
import { type StagedExtraConfig } from './stagedConfig';
import { getStagedConfigPermissions } from './stagedConfigPermissions';
import { useRefetchStagedConfigOnSyncChange, useStagedConfig } from './useStagedConfig';

const IMPORT_WIZARD_URL = '/alerting/import-to-gma';

function ImportSettingsPage() {
  const { navId, pageNav } = useSettingsPageNav();

  return (
    <AlertingPageWrapper
      navId={navId}
      pageNav={pageNav}
      actions={[
        <WithReturnButton
          key="add-alertmanager"
          title={t('alerting.settings-content.title-alerting-settings', 'Alerting settings')}
          component={
            <LinkButton href="/connections/datasources/alertmanager" icon="plus" variant="primary">
              <Trans i18nKey="alerting.settings-content.add-new-alertmanager">Add new Alertmanager</Trans>
            </LinkButton>
          }
        />,
      ]}
    >
      <AlertmanagerProvider accessType="notification" alertmanagerSourceName={GRAFANA_RULES_SOURCE_NAME}>
        <ImportSettingsContent />
      </AlertmanagerProvider>
    </AlertingPageWrapper>
  );
}

function ImportSettingsContent() {
  const isAutoSyncEnabled = config.featureToggles['alerting.syncExternalAlertmanager'];
  return isAutoSyncEnabled ? <ImportSettingsWithAutoSync /> : <StagedConfigurationSection />;
}

function ImportSettingsWithAutoSync() {
  const { stagedConfig, refetch: refetchStagedConfig } = useStagedConfig();
  useRefetchStagedConfigOnSyncChange(refetchStagedConfig);
  const autoSync = useAutoSyncConfiguration();

  // Deciding between the two layouts needs the sync setting; guessing would flash the wrong one.
  if (autoSync.isLoading) {
    return <LoadingPlaceholder text={t('alerting.settings.import.loading', 'Loading imported configurations…')} />;
  }

  // Once sync is set up, its setting and the staged copy it keeps are one thing, shown in one card. That
  // stays so after sync is turned off while its last copy is still staged, so it can be turned back on
  // from the same place. Otherwise the setting has its own card to pick a datasource and turn sync on.
  const isSyncSetUp = autoSync.state.kind === 'configured' || autoSync.state.kind === 'operator-managed';
  const hasStoppedSyncCopy = Boolean(stagedConfig && isStoppedSyncCopy(autoSync, stagedConfig));
  const showSyncCard = isSyncSetUp || hasStoppedSyncCopy;

  return (
    <Stack direction="column" gap={2}>
      {!showSyncCard && <AutoSyncConfiguration stagedConfigIdentifier={stagedConfig?.identifier} />}
      <StagedConfigurationSection autoSync={showSyncCard ? autoSync : undefined} />
    </Stack>
  );
}

/**
 * Whether the staged config is the copy a sync left behind when it was turned off. Once sync is off the
 * backend no longer marks the copy as auto-sync, but the status still remembers the datasource the sync
 * last ran from and the name it imported under.
 */
function isStoppedSyncCopy(autoSync: ReturnType<typeof useAutoSyncConfiguration>, stagedConfig: StagedExtraConfig) {
  return (
    autoSync.state.kind === 'unconfigured' &&
    Boolean(autoSync.lastSyncedUid) &&
    stagedConfig.identifier === autoSync.getSyncIdentifier(autoSync.lastSyncedUid)
  );
}

interface StagedConfigurationSectionProps {
  /** Set while auto-sync is set up, so the section shows the sync and its copy together. */
  autoSync?: ReturnType<typeof useAutoSyncConfiguration>;
}

function StagedConfigurationSection({ autoSync }: StagedConfigurationSectionProps) {
  const { canPromote, canRevert } = getStagedConfigPermissions();
  const { stagedConfig, liveConfig, isLoading, isError, error, refetch } = useStagedConfig();

  const isSyncManaged = stagedConfig?.managed_by === 'auto-sync';
  const syncUid = autoSync && hasAutoSyncSource(autoSync.state) ? autoSync.state.uid : '';
  // The sync's own copy, or another import that keeps the sync from writing one.
  const isSyncedCopy = Boolean(
    autoSync &&
      stagedConfig &&
      (autoSync.state.kind === 'unconfigured'
        ? isStoppedSyncCopy(autoSync, stagedConfig)
        : stagedConfig.identifier === autoSync.getSyncIdentifier(syncUid))
  );
  const otherStagedConfig = stagedConfig && !isSyncedCopy ? stagedConfig : undefined;

  useEffect(() => {
    if (isError) {
      logError(new Error(stringifyErrorLike(error)));
    }
  }, [isError, error]);

  return (
    <Stack direction="column" gap={1}>
      <Text variant="h4">
        <Trans i18nKey="alerting.settings.import.staged-title">Staged configuration</Trans>
      </Text>
      <Text variant="bodySmall" color="secondary">
        <Trans i18nKey="alerting.settings.import.staged-description">
          A read-only, reversible copy of an imported Alertmanager config. Promote it to merge into your live config, or
          revert to remove it.
        </Trans>
      </Text>

      {isLoading && (
        <LoadingPlaceholder text={t('alerting.settings.import.loading', 'Loading imported configurations…')} />
      )}

      {isError && (
        <Alert
          severity="error"
          title={t('alerting.settings.import.error-title', "Couldn't load imported configurations")}
        >
          <Stack direction="column" gap={2} alignItems="flex-start">
            <Trans i18nKey="alerting.settings.import.error-body">
              The request to the Alerting API failed. Check your connection and try again.
            </Trans>
            <Button variant="secondary" fill="outline" icon="sync" onClick={() => refetch()}>
              <Trans i18nKey="alerting.settings.import.error-retry">Retry</Trans>
            </Button>
          </Stack>
        </Alert>
      )}

      {!isLoading && !isError && autoSync && (
        <AutoSyncStagedCard
          autoSync={autoSync}
          syncedConfig={isSyncedCopy ? stagedConfig : undefined}
          conflictIdentifier={otherStagedConfig?.identifier}
          liveConfig={liveConfig}
          canPromote={canPromote}
          canRevert={canRevert}
        />
      )}

      {!isLoading && !isError && autoSync && otherStagedConfig && (
        <StagedConfiguration
          stagedConfig={otherStagedConfig}
          canPromote={canPromote}
          canRevert={canRevert}
          liveConfig={liveConfig}
        />
      )}

      {!isLoading && !isError && !autoSync && !stagedConfig && (
        <EmptyState
          variant="call-to-action"
          message={t('alerting.settings.import.empty-title', 'No configuration imported yet')}
          button={
            <LinkButton icon="cloud-upload" size="lg" href={IMPORT_WIZARD_URL}>
              <Trans i18nKey="alerting.settings.import.empty-cta">Import Alertmanager configuration</Trans>
            </LinkButton>
          }
        >
          <Trans i18nKey="alerting.settings.import.empty-body">
            Import an Alertmanager configuration to stage it here as a safe, reversible copy. Review what it contains,
            then promote it into your live Grafana Alertmanager when you&apos;re ready.
          </Trans>{' '}
          <TextLink href={DOCS_URL_ALERTING_MIGRATION} external>
            <Trans i18nKey="alerting.settings.import.empty-learn-more">Learn more about importing configurations</Trans>
          </TextLink>
        </EmptyState>
      )}

      {!isLoading && !isError && !autoSync && stagedConfig && (
        <StagedConfiguration
          stagedConfig={stagedConfig}
          canPromote={canPromote}
          canRevert={canRevert}
          isSyncManaged={isSyncManaged}
          liveConfig={liveConfig}
        />
      )}
    </Stack>
  );
}

export default withPageErrorBoundary(ImportSettingsPage);
