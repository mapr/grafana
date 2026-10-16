import { useLocation } from 'react-router-dom-v5-compat';

import { type NavModelItem } from '@grafana/data';
import { t } from '@grafana/i18n';
import { useSelector } from 'app/types/store';

import { useIsAlertmanagerImportEnabled } from '../featureToggles';

import { useSettingsExtensionsNav } from './extensions';

/** Deep-link target for the Alerting settings Alertmanager tab. */
const ALERTING_SETTINGS_URL = '/alerting/admin/alertmanager';

/** Deep-link target for the Alerting settings Import tab (auto-sync + staged config live here). */
export const ALERTING_IMPORT_SETTINGS_URL = '/alerting/admin/import';

export function useSettingsPageNav() {
  const location = useLocation();

  const navIndex = useSelector((state) => state.navIndex);
  const settingsNav = navIndex['alerting-admin'];

  const extensionTabs = useSettingsExtensionsNav();
  const alertmanagerImportEnabled = useIsAlertmanagerImportEnabled();

  const importTab: NavModelItem = {
    id: 'import',
    text: t('alerting.settings.tabs.import', 'Import'),
    url: ALERTING_IMPORT_SETTINGS_URL,
    active: location.pathname === ALERTING_IMPORT_SETTINGS_URL,
    icon: 'cloud-upload',
    parentItem: settingsNav,
  };

  // All available tabs including the main alertmanager tab
  const allTabs: NavModelItem[] = [
    {
      id: 'alertmanager',
      text: t('alerting.settings.tabs.alert-managers', 'Alertmanagers'),
      url: ALERTING_SETTINGS_URL,
      active: location.pathname === ALERTING_SETTINGS_URL,
      icon: 'cloud',
      parentItem: settingsNav,
    },
    // The wizard flags and alerting.syncExternalAlertmanager always ship together, and auto-sync needs the
    // Alertmanager import API anyway, so gating the Import tab (which now hosts the auto-sync card) on the
    // wizard never hides a working auto-sync from an instance that has sync enabled.
    ...(alertmanagerImportEnabled ? [importTab] : []),
    ...extensionTabs,
  ];

  // Create pageNav that represents the Settings page with tabs as children
  const pageNav: NavModelItem = {
    ...settingsNav,
    children: allTabs,
  };

  return {
    navId: 'alerting-admin',
    pageNav,
  };
}
