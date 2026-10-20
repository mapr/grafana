import { HttpResponse, http } from 'msw';
import { act, render, screen, testWithFeatureToggles, waitFor, within } from 'test/test-utils';

import { selectors } from '@grafana/e2e-selectors';
import { locationService, reportInteraction } from '@grafana/runtime';
import { FlagKeys } from '@grafana/runtime/internal';
import { setTestFlags } from '@grafana/test-utils/unstable';
import { AccessControlAction } from 'app/types/accessControl';

import { setupMswServer } from '../../mockApi';
import { grantUserPermissions, mockDataSource } from '../../mocks';
import { setupDatasourcesEndpoint } from '../../mocks/server/configure/datasources';
import {
  setupAutoSyncConfig,
  setupAutoSyncConfigAbsent,
  setupAutoSyncConfigWriteError,
  setupStatefulAutoSyncConfig,
} from '../../mocks/server/handlers/k8s/config.k8s';
import { setupDataSources } from '../../testSetup/datasources';
import { DataSourceType } from '../../utils/datasource';

import { ImportWizardGate } from './ImportToGMA';

// Spread requireActual so config/locationService/feature-toggle reads stay real; only stub the
// analytics sink so we can assert the reported payload.
jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  reportInteraction: jest.fn(),
}));

// Selects which fixture the mocked Step1Content below seeds. Prefixed `mock` per Jest's rule for
// variables referenced from inside a jest.mock factory. Each describe block resets it in its own setup.
let mockScenario: 'yaml' | 'auto-sync' = 'yaml';

// Seeds either a valid YAML notifications source (config + policy tree name + template files, dry-run
// triggered) or an Auto-sync-checked data source (dry-run triggered under the name auto-sync will use).
// Next is gated on a passing dry-run for both, so the onTriggerDryRun call is required for the wizard
// to advance. The real step body pulls in network-backed pickers we don't
// need — the assertion target is handleConfirmImport's behavior, not the step UI.
jest.mock('./steps/Step1AlertmanagerResources', () => {
  const { useEffect } = require('react');
  const { useFormContext } = require('react-hook-form');
  return {
    Step1Content: function Step1Content({ onTriggerDryRun }: { onTriggerDryRun?: (configIdentifier: string) => void }) {
      const { setValue } = useFormContext();
      useEffect(() => {
        if (mockScenario === 'auto-sync') {
          setValue('notificationsSource', 'datasource');
          setValue('notificationsDatasourceUID', 'mimir-uid');
          setValue('notificationsDatasourceName', 'Mimir Alertmanager');
          setValue('autoSyncNotificationsEnabled', true);
          queueMicrotask(() => onTriggerDryRun?.('mimir-alertmanager'));
          return;
        }
        setValue('notificationsSource', 'yaml');
        setValue('policyTreeName', 'prometheus-prod');
        setValue(
          'notificationsYamlFile',
          new File(['route:\n  receiver: default\nreceivers:\n  - name: default\n'], 'alertmanager.yaml', {
            type: 'application/yaml',
          })
        );
        setValue('notificationsTemplateFiles', [
          new File(['{{ define "email" }}{{ end }}'], 'email.tmpl', { type: 'text/plain' }),
          new File(['{{ define "slack" }}{{ end }}'], 'slack.tmpl', { type: 'text/plain' }),
        ]);
        // Mounting on the wizard's very first render (Notifications is now step one), these setValue
        // calls aren't guaranteed to be visible via getValues() yet within the same tick — defer so
        // handleTriggerDryRun reads the values above rather than the stale defaults.
        queueMicrotask(() => onTriggerDryRun?.('prometheus-prod'));
      }, [setValue, onTriggerDryRun]);
      return null;
    },
    useStep1Validation: () => true,
  };
});
// Most flows skip Rules, which needs the step left empty: its single button only skips while no rules
// source is chosen. A test that completes the step instead turns this on, so handleConfirmImport has the
// rulesDatasourceUID it needs to fire the import.
let mockSeedRulesSource = false;
jest.mock('./steps/Step2AlertRules', () => {
  const { useEffect } = require('react');
  const { useFormContext } = require('react-hook-form');
  return {
    Step2Content: function Step2Content() {
      const { setValue } = useFormContext();
      useEffect(() => {
        if (mockSeedRulesSource) {
          setValue('rulesDatasourceUID', 'prometheus-uid');
        }
      }, [setValue]);
      return null;
    },
    useStep2Validation: () => true,
  };
});

const CONVERT_URL = '/api/convert/api/v1/alerts';

const server = setupMswServer();

const mockReportInteraction = jest.mocked(reportInteraction);

testWithFeatureToggles({ enable: ['alerting.syncExternalAlertmanager'] });

// The wizard locks its notifications step unless the Alertmanager import API is on.
beforeEach(() => setTestFlags({ [FlagKeys.AlertingImportAlertmanagerAPI]: true }));
// The reset fires OpenFeature events while components are still mounted, so it needs act.
afterEach(async () => {
  await act(async () => {
    setTestFlags();
  });
});

beforeEach(() => {
  mockSeedRulesSource = false;
  mockReportInteraction.mockClear();
  // Default: the notifications import succeeds. Individual tests override this to force a failure.
  server.use(http.post(CONVERT_URL, () => HttpResponse.json({ status: 'success' })));
  grantUserPermissions([
    AccessControlAction.AlertingNotificationsWrite,
    AccessControlAction.AlertingRuleCreate,
    AccessControlAction.AlertingProvisioningSetStatus,
    // useAutoSyncConfiguration (saveAutoSync) gates its Config query on this permission.
    AccessControlAction.ActionAlertingNotificationsConfigRead,
  ]);
  // useAutoSyncConfiguration (called directly by the wizard for saveAutoSync) fires this
  // unconditionally, independent of the mocked Step1Content above.
  setupDatasourcesEndpoint(server, []);
  locationService.push('/');
});

/**
 * Drives the wizard: complete the notifications step, skip the rules step, then open and accept the
 * confirm modal. Leaves the rest to the caller's assertions.
 */
async function importWith(user: ReturnType<typeof render>['user']) {
  await screen.findByRole('group', { name: /import notification configuration/i });
  // Notifications -> Rules: the stub triggers a dry-run; wait for it to pass so Next is enabled
  // (Next is gated on a passing dry-run, and is disabled while blocked). Re-query each poll — the
  // button node is replaced when its disabled-state tooltip wrapper is removed on enable.
  await waitFor(
    () =>
      expect(screen.getByTestId(selectors.pages.Alerting.ImportToGMA.nextButton)).toHaveAttribute(
        'aria-disabled',
        'false'
      ),
    {
      timeout: 3000,
    }
  );
  await user.click(screen.getByTestId(selectors.pages.Alerting.ImportToGMA.nextButton));
  await screen.findByRole('group', { name: /import rules/i });
  // Skip Rules -> Review
  await user.click(await screen.findByTestId(selectors.pages.Alerting.ImportToGMA.nextButton));
  // Review -> open confirm modal
  await user.click(await screen.findByRole('button', { name: /start import/i }));
  // Confirm inside the modal
  const dialog = await screen.findByRole('dialog');
  await user.click(within(dialog).getByRole('button', { name: /start import/i }));
}

describe('ImportWizardGate — gating on mount', () => {
  it('locks the notifications step when auto-sync is already active on mount', async () => {
    setupAutoSyncConfig(server, { specUid: 'mimir-uid' });

    render(<ImportWizardGate />);

    expect(await screen.findByText(/auto-sync is enabled/i)).toBeInTheDocument();
    // The locked step has nothing to skip, it only offers to continue.
    expect(screen.getByTestId(selectors.pages.Alerting.ImportToGMA.nextButton)).toHaveAttribute(
      'aria-disabled',
      'false'
    );
    expect(screen.queryByTestId(selectors.pages.Alerting.ImportToGMA.skipButton)).not.toBeInTheDocument();
  });
});

describe('ImportToGMA wizard — opened on the rules step', () => {
  it('starts on the rules step and can move on to the review from there', async () => {
    const { user } = render(<ImportWizardGate />, {
      historyOptions: { initialEntries: ['/alerting/import-to-gma?step=rules'] },
    });

    expect(await screen.findByRole('group', { name: /import rules/i })).toBeInTheDocument();
    expect(screen.queryByRole('group', { name: /import notification configuration/i })).not.toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.pages.Alerting.ImportToGMA.nextButton));
    expect(await screen.findByText(/review import/i)).toBeInTheDocument();
  });
});

describe('ImportToGMA wizard — stage analytics', () => {
  it('tracks success and lands on the Import settings tab', async () => {
    const { user } = render(<ImportWizardGate />);

    await importWith(user);

    await waitFor(() =>
      expect(mockReportInteraction).toHaveBeenCalledWith(
        'grafana_alerting_import_to_gma_success',
        expect.objectContaining({ notificationsSource: 'yaml' })
      )
    );
    await waitFor(() => expect(locationService.getLocation().pathname).toContain('/alerting/admin/import'), {
      timeout: 3000,
    });
  });

  it('tracks an error when the import fails', async () => {
    // Only fail the real import — the dry-run (same URL, distinguished by the dry-run header) must
    // still pass so the wizard can advance to the confirm step under the passing-dry-run gate.
    server.use(
      http.post(CONVERT_URL, ({ request }) =>
        request.headers.get('X-Grafana-Alerting-Dry-Run') === 'true'
          ? HttpResponse.json({ status: 'success' })
          : new HttpResponse(null, { status: 500 })
      )
    );
    const { user } = render(<ImportWizardGate />);

    await importWith(user);

    await waitFor(() =>
      expect(mockReportInteraction).toHaveBeenCalledWith(
        'grafana_alerting_import_to_gma_error',
        expect.objectContaining({ notificationsSource: 'yaml' })
      )
    );
    expect(mockReportInteraction).not.toHaveBeenCalledWith('grafana_alerting_import_to_gma_success', expect.anything());
    expect(locationService.getLocation().pathname).not.toContain('/alerting/list');
  });
});

describe('ImportToGMA wizard — step 1 dry-run gating & review', () => {
  it('keeps the notifications-step Next disabled when the dry-run fails', async () => {
    // Fail the dry-run itself, so the step never reaches a passing validation state.
    server.use(
      http.post(CONVERT_URL, ({ request }) =>
        request.headers.get('X-Grafana-Alerting-Dry-Run') === 'true'
          ? new HttpResponse(null, { status: 500 })
          : HttpResponse.json({ status: 'success' })
      )
    );
    const { user } = render(<ImportWizardGate />);

    await screen.findByRole('group', { name: /import notification configuration/i });

    // The dry-run runs and fails; Next stays disabled (aria-disabled keeps the tooltip reachable).
    await waitFor(() =>
      expect(mockReportInteraction).toHaveBeenCalledWith('grafana_alerting_import_to_gma_dryrun_error')
    );
    const nextButton = screen.getByTestId(selectors.pages.Alerting.ImportToGMA.nextButton);
    expect(nextButton).toHaveAttribute('aria-disabled', 'true');

    // Clicking a blocked Next must not advance to the rules step.
    await user.click(nextButton);
    expect(screen.queryByRole('group', { name: /import rules/i })).not.toBeInTheDocument();
  });

  it('lists the uploaded template files in the review step', async () => {
    const { user } = render(<ImportWizardGate />);

    await screen.findByRole('group', { name: /import notification configuration/i });
    // Notifications -> Rules (wait for the seeded dry-run to pass; re-query — the button node is
    // replaced when its disabled-state tooltip wrapper is removed on enable).
    await waitFor(
      () =>
        expect(screen.getByTestId(selectors.pages.Alerting.ImportToGMA.nextButton)).toHaveAttribute(
          'aria-disabled',
          'false'
        ),
      {
        timeout: 3000,
      }
    );
    await user.click(screen.getByTestId(selectors.pages.Alerting.ImportToGMA.nextButton));
    await screen.findByRole('group', { name: /import rules/i });
    // Skip Rules -> Review
    await user.click(await screen.findByTestId(selectors.pages.Alerting.ImportToGMA.nextButton));

    // The review notifications card lists the uploaded template files by name.
    expect(await screen.findByText('email.tmpl, slack.tmpl')).toBeInTheDocument();
  });

  it("opens the matching preview modal from each card's Preview button on the review step", async () => {
    mockSeedRulesSource = true;
    const { user } = render(<ImportWizardGate />);

    await screen.findByRole('group', { name: /import notification configuration/i });
    await waitFor(() =>
      expect(screen.getByTestId(selectors.pages.Alerting.ImportToGMA.nextButton)).toHaveAttribute(
        'aria-disabled',
        'false'
      )
    );
    await user.click(screen.getByTestId(selectors.pages.Alerting.ImportToGMA.nextButton));
    await screen.findByRole('group', { name: /import rules/i });
    // Complete Rules instead of skipping it, so its card also renders a Preview control on Review.
    await user.click(screen.getByTestId(selectors.pages.Alerting.ImportToGMA.nextButton));
    await screen.findByText(/review import/i);

    // aria-label disambiguates the two cards' otherwise-identical "Preview" buttons.
    await user.click(await screen.findByRole('button', { name: /preview configuration/i }));
    const notificationsDialog = await screen.findByRole('dialog', { name: /notifications config preview/i });
    // Modal renders both its own close-icon button and the explicit "Close" button below the content.
    const closeButtons = within(notificationsDialog).getAllByRole('button', { name: /close/i });
    await user.click(closeButtons[closeButtons.length - 1]);

    await user.click(await screen.findByRole('button', { name: /preview alert rules/i }));
    expect(await screen.findByRole('dialog', { name: /alert rules preview/i })).toBeInTheDocument();
  });
});

describe('ImportToGMA wizard — auto-sync confirm flow', () => {
  beforeEach(() => {
    mockScenario = 'auto-sync';
    // The dry-run reads the configuration from the selected Alertmanager before checking it.
    setupDataSources(
      mockDataSource({ uid: 'mimir-uid', name: 'Mimir Alertmanager', type: DataSourceType.Alertmanager })
    );
    server.use(
      http.get('/api/alertmanager/:name/config/api/v1/alerts', () =>
        HttpResponse.json({
          template_files: {},
          alertmanager_config: { route: { receiver: 'default' }, receivers: [{ name: 'default' }] },
        })
      )
    );
  });

  afterEach(() => {
    mockScenario = 'yaml';
  });

  /** Notifications -> Rules (now always reachable, even with Auto-sync checked). */
  async function advanceToRules(user: ReturnType<typeof render>['user']) {
    await screen.findByRole('group', { name: /import notification configuration/i });
    await waitFor(() =>
      expect(screen.getByTestId(selectors.pages.Alerting.ImportToGMA.nextButton)).toHaveAttribute(
        'aria-disabled',
        'false'
      )
    );
    await user.click(screen.getByTestId(selectors.pages.Alerting.ImportToGMA.nextButton));
    await screen.findByRole('group', { name: /import rules/i });
  }

  it('still shows the Rules step when Auto-sync is checked, and lets it be skipped', async () => {
    const { user } = render(<ImportWizardGate />);

    await advanceToRules(user);

    await user.click(await screen.findByTestId(selectors.pages.Alerting.ImportToGMA.nextButton));
    await screen.findByText(/review import/i);
    expect(screen.getByText(/will sync continuously/i)).toBeInTheDocument();
    // Both cards are in a static, non-interactive state here (auto-sync badge, Skipped) — neither
    // should render a Preview control.
    expect(screen.queryByRole('button', { name: /preview/i })).not.toBeInTheDocument();
  });

  // A filled-in step can't be skipped, so an Auto-sync selection can't be left behind by a skip. Moving on
  // without it means unticking it first. isAutoSyncCommitted (steps.test.ts) still guards a skipped step.
  it('offers no skip once Auto-sync is selected, only continuing with it', async () => {
    setupStatefulAutoSyncConfig(server);

    render(<ImportWizardGate />);

    await screen.findByRole('group', { name: /import notification configuration/i });
    await waitFor(() =>
      expect(screen.getByTestId(selectors.pages.Alerting.ImportToGMA.nextButton)).toHaveAttribute(
        'aria-disabled',
        'false'
      )
    );
    expect(screen.queryByTestId(selectors.pages.Alerting.ImportToGMA.skipButton)).not.toBeInTheDocument();
    expect(screen.getByTestId(selectors.pages.Alerting.ImportToGMA.nextButton)).toHaveTextContent(/^continue$/i);
  });

  it('calls saveAutoSync (not the staging import) and tracks success when Rules is skipped', async () => {
    const { getStored } = setupStatefulAutoSyncConfig(server);
    let stagingImportCalled = false;
    server.use(
      http.post(CONVERT_URL, ({ request }) => {
        // The dry-run that checks the configuration goes to the same endpoint; only a real import counts.
        if (request.headers.get('X-Grafana-Alerting-Dry-Run') !== 'true') {
          stagingImportCalled = true;
        }
        return HttpResponse.json({ status: 'success' });
      })
    );

    const { user } = render(<ImportWizardGate />);
    await advanceToRules(user);
    await user.click(await screen.findByTestId(selectors.pages.Alerting.ImportToGMA.nextButton));
    await screen.findByText(/review import/i);

    await user.click(screen.getByRole('button', { name: /enable auto-sync/i }));
    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: /enable auto-sync/i }));

    await waitFor(() => expect(getStored().spec.externalAlertmanagerSync).toEqual({ datasourceUid: 'mimir-uid' }));
    expect(stagingImportCalled).toBe(false);
    await waitFor(() =>
      expect(mockReportInteraction).toHaveBeenCalledWith(
        'grafana_alerting_import_to_gma_success',
        expect.objectContaining({ notificationsSource: 'datasource' })
      )
    );
    await waitFor(() => expect(locationService.getLocation().pathname).toContain('/alerting/admin/import'), {
      timeout: 3000,
    });
  });

  it('enables auto-sync and imports rules together when Rules is completed instead of skipped', async () => {
    mockSeedRulesSource = true;
    const { getStored } = setupStatefulAutoSyncConfig(server);
    let rulesImportCalled = false;
    server.use(
      // Confirmed via convertToGMAApi.ts: useConvertToGMAMutation (rules import) posts here,
      // a different endpoint from CONVERT_URL (notifications/dry-run only).
      http.post('/api/convert/prometheus/config/v1/rules', () => {
        rulesImportCalled = true;
        return HttpResponse.json({});
      })
    );

    const { user } = render(<ImportWizardGate />);
    await advanceToRules(user);
    // Step2Content renders null and useStep2Validation is stubbed true (see the mock above), so
    // Next is enabled immediately without needing the real rules-picking UI.
    await user.click(screen.getByTestId(selectors.pages.Alerting.ImportToGMA.nextButton));
    await screen.findByText(/review import/i);

    await user.click(screen.getByRole('button', { name: /enable auto-sync/i }));
    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: /enable auto-sync/i }));

    await waitFor(() => expect(getStored().spec.externalAlertmanagerSync).toEqual({ datasourceUid: 'mimir-uid' }));
    expect(rulesImportCalled).toBe(true);
    await waitFor(() => expect(within(dialog).getByText(/alert rules were also imported/i)).toBeInTheDocument());
  });

  it('reports a Rules-specific failure — not an Auto-sync failure — when Rules import fails after Auto-sync already succeeded', async () => {
    mockSeedRulesSource = true;
    const { getStored } = setupStatefulAutoSyncConfig(server);
    server.use(http.post('/api/convert/prometheus/config/v1/rules', () => new HttpResponse(null, { status: 500 })));

    const { user } = render(<ImportWizardGate />);
    await advanceToRules(user);
    await user.click(screen.getByTestId(selectors.pages.Alerting.ImportToGMA.nextButton));
    await screen.findByText(/review import/i);

    await user.click(screen.getByRole('button', { name: /enable auto-sync/i }));
    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: /enable auto-sync/i }));

    // Auto-sync's own save call succeeded even though the overall submit errors out below.
    await waitFor(() => expect(getStored().spec.externalAlertmanagerSync).toEqual({ datasourceUid: 'mimir-uid' }));
    await waitFor(() => expect(within(dialog).getByText(/rules import failed/i)).toBeInTheDocument());
    expect(within(dialog).queryByText(/failed to enable auto-sync/i)).not.toBeInTheDocument();
    await waitFor(() =>
      expect(mockReportInteraction).toHaveBeenCalledWith(
        'grafana_alerting_import_to_gma_error',
        expect.objectContaining({ notificationsSource: 'datasource' })
      )
    );
  });

  it('tracks an error and does not fall through to the staging import path when saveAutoSync fails', async () => {
    setupStatefulAutoSyncConfig(server);
    setupAutoSyncConfigWriteError(server, { code: 500, message: 'failed to save the configuration' });

    const { user } = render(<ImportWizardGate />);
    await advanceToRules(user);
    await user.click(await screen.findByTestId(selectors.pages.Alerting.ImportToGMA.nextButton));
    await screen.findByText(/review import/i);

    await user.click(screen.getByRole('button', { name: /enable auto-sync/i }));
    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: /enable auto-sync/i }));

    await waitFor(() =>
      expect(mockReportInteraction).toHaveBeenCalledWith(
        'grafana_alerting_import_to_gma_error',
        expect.objectContaining({ notificationsSource: 'datasource' })
      )
    );
    expect(mockReportInteraction).not.toHaveBeenCalledWith('grafana_alerting_import_to_gma_success', expect.anything());
    expect(within(dialog).getByText(/failed to enable auto-sync/i)).toBeInTheDocument();
  });

  it('reports an error and does not fall through to the staging import path when the Config singleton has not been seeded yet', async () => {
    // Humans can't create the Config singleton — this is the pre-seed state, not a write failure.
    setupAutoSyncConfigAbsent(server);

    const { user } = render(<ImportWizardGate />);
    await advanceToRules(user);
    await user.click(await screen.findByTestId(selectors.pages.Alerting.ImportToGMA.nextButton));
    await screen.findByText(/review import/i);

    await user.click(screen.getByRole('button', { name: /enable auto-sync/i }));
    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: /enable auto-sync/i }));

    await waitFor(() =>
      expect(mockReportInteraction).toHaveBeenCalledWith(
        'grafana_alerting_import_to_gma_error',
        expect.objectContaining({ notificationsSource: 'datasource' })
      )
    );
    expect(mockReportInteraction).not.toHaveBeenCalledWith('grafana_alerting_import_to_gma_success', expect.anything());
    expect(within(dialog).getByText(/failed to enable auto-sync/i)).toBeInTheDocument();
  });
});
