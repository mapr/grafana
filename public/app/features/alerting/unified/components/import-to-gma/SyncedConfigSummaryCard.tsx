import { css } from '@emotion/css';

import { type GrafanaTheme2 } from '@grafana/data';
import { t } from '@grafana/i18n';
import { type IconName, Stack, Text, TextLink, useStyles2 } from '@grafana/ui';
import { DataSourceLogo } from 'app/features/datasources/components/picker/DataSourceLogo';

import {
  type StagedExtraConfig,
  parseStagedAlertmanagerConfig,
  summarizeStagedConfig,
} from '../../settings/import/stagedConfig';
import { ALERTING_IMPORT_SETTINGS_URL } from '../../settings/navigation';
import { type AutoSyncHealth } from '../../utils/autoSync';
import { getDataSourceByUid } from '../../utils/datasource';
import { MetaText } from '../MetaText';
import { WithReturnButton } from '../WithReturnButton';
import { SyncedCopyStatusBadge } from '../settings/AutoSyncStatusBadge';

interface Props {
  /** The staged copy auto-sync keeps up to date. Missing until the first sync lands, or once it is promoted. */
  syncedConfig?: StagedExtraConfig;
  /** The datasource auto-sync mirrors from. Shown instead of the config's identifier, which may be its UID. */
  datasourceUid: string;
  /** What the sync worker last reported, shown while there is no synced config to summarise. */
  syncHealth: AutoSyncHealth;
  /** The Alerting settings page is admin-only, so only admins get the link to it. */
  showSettingsLink: boolean;
}

interface SummaryItem {
  key: string;
  icon: IconName;
  count: number;
  label: string;
}

/**
 * A short summary of what auto-sync mirrors from the external Alertmanager: one count per kind of
 * resource, laid out like the metadata row of a notification policy card. Until there is a synced
 * config, it shows the sync status instead. The full breakdown lives on the Alerting settings page.
 */
export function SyncedConfigSummaryCard({ syncedConfig, datasourceUid, syncHealth, showSettingsLink }: Props) {
  const styles = useStyles2(getStyles);
  const items = syncedConfig ? getSummaryItems(syncedConfig) : [];
  // Falls back to the identifier when the datasource has been deleted or isn't visible to this user.
  const datasource = datasourceUid ? getDataSourceByUid(datasourceUid) : undefined;
  const statusDescription = getStatusDescription(syncHealth, Boolean(syncedConfig));

  return (
    <div className={styles.card}>
      <Stack direction="column" gap={1}>
        <Stack direction="row" alignItems="center" justifyContent="space-between" gap={1} wrap="wrap">
          <Stack direction="row" alignItems="center" gap={1}>
            {datasource && <DataSourceLogo dataSource={datasource} />}
            <Text element="h4" variant="body" weight="medium">
              {datasource?.name ?? syncedConfig?.identifier ?? datasourceUid}
            </Text>
            <SyncedCopyStatusBadge uid={datasourceUid} syncHealth={syncHealth} hasSyncedCopy={Boolean(syncedConfig)} />
          </Stack>
          {showSettingsLink && (
            <WithReturnButton
              title={t('alerting.import-to-gma-tool.pageTitle', 'Import to Grafana Alerting')}
              component={
                <TextLink href={ALERTING_IMPORT_SETTINGS_URL} inline={false}>
                  {t('alerting.import-to-gma.autosync-active-block.view-details', 'View details')}
                </TextLink>
              }
            />
          )}
        </Stack>
        {statusDescription && (
          <Text variant="bodySmall" color="secondary">
            {statusDescription}
          </Text>
        )}
        {items.length > 0 && (
          <Stack direction="row" alignItems="center" gap={1} wrap="wrap">
            {items.map((item) => (
              <MetaText key={item.key} icon={item.icon} data-testid={`synced-config-${item.key}`}>
                <Text color="primary">{item.count}</Text>
                <span>{item.label}</span>
              </MetaText>
            ))}
          </Stack>
        )}
      </Stack>
    </div>
  );
}

function getStatusDescription(syncHealth: AutoSyncHealth, hasSyncedCopy: boolean): string | undefined {
  if (hasSyncedCopy) {
    // The counts below come from the last sync that went through, which may no longer match the source.
    switch (syncHealth.kind) {
      case 'failing':
        return t(
          'alerting.import-to-gma.synced-summary.status-failing-with-copy',
          'The last sync attempt failed. This is the configuration from the last successful sync.'
        );
      case 'pending':
        return t(
          'alerting.import-to-gma.synced-summary.status-pending-with-copy',
          'Waiting for the next sync. This is the configuration from the last sync.'
        );
      default:
        return undefined;
    }
  }
  switch (syncHealth.kind) {
    case 'failing':
      return t(
        'alerting.import-to-gma.synced-summary.status-failing',
        'The last sync attempt failed, so there is nothing to show yet.'
      );
    case 'merge-committed':
      return t(
        'alerting.import-to-gma.synced-summary.status-merged',
        'The synced configuration was merged into Grafana, and sync has stopped.'
      );
    default:
      return t(
        'alerting.import-to-gma.synced-summary.status-pending',
        'Waiting for the first sync. This usually takes a minute or two.'
      );
  }
}

function getSummaryItems(syncedConfig: StagedExtraConfig): SummaryItem[] {
  const config = parseStagedAlertmanagerConfig(syncedConfig.alertmanager_config);
  if (!config) {
    return [];
  }
  const summary = summarizeStagedConfig(config, syncedConfig.template_files);

  const items: SummaryItem[] = [
    {
      key: 'contact-points',
      icon: 'at',
      count: summary.receivers.length,
      label: t('alerting.import-to-gma.synced-summary.contact-points', '', {
        count: summary.receivers.length,
        defaultValue_one: 'contact point',
        defaultValue_other: 'contact points',
      }),
    },
    {
      key: 'notification-policies',
      icon: 'sitemap',
      count: summary.hasRoutingTree ? 1 : 0,
      label: t('alerting.import-to-gma.synced-summary.notification-policies', '', {
        count: summary.hasRoutingTree ? 1 : 0,
        defaultValue_one: 'notification policy tree',
        defaultValue_other: 'notification policy trees',
      }),
    },
    {
      key: 'templates',
      icon: 'file-alt',
      count: summary.templates.length,
      label: t('alerting.import-to-gma.synced-summary.templates', '', {
        count: summary.templates.length,
        defaultValue_one: 'template',
        defaultValue_other: 'templates',
      }),
    },
    {
      key: 'time-intervals',
      icon: 'history',
      count: summary.timeIntervals.length,
      label: t('alerting.import-to-gma.synced-summary.time-intervals', '', {
        count: summary.timeIntervals.length,
        defaultValue_one: 'time interval',
        defaultValue_other: 'time intervals',
      }),
    },
    {
      key: 'inhibition-rules',
      icon: 'shield',
      count: summary.inhibitionRuleCount,
      label: t('alerting.import-to-gma.synced-summary.inhibition-rules', '', {
        count: summary.inhibitionRuleCount,
        defaultValue_one: 'inhibition rule',
        defaultValue_other: 'inhibition rules',
      }),
    },
  ];

  // Kinds the external Alertmanager doesn't use would only add noise.
  return items.filter((item) => item.count > 0);
}

// Matches the notification policy card, so the summary reads as the same kind of object.
const getStyles = (theme: GrafanaTheme2) => ({
  card: css({
    padding: theme.spacing(1.5),
    background: theme.flags.visualDesignRefresh ? theme.colors.background.primary : theme.colors.background.secondary,
    borderRadius: theme.shape.radius.lg,
    border: `solid 1px ${theme.colors.border.weak}`,
  }),
});
