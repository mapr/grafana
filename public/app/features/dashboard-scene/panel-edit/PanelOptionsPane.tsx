import { css } from '@emotion/css';
import { useMemo } from 'react';
import { useToggle } from 'react-use';

import {
  FieldConfigSource,
  filterFieldConfigOverrides,
  GrafanaTheme2,
  isStandardFieldProp,
  PanelPluginMeta,
  restoreCustomOverrideRules,
  SelectableValue,
} from '@grafana/data';
import { selectors } from '@grafana/e2e-selectors';
import { t } from '@grafana/i18n';
import { locationService, reportInteraction } from '@grafana/runtime';
import {
  DeepPartial,
  SceneComponentProps,
  SceneObjectBase,
  SceneObjectRef,
  SceneObjectState,
  VizPanel,
  sceneGraph,
} from '@grafana/scenes';
import { ScrollContainer, ToolbarButton, useStyles2, useSiderbar, Sidebar, Box } from '@grafana/ui';
import { OptionFilter } from 'app/features/dashboard/components/PanelEditor/OptionsPaneOptions';
import { VisualizationSelectPaneTab } from 'app/features/dashboard/components/PanelEditor/types';
import { getPanelPluginNotFound } from 'app/features/panel/components/PanelPluginError';
import { VizTypeChangeDetails } from 'app/features/panel/components/VizTypePicker/types';
import { getAllPanelPluginMeta } from 'app/features/panel/state/util';

import { PanelOptions } from './PanelOptions';
import { PanelVizTypePicker } from './PanelVizTypePicker';
import { INTERACTION_EVENT_NAME, INTERACTION_ITEM } from './interaction';
import { useScrollReflowLimit } from './useScrollReflowLimit';

export interface PanelOptionsPaneState extends SceneObjectState {
  openView?: string;
  searchQuery: string;
  listMode: OptionFilter;
  panelRef: SceneObjectRef<VizPanel>;
  compact?: boolean;
}

interface PluginOptionsCache {
  options: DeepPartial<{}>;
  fieldConfig: FieldConfigSource<DeepPartial<{}>>;
}

export class PanelOptionsPane extends SceneObjectBase<PanelOptionsPaneState> {
  private _cachedPluginOptions: Record<string, PluginOptionsCache | undefined> = {};

  // onToggleVizPicker = () => {
  //   reportInteraction(INTERACTION_EVENT_NAME, {
  //     item: INTERACTION_ITEM.TOGGLE_DROPDOWN,
  //     open: !this.state.isVizPickerOpen,
  //   });
  //   this.setState({ isVizPickerOpen: !this.state.isVizPickerOpen });
  // };

  onChangePanelPlugin = (options: VizTypeChangeDetails) => {
    const panel = this.state.panelRef.resolve();
    const { options: prevOptions, fieldConfig: prevFieldConfig, pluginId: prevPluginId } = panel.state;
    const pluginId = options.pluginId;

    reportInteraction(INTERACTION_EVENT_NAME, {
      item: INTERACTION_ITEM.SELECT_PANEL_PLUGIN,
      plugin_id: pluginId,
    });

    // clear custom options
    let newFieldConfig: FieldConfigSource = {
      defaults: {
        ...prevFieldConfig.defaults,
        custom: {},
      },
      overrides: filterFieldConfigOverrides(prevFieldConfig.overrides, isStandardFieldProp),
    };

    this._cachedPluginOptions[prevPluginId] = { options: prevOptions, fieldConfig: prevFieldConfig };

    const cachedOptions = this._cachedPluginOptions[pluginId]?.options;
    const cachedFieldConfig = this._cachedPluginOptions[pluginId]?.fieldConfig;

    if (cachedFieldConfig) {
      newFieldConfig = restoreCustomOverrideRules(newFieldConfig, cachedFieldConfig);
    }

    panel.changePluginType(pluginId, options.options, newFieldConfig);

    if (options.options) {
      panel.onOptionsChange(options.options, true);
    }

    if (options.fieldConfig) {
      const fieldConfigWithOverrides = {
        ...options.fieldConfig,
        overrides: newFieldConfig.overrides,
      };
      panel.onFieldConfigChange(fieldConfigWithOverrides, true);
    }

    // this.setState({ openView: 'settings' });
  };

  onOpenView = (openView: string) => {
    if (openView === this.state.openView) {
      openView = 'closed';
    }

    this.setState({ openView });
  };

  onSetSearchQuery = (searchQuery: string) => {
    this.setState({ searchQuery });
  };

  onSetListMode = (listMode: OptionFilter) => {
    this.setState({ listMode });
  };

  onOpenPanelJSON = (vizPanel: VizPanel) => {
    locationService.partial({
      inspect: vizPanel.state.key,
      inspectTab: 'json',
    });
  };

  getOptionRadioFilters(): Array<SelectableValue<OptionFilter>> {
    return [
      { label: OptionFilter.All, value: OptionFilter.All },
      { label: OptionFilter.Overrides, value: OptionFilter.Overrides },
    ];
  }

  static Component = PanelOptionsPaneComponent;
}

function PanelOptionsPaneComponent({ model }: SceneComponentProps<PanelOptionsPane>) {
  const { openView = 'settings', searchQuery, listMode, panelRef, compact = false } = model.useState();
  const panel = panelRef.resolve();
  const { pluginId } = panel.useState();
  const { data } = sceneGraph.getData(panel).useState();
  const styles = useStyles2(getStyles);
  const isSearching = searchQuery.length > 0;
  const hasFieldConfig = !isSearching && !panel.getPlugin()?.fieldConfigRegistry.isEmpty();
  const [isSearchingOptions, setIsSearchingOptions] = useToggle(false);
  const onlyOverrides = listMode === OptionFilter.Overrides;
  const isScrollingLayout = useScrollReflowLimit();

  const { toolbarProps, sidebarProps, openPaneProps } = useSiderbar({
    position: 'right',
    tabsMode: true,
  });

  return (
    <div {...sidebarProps}>
      {openView === 'settings' && (
        <div {...openPaneProps}>
          <Sidebar.PaneHeader title="All options" onClose={() => model.setState({ openView: 'closed' })} />
          <ScrollContainer minHeight={isScrollingLayout ? 'max-content' : 0}>
            <PanelOptions panel={panel} searchQuery={searchQuery} listMode={listMode} data={data} quickMode={false} />
          </ScrollContainer>
        </div>
      )}
      {openView === 'quick' && (
        <div {...openPaneProps}>
          <Sidebar.PaneHeader title="Quick options" onClose={() => model.setState({ openView: 'closed' })} />
          <ScrollContainer minHeight={isScrollingLayout ? 'max-content' : 0}>
            <Box padding={2}>
              <PanelOptions panel={panel} searchQuery={searchQuery} listMode={listMode} data={data} quickMode={true} />
            </Box>
          </ScrollContainer>
        </div>
      )}
      {openView === 'presets' && (
        <div {...openPaneProps}>
          <PanelVizTypePicker
            panel={panel}
            onChange={model.onChangePanelPlugin}
            onClose={() => model.setState({ openView: 'closed' })}
            listMode={VisualizationSelectPaneTab.Presets}
            data={data}
          />
        </div>
      )}
      {openView === 'viz-picker' && (
        <div {...openPaneProps}>
          <PanelVizTypePicker
            panel={panel}
            onChange={model.onChangePanelPlugin}
            onClose={() => {}}
            listMode={VisualizationSelectPaneTab.Suggestions}
            data={data}
          />
        </div>
      )}
      <div {...toolbarProps} onDoubleClick={() => model.setState({ compact: !compact })}>
        {/* <ToolbarButton icon="save" tooltip="Save" variant="primary" />
        <ToolbarButton icon="arrow-left" tooltip="Back to dashboard" />
        <Sidebar.Divider /> */}
        <Sidebar.Button
          icon="rocket"
          active={openView === 'quick'}
          onClick={() => model.onOpenView('quick')}
          title="Quick"
          tooltip="Quick options"
          compact={compact}
        />
        <Sidebar.Button
          icon="sliders-v-alt"
          active={openView === 'settings'}
          onClick={() => model.onOpenView('settings')}
          title="Options"
          compact={compact}
        />
        <Sidebar.Button
          icon="palette"
          active={openView === 'presets'}
          onClick={() => model.onOpenView('presets')}
          title="Presets"
          tooltip="Visualization presets"
          compact={compact}
        />
        <Sidebar.Button
          icon="graph-bar"
          active={openView === 'viz-picker'}
          onClick={() => model.onOpenView('viz-picker')}
          compact={compact}
          title="Change"
          tooltip="Change visualization"
        />
        {/* <Sidebar.Button icon="search" title="Search" compact={compact} tooltip="Search all options" /> */}
      </div>
    </div>
  );
}

function getStyles(theme: GrafanaTheme2) {
  return {
    top: css({
      display: 'flex',
      flexDirection: 'column',
      padding: theme.spacing(1, 2, 2, 2),
      gap: theme.spacing(2),
    }),
    searchOptions: css({
      minHeight: theme.spacing(4),
    }),
    searchWrapper: css({
      padding: theme.spacing(2, 2, 2, 0),
    }),
    rotateIcon: css({
      rotate: '180deg',
    }),
  };
}

interface VisualizationButtonProps {
  pluginId: string;
  onOpen: () => void;
}

export function VisualizationButton({ pluginId, onOpen }: VisualizationButtonProps) {
  const styles = useStyles2(getVizButtonStyles);
  let pluginMeta: PanelPluginMeta | undefined = useMemo(
    () => getAllPanelPluginMeta().filter((p) => p.id === pluginId)[0],
    [pluginId]
  );

  if (!pluginMeta) {
    const notFound = getPanelPluginNotFound(`Panel plugin not found (${pluginId})`, true);
    pluginMeta = notFound.meta;
  }

  return (
    <ToolbarButton
      className={styles.vizButton}
      tooltip={t(
        'dashboard-scene.visualization-button.tooltip-click-to-change-visualization',
        'Click to change visualization'
      )}
      imgSrc={pluginMeta.info.logos.small}
      onClick={onOpen}
      data-testid={selectors.components.PanelEditor.toggleVizPicker}
      aria-label={t('dashboard-scene.visualization-button.aria-label-change-visualization', 'Change visualization')}
      variant="canvas"
      isOpen={false}
      fullWidth
    >
      {pluginMeta.name}
    </ToolbarButton>
  );
}

function getVizButtonStyles(theme: GrafanaTheme2) {
  return {
    vizButton: css({
      textAlign: 'left',
    }),
  };
}
