import { css } from '@emotion/css';
import { useEffect } from 'react';

import { GrafanaTheme2 } from '@grafana/data';
import { selectors } from '@grafana/e2e-selectors';
import {
  SceneObjectBase,
  SceneObjectRef,
  SceneObjectState,
  SceneObjectUrlSyncConfig,
  SceneObjectUrlValues,
  useSceneObjectState,
  VizPanel,
} from '@grafana/scenes';
import { Container, ScrollContainer, Sidebar, useStyles2, useSiderbar } from '@grafana/ui';
import { getConfig } from 'app/core/config';
import { contextSrv } from 'app/core/core';
import { getRulesPermissions } from 'app/features/alerting/unified/utils/access-control';
import { GRAFANA_RULES_SOURCE_NAME } from 'app/features/alerting/unified/utils/datasource';

import { PanelDataAlertingTab } from './PanelDataAlertingTab';
import { PanelDataQueriesTab } from './PanelDataQueriesTab';
import { PanelDataTransformationsTab } from './PanelDataTransformationsTab';
import { PanelDataPaneTab, TabId } from './types';

export interface PanelDataPaneState extends SceneObjectState {
  tabs: PanelDataPaneTab[];
  tab: TabId;
  panelRef: SceneObjectRef<VizPanel>;
  compact?: boolean;
}

export class PanelDataPane extends SceneObjectBase<PanelDataPaneState> {
  static Component = PanelDataPaneRenderer;
  protected _urlSync = new SceneObjectUrlSyncConfig(this, { keys: ['tab'] });

  public static createFor(panel: VizPanel) {
    const panelRef = panel.getRef();
    const tabs: PanelDataPaneTab[] = [
      new PanelDataQueriesTab({ panelRef }),
      new PanelDataTransformationsTab({ panelRef }),
    ];

    if (shouldShowAlertingTab(panel.state.pluginId)) {
      tabs.push(new PanelDataAlertingTab({ panelRef }));
    }

    return new PanelDataPane({
      panelRef,
      tabs,
      tab: TabId.Queries,
    });
  }

  public onChangeTab = (tab: TabId) => {
    if (this.state.tab === tab) {
      this.setState({ tab: TabId.Closed });
      return;
    }

    this.setState({ tab });
  };

  public getUrlState() {
    return { tab: this.state.tab };
  }

  public updateFromUrl(values: SceneObjectUrlValues) {
    if (!values.tab) {
      return;
    }
    if (typeof values.tab === 'string') {
      this.setState({ tab: values.tab as TabId });
    }
  }
}

export function PanelDataPaneRenderer({
  model,
  collapsed,
  onToggleCollapse,
}: {
  model: PanelDataPane;
  collapsed?: boolean;
  onToggleCollapse?: () => void;
}) {
  const { tab, tabs, compact = true } = useSceneObjectState(model, { shouldActivateOrKeepAlive: true });
  const styles = useStyles2(getStyles);

  useEffect(() => {
    if (tab === TabId.Closed) {
      model.setState({ tab: TabId.Queries });
    }
  }, [tab]);

  if (!tabs || !tabs.length) {
    return;
  }

  const onChangeTab = (newTab: TabId) => {
    if (tab === newTab) {
      model.setState({ tab: TabId.Closed });
      onToggleCollapse?.();
      return;
    }

    model.onChangeTab(newTab);
  };

  const currentTab = tabs.find((t) => t.tabId === tab);
  const position = 'right';

  const { toolbarProps, sidebarProps, openPaneProps } = useSiderbar({
    position: position,
    tabsMode: true,
    compact: !!compact,
  });

  return (
    <div className={styles.dataPane} data-testid={selectors.components.PanelEditor.DataPane.content}>
      <div {...sidebarProps}>
        {currentTab && (
          <div {...openPaneProps}>
            <ScrollContainer minHeight={'100%'}>
              <Container padding={'sm'}>{currentTab && <currentTab.Component model={currentTab} />}</Container>
            </ScrollContainer>
          </div>
        )}
        <div {...toolbarProps} onDoubleClick={() => model.setState({ compact: !compact })}>
          <Sidebar.Button
            icon="database"
            active={tab === TabId.Queries}
            toolbarPosition={position}
            compact={compact}
            title="Queries"
            onClick={() => onChangeTab(TabId.Queries)}
          />
          <Sidebar.Button
            icon="process"
            toolbarPosition={position}
            compact={compact}
            active={tab === TabId.Transformations}
            title="Data"
            tooltip="Data transformations"
            onClick={() => onChangeTab(TabId.Transformations)}
          />
          <Sidebar.Button
            icon="bell"
            toolbarPosition={position}
            compact={compact}
            active={tab === TabId.Alert}
            title="Alerts"
            tooltip="Link alert rule to panel"
            onClick={() => onChangeTab(TabId.Alert)}
          />
        </div>
      </div>
    </div>
  );
}

export function shouldShowAlertingTab(pluginId: string) {
  const { unifiedAlertingEnabled = false } = getConfig();
  const hasRuleReadPermissions = contextSrv.hasPermission(getRulesPermissions(GRAFANA_RULES_SOURCE_NAME).read);
  const isAlertingAvailable = unifiedAlertingEnabled && hasRuleReadPermissions;
  if (!isAlertingAvailable) {
    return false;
  }

  const isGraph = pluginId === 'graph';
  const isTimeseries = pluginId === 'timeseries';

  return isGraph || isTimeseries;
}

function getStyles(theme: GrafanaTheme2) {
  return {
    dataPane: css({
      display: 'flex',
      flexDirection: 'row',
      flexGrow: 1,
      minHeight: 0,
      height: '100%',
      paddingLeft: theme.spacing(2),
    }),
    tabBorder: css({
      background: theme.colors.background.primary,
      border: `1px solid ${theme.colors.border.weak}`,
      borderLeft: 'none',
      borderBottom: 'none',
      borderTopRightRadius: theme.shape.radius.default,
      flexGrow: 1,
      overflow: 'hidden',
    }),
    tabContent: css({
      padding: theme.spacing(2),
      height: '100%',
    }),
    tabsBar: css({
      flexShrink: 0,
      paddingLeft: theme.spacing(2),
    }),
  };
}
