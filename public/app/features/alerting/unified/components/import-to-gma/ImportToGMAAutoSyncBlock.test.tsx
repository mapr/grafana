import { HttpResponse, http } from 'msw';
import { act, render, screen, testWithFeatureToggles, waitFor, within } from 'test/test-utils';
import { byPlaceholderText, byRole, byTestId, byText } from 'testing-library-selector';

import { OrgRole } from '@grafana/data';
import { selectors } from '@grafana/e2e-selectors';
import { FlagKeys } from '@grafana/runtime/internal';
import { setTestFlags } from '@grafana/test-utils/unstable';
import { type AlertManagerCortexConfig } from 'app/plugins/datasource/alertmanager/types';
import { AccessControlAction } from 'app/types/accessControl';

import { setupMswServer } from '../../mockApi';
import { grantUserPermissions, grantUserRole, mockDataSource } from '../../mocks';
import { setupDatasourcesEndpoint } from '../../mocks/server/configure/datasources';
import {
  MIMIR_FETCH_FAILED_CONDITION,
  SYNC_SUCCEEDED_CONDITION,
  setupAutoSyncConfig,
} from '../../mocks/server/handlers/k8s/config.k8s';
import { type StagedExtraConfig } from '../../settings/import/stagedConfig';
import { setupDataSources } from '../../testSetup/datasources';
import { DataSourceType } from '../../utils/datasource';

import { ImportWizardGate } from './ImportToGMA';

const mockReturnToPrevious = jest.fn();
// The Settings links set up a way back to the wizard, which needs the app chrome these tests don't render.
jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  useReturnToPrevious: () => mockReturnToPrevious,
}));

const server = setupMswServer();

const MIMIR_DS_UID = 'mimir-uid';

const AM_CONFIG_URL = '/api/alertmanager/:name/config/api/v1/alerts';

const syncedConfig: StagedExtraConfig = {
  identifier: 'mimir-prod',
  alertmanager_config: 'route:\n  receiver: default\nreceivers:\n  - name: default\n',
  template_files: { 'slack.tmpl': '', 'email.tmpl': '' },
  managed_by: 'auto-sync',
};

/** Serves the Grafana Alertmanager config carrying the given staged config, and counts the requests. */
function serveStagedConfig(staged?: StagedExtraConfig) {
  const requests = { count: 0 };
  server.use(
    http.get(AM_CONFIG_URL, () => {
      requests.count++;
      return HttpResponse.json<AlertManagerCortexConfig>({
        alertmanager_config: {},
        template_files: {},
        extra_config: staged ? [staged] : undefined,
      });
    })
  );
  return requests;
}

// useAutoSyncConfiguration (mounted by the wizard once the gate lets it through) fires this
// unconditionally.
beforeEach(() => {
  setupDatasourcesEndpoint(server, []);
});

// The wizard locks its notifications step unless the Alertmanager import API is on.
beforeEach(() => setTestFlags({ [FlagKeys.AlertingImportAlertmanagerAPI]: true }));
// The reset fires OpenFeature events while components are still mounted, so it needs act.
afterEach(async () => {
  await act(async () => {
    setTestFlags();
  });
});

const ui = {
  blockTitle: byText(/auto-sync is enabled/i),
  importAPIDisabledTitle: byText(/notification configuration import is unavailable/i),
  goToSettings: byRole('link', { name: /go to alerting settings/i }),
  viewDetails: byRole('link', { name: /^view details$/i }),
  syncedConfigHeading: byRole('heading', { name: 'mimir-prod' }),
  syncedDatasourceHeading: byRole('heading', { name: 'Mimir Prod' }),
  syncedDatasourceLogo: byRole('img', { name: /logo$/i }),
  contactPointCount: byTestId('synced-config-contact-points'),
  policyTreeCount: byTestId('synced-config-notification-policies'),
  templateCount: byTestId('synced-config-templates'),
  inhibitionRuleCount: byTestId('synced-config-inhibition-rules'),
  syncedBadge: byText(/synced · read-only/i),
  pendingCardHeading: byRole('heading', { name: MIMIR_DS_UID }),
  pendingBadge: byText(/pending first sync/i),
  pendingDescription: byText(/waiting for the first sync/i),
  failingBadge: byText(/sync failing/i),
  failingWithCopyDescription: byText(/configuration from the last successful sync/i),
  pendingWithCopyDescription: byText(/waiting for the next sync/i),
  importRules: byRole('link', { name: /import alert rules/i }),
  disableSyncHint: byText(/disable auto-sync in Alerting settings/i),
  notificationsStep: byRole('group', { name: /import notification configuration/i }),
  rulesStep: byRole('group', { name: /import rules/i }),
  nextButton: byTestId(selectors.pages.Alerting.ImportToGMA.nextButton),
  // The wizard sidebar is the only list on the page.
  stepper: byRole('list'),
  policyTreeNameInput: byPlaceholderText('prometheus-prod'),
  dataSourceSourceOption: byText(/^data source$/i),
  notificationsDataSourcePlaceholder: byText('Select data source'),
  clearValueButton: byRole('button', { name: /clear value/i }),
  skipButton: byTestId(selectors.pages.Alerting.ImportToGMA.skipButton),
  step1SkippedNotice: byText(/step 1 was skipped/i),
  nothingToImport: byRole('status', { name: /nothing to import/i }),
};

// Read access to the sync Config plus the permissions the wizard itself needs. Note: no admin role —
// the lock must apply to any user allowed to reach the wizard.
function grantWizardPermissions() {
  grantUserPermissions([
    AccessControlAction.ActionAlertingNotificationsConfigRead,
    AccessControlAction.AlertingRuleCreate,
    AccessControlAction.AlertingProvisioningSetStatus,
  ]);
}

describe('Import wizard auto-sync gate', () => {
  it('renders the wizard normally when the sync feature is off, regardless of config', async () => {
    setupAutoSyncConfig(server, { specUid: MIMIR_DS_UID });

    render(<ImportWizardGate />);

    // Flag off -> the sync query is skipped and the wizard renders.
    expect(await ui.notificationsStep.find()).toBeInTheDocument();
    expect(ui.blockTitle.query()).not.toBeInTheDocument();
  });

  describe('when the sync feature is enabled', () => {
    testWithFeatureToggles({ enable: ['alerting.syncExternalAlertmanager'] });

    beforeEach(grantWizardPermissions);

    it('locks only the notifications step when auto-sync is active', async () => {
      grantUserRole(OrgRole.Admin);
      setupAutoSyncConfig(server, { specUid: MIMIR_DS_UID });

      render(<ImportWizardGate />);

      expect(await ui.notificationsStep.find()).toBeInTheDocument();
      expect(ui.blockTitle.get()).toBeInTheDocument();
      expect(ui.disableSyncHint.get()).toBeInTheDocument();
      // Settings is reached from the summary card only, never from the notice.
      expect(ui.goToSettings.query()).not.toBeInTheDocument();
      // The locked step has nothing to skip, it only offers to continue.
      expect(ui.nextButton.get()).toHaveAttribute('aria-disabled', 'false');
      expect(ui.skipButton.query()).not.toBeInTheDocument();
      // Rule import happens in the wizard itself, so there is no link out to the old import page.
      expect(ui.importRules.query()).not.toBeInTheDocument();
    });

    describe('previewing the synced configuration', () => {
      beforeEach(() => {
        grantUserPermissions([
          AccessControlAction.ActionAlertingNotificationsConfigRead,
          AccessControlAction.AlertingRuleCreate,
          AccessControlAction.AlertingProvisioningSetStatus,
          AccessControlAction.AlertingNotificationsRead,
        ]);
        // A healthy sync by default; tests about other states set their own.
        setupAutoSyncConfig(server, {
          specUid: MIMIR_DS_UID,
          statusUid: MIMIR_DS_UID,
          condition: SYNC_SUCCEEDED_CONDITION,
        });
        // Start without registered datasources; a test that needs one registers it.
        setupDataSources();
      });

      it('names the card after the synced datasource, with its logo', async () => {
        grantUserRole(OrgRole.Admin);
        setupDataSources(mockDataSource({ uid: MIMIR_DS_UID, name: 'Mimir Prod', type: DataSourceType.Alertmanager }));
        serveStagedConfig(syncedConfig);

        render(<ImportWizardGate />);

        expect(await ui.syncedDatasourceHeading.find()).toBeInTheDocument();
        expect(ui.syncedDatasourceLogo.get()).toBeInTheDocument();
        // The datasource name replaces the config's identifier.
        expect(ui.syncedConfigHeading.query()).not.toBeInTheDocument();
      });

      // No datasource is registered in the other tests, so they also cover falling back to the identifier.
      it('summarises what the synced config contains, with a details link for admins', async () => {
        grantUserRole(OrgRole.Admin);
        serveStagedConfig(syncedConfig);

        const { user } = render(<ImportWizardGate />);

        expect(await ui.syncedConfigHeading.find()).toBeInTheDocument();
        expect(ui.syncedBadge.get()).toBeInTheDocument();
        expect(ui.contactPointCount.get()).toHaveTextContent('1contact point');
        expect(ui.policyTreeCount.get()).toHaveTextContent('1notification policy tree');
        expect(ui.templateCount.get()).toHaveTextContent('2templates');
        // Kinds the synced config doesn't have are left out.
        expect(ui.inhibitionRuleCount.query()).not.toBeInTheDocument();
        expect(ui.viewDetails.get()).toHaveAttribute('href', '/alerting/admin/import');
        // Following the link offers a way back to the wizard.
        await user.click(ui.viewDetails.get());
        expect(mockReturnToPrevious).toHaveBeenCalledWith('Import to Grafana Alerting');
        // The card carries the Settings link, so the notice doesn't repeat it.
        expect(ui.goToSettings.query()).not.toBeInTheDocument();
      });

      it('shows the preview without the admin-only Settings link to non-admins', async () => {
        grantUserRole(OrgRole.Editor);
        serveStagedConfig(syncedConfig);

        render(<ImportWizardGate />);

        expect(await ui.syncedConfigHeading.find()).toBeInTheDocument();
        expect(ui.viewDetails.query()).not.toBeInTheDocument();
      });

      it('skips the preview, and its request, for users who cannot read notifications', async () => {
        grantUserRole(OrgRole.Editor);
        grantUserPermissions([
          AccessControlAction.ActionAlertingNotificationsConfigRead,
          AccessControlAction.AlertingRuleCreate,
          AccessControlAction.AlertingProvisioningSetStatus,
        ]);
        const requests = serveStagedConfig(syncedConfig);

        render(<ImportWizardGate />);

        expect(await ui.blockTitle.find()).toBeInTheDocument();
        expect(ui.syncedConfigHeading.query()).not.toBeInTheDocument();
        expect(requests.count).toBe(0);
      });

      // An older copy must not read as up to date while the sync is failing or hasn't run since.
      it('shows a failing sync as failing even with a copy from an earlier sync', async () => {
        grantUserRole(OrgRole.Admin);
        setupAutoSyncConfig(server, {
          specUid: MIMIR_DS_UID,
          statusUid: MIMIR_DS_UID,
          condition: MIMIR_FETCH_FAILED_CONDITION,
        });
        serveStagedConfig(syncedConfig);

        render(<ImportWizardGate />);

        expect(await ui.failingBadge.find()).toBeInTheDocument();
        expect(ui.syncedBadge.query()).not.toBeInTheDocument();
        expect(ui.failingWithCopyDescription.get()).toBeInTheDocument();
        // The copy it still has is summarised.
        expect(ui.contactPointCount.get()).toBeInTheDocument();
      });

      it('shows a sync that has not run since being turned back on as pending', async () => {
        grantUserRole(OrgRole.Admin);
        setupAutoSyncConfig(server, { specUid: MIMIR_DS_UID });
        serveStagedConfig(syncedConfig);

        render(<ImportWizardGate />);

        expect(await ui.pendingBadge.find()).toBeInTheDocument();
        expect(ui.syncedBadge.query()).not.toBeInTheDocument();
        expect(ui.pendingWithCopyDescription.get()).toBeInTheDocument();
      });

      it('shows the card as pending while nothing has synced yet', async () => {
        grantUserRole(OrgRole.Admin);
        setupAutoSyncConfig(server, { specUid: MIMIR_DS_UID });
        serveStagedConfig(undefined);

        render(<ImportWizardGate />);

        // No datasource is registered here, so the card falls back to the datasource UID.
        expect(await ui.pendingCardHeading.find()).toBeInTheDocument();
        expect(ui.pendingBadge.get()).toBeInTheDocument();
        expect(ui.pendingDescription.get()).toBeInTheDocument();
        expect(ui.syncedBadge.query()).not.toBeInTheDocument();
        expect(ui.viewDetails.get()).toBeInTheDocument();
      });
    });

    it('lets the user continue to the rules step when auto-sync is active', async () => {
      grantUserRole(OrgRole.Viewer);
      setupAutoSyncConfig(server, { specUid: MIMIR_DS_UID });
      // The rules step renders a data source picker.
      setupDataSources();

      const { user } = render(<ImportWizardGate />);

      expect(await ui.blockTitle.find()).toBeInTheDocument();
      await user.click(ui.nextButton.get());

      expect(await ui.rulesStep.find()).toBeInTheDocument();
      // Moving on from a locked step isn't a skip.
      expect(ui.step1SkippedNotice.query()).not.toBeInTheDocument();

      // Nothing from the locked step reaches the import: with the rules skipped too, there is nothing left.
      // Without a rules source, the step's single button skips.
      expect(ui.nextButton.get()).toHaveTextContent(/^skip$/i);
      await user.click(ui.nextButton.get());
      expect(await ui.nothingToImport.find()).toBeInTheDocument();
    });

    it('lets the user jump to the rules step from the sidebar when auto-sync is active', async () => {
      grantUserRole(OrgRole.Viewer);
      setupAutoSyncConfig(server, { specUid: MIMIR_DS_UID });
      // The rules step renders a data source picker.
      setupDataSources();

      const { user } = render(<ImportWizardGate />);

      expect(await ui.blockTitle.find()).toBeInTheDocument();
      const sidebarRulesStep = within(ui.stepper.get()).getByRole('button', { name: /rules/i });
      expect(sidebarRulesStep).toBeEnabled();
      await user.click(sidebarRulesStep);

      expect(await ui.rulesStep.find()).toBeInTheDocument();
      expect(ui.step1SkippedNotice.query()).not.toBeInTheDocument();
    });

    it('hides the admin-only Settings link from non-admins', async () => {
      // The Settings route is gated on the Org Admin role, so linking non-admins there would bounce
      // them to the home page.
      grantUserRole(OrgRole.Editor);
      setupAutoSyncConfig(server, { specUid: MIMIR_DS_UID });

      render(<ImportWizardGate />);

      expect(await ui.blockTitle.find()).toBeInTheDocument();
      expect(ui.goToSettings.query()).not.toBeInTheDocument();
      expect(ui.disableSyncHint.query()).not.toBeInTheDocument();
    });

    it('renders the wizard when auto-sync is not active', async () => {
      setupAutoSyncConfig(server, {});

      render(<ImportWizardGate />);

      expect(await ui.notificationsStep.find()).toBeInTheDocument();
      await waitFor(() => expect(ui.blockTitle.query()).not.toBeInTheDocument());
    });
  });
});

describe('Import wizard with an empty notifications step', () => {
  beforeEach(() => {
    grantWizardPermissions();
    grantUserPermissions([
      AccessControlAction.ActionAlertingNotificationsConfigRead,
      AccessControlAction.AlertingRuleCreate,
      AccessControlAction.AlertingProvisioningSetStatus,
      AccessControlAction.AlertingNotificationsWrite,
    ]);
    // The rules step renders a data source picker.
    setupDataSources();
  });

  it('lets Next move on as a skip while nothing is entered', async () => {
    const { user } = render(<ImportWizardGate />);

    expect(await ui.notificationsStep.find()).toBeInTheDocument();
    // One button, which says what it does: there is nothing to continue with yet.
    expect(ui.nextButton.get()).toHaveTextContent(/^skip$/i);
    expect(ui.skipButton.query()).not.toBeInTheDocument();
    expect(ui.nextButton.get()).toHaveAttribute('aria-disabled', 'false');
    await user.click(ui.nextButton.get());

    expect(await ui.rulesStep.find()).toBeInTheDocument();
    expect(ui.step1SkippedNotice.get()).toBeInTheDocument();
  });

  it('lets the user jump to the rules step from the sidebar while nothing is entered', async () => {
    const { user } = render(<ImportWizardGate />);

    expect(await ui.notificationsStep.find()).toBeInTheDocument();
    await user.click(within(ui.stepper.get()).getByRole('button', { name: /rules/i }));

    expect(await ui.rulesStep.find()).toBeInTheDocument();
  });

  it('turns back into Skip when a picked data source is cleared again', async () => {
    setupDataSources(
      mockDataSource(
        { uid: 'mimir-prod-uid', name: 'Mimir Prod', type: DataSourceType.Alertmanager },
        { alerting: true }
      )
    );
    const { user } = render(<ImportWizardGate />);

    expect(await ui.notificationsStep.find()).toBeInTheDocument();
    await user.click(ui.dataSourceSourceOption.get());
    await user.click(await ui.notificationsDataSourcePlaceholder.find());
    await user.click(await screen.findByText('Mimir Prod'));

    // Picking a data source fills in the policy tree name and turns the button into Continue.
    expect(ui.policyTreeNameInput.get()).toHaveValue('mimir-prod');
    expect(ui.nextButton.get()).toHaveTextContent(/^continue$/i);

    await user.click(ui.clearValueButton.get());

    // Clearing takes the name it filled in along with it, so the step is empty again.
    expect(ui.policyTreeNameInput.get()).toHaveValue('');
    await waitFor(() => expect(ui.nextButton.get()).toHaveTextContent(/^skip$/i));
  });

  it('offers to skip an empty rules step too, and lets the sidebar reach the review', async () => {
    const { user } = render(<ImportWizardGate />);

    expect(await ui.notificationsStep.find()).toBeInTheDocument();
    await user.click(ui.nextButton.get());
    expect(await ui.rulesStep.find()).toBeInTheDocument();

    // No rules source chosen yet, so the single button skips.
    expect(ui.nextButton.get()).toHaveTextContent(/^skip$/i);
    expect(ui.skipButton.query()).not.toBeInTheDocument();
    expect(within(ui.stepper.get()).getByRole('button', { name: /review/i })).toBeEnabled();
  });

  it('asks for the step to be completed again once something is entered', async () => {
    const { user } = render(<ImportWizardGate />);

    expect(await ui.notificationsStep.find()).toBeInTheDocument();
    await user.type(ui.policyTreeNameInput.get(), 'mimir-prod');

    await waitFor(() => expect(ui.nextButton.get()).toHaveAttribute('aria-disabled', 'true'));
    expect(within(ui.stepper.get()).getByRole('button', { name: /rules/i })).toBeDisabled();
  });
});

describe('Import wizard without the Alertmanager import API', () => {
  testWithFeatureToggles({ enable: ['alerting.syncExternalAlertmanager'] });

  beforeEach(() => {
    grantWizardPermissions();
    // Runs after the file-level beforeEach, so this turns the import API back off.
    setTestFlags({});
  });

  it('locks the notifications step and explains why', async () => {
    setupAutoSyncConfig(server, {});

    render(<ImportWizardGate />);

    expect(await ui.importAPIDisabledTitle.find()).toBeInTheDocument();
    // The locked step has nothing to skip, it only offers to continue.
    expect(ui.nextButton.get()).toHaveAttribute('aria-disabled', 'false');
    expect(ui.skipButton.query()).not.toBeInTheDocument();
    expect(ui.blockTitle.query()).not.toBeInTheDocument();
  });

  it('lets the user continue to the rules step', async () => {
    setupAutoSyncConfig(server, {});
    // The rules step renders a data source picker.
    setupDataSources();

    const { user } = render(<ImportWizardGate />);

    expect(await ui.importAPIDisabledTitle.find()).toBeInTheDocument();
    await user.click(ui.nextButton.get());

    expect(await ui.rulesStep.find()).toBeInTheDocument();
    expect(ui.step1SkippedNotice.query()).not.toBeInTheDocument();
  });
});
