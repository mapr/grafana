import { css } from '@emotion/css';
import { debounce } from 'lodash';
import { useMemo, useState } from 'react';

import { GrafanaTheme2, PanelData } from '@grafana/data';
import { t } from '@grafana/i18n';
import { reportInteraction } from '@grafana/runtime';
import { VizPanel } from '@grafana/scenes';
import { ScrollContainer, Sidebar, useStyles2 } from '@grafana/ui';
import { VisualizationSelectPaneTab } from 'app/features/dashboard/components/PanelEditor/types';
import { VisualizationSuggestions } from 'app/features/panel/components/VizTypePicker/VisualizationSuggestions';
import { VizTypePicker } from 'app/features/panel/components/VizTypePicker/VizTypePicker';
import { VizTypeChangeDetails } from 'app/features/panel/components/VizTypePicker/types';

import { PanelModelCompatibilityWrapper } from '../utils/PanelModelCompatibilityWrapper';

import { INTERACTION_EVENT_NAME, INTERACTION_ITEM } from './interaction';

export interface Props {
  data?: PanelData;
  panel: VizPanel;
  onChange: (options: VizTypeChangeDetails) => void;
  onClose: () => void;
  listMode?: VisualizationSelectPaneTab;
}

export function PanelVizTypePicker({ panel, data, onChange, onClose, listMode }: Props) {
  const styles = useStyles2(getStyles);
  const [searchQuery, setSearchQuery] = useState('');
  const trackSearch = useMemo(
    () =>
      debounce((q, count) => {
        if (q) {
          reportInteraction(INTERACTION_EVENT_NAME, {
            item: INTERACTION_ITEM.SEARCH,
            query: q,
            result_count: count,
            creator_team: 'grafana_plugins_catalog',
            schema_version: '1.0.0',
          });
        }
      }, 300),
    []
  );

  const panelModel = useMemo(() => new PanelModelCompatibilityWrapper(panel), [panel]);

  const onPresetSelected = (options: VizTypeChangeDetails) => {
    onChange(options);
    // onClose();
  };

  return (
    <div className={styles.wrapper}>
      {listMode === VisualizationSelectPaneTab.Presets && (
        <Sidebar.PaneHeader
          title={t('dashboard-scene.panel-viz-type-picker.title', 'Visualization presets')}
          onClose={onClose}
        />
      )}

      {listMode === VisualizationSelectPaneTab.Suggestions && (
        <Sidebar.PaneHeader
          title={t('dashboard-scene.panel-viz-type-picker.title', 'Change visualization')}
          onClose={onClose}
        />
      )}

      {/* <FilterInput
          className={styles.filter}
          value={searchQuery}
          onChange={handleSearchChange}
          autoFocus={true}
          placeholder={t('dashboard-scene.panel-viz-type-picker.placeholder-search-for', 'Search for...')}
        />
        <Button
          aria-label={t('dashboard-scene.panel-viz-type-picker.title-close', 'Close')}
          variant="secondary"
          icon="angle-up"
          className={styles.closeButton}
          data-testid={selectors.components.PanelEditor.toggleVizPicker}
          onClick={onClose}
        />
      </div>
      <Field className={styles.customFieldMargin}>
        <RadioButtonGroup options={radioOptions} value={listMode} onChange={handleListModeChange} fullWidth />
      </Field> */}
      <ScrollContainer>
        {listMode === VisualizationSelectPaneTab.Visualizations && (
          <VizTypePicker
            pluginId={panel.state.pluginId}
            searchQuery={searchQuery}
            trackSearch={trackSearch}
            onChange={onChange}
          />
        )}
        {listMode === VisualizationSelectPaneTab.Suggestions && (
          <VisualizationSuggestions
            onChange={onPresetSelected}
            trackSearch={trackSearch}
            searchQuery={searchQuery}
            panel={panelModel}
            presets={false}
            data={data}
          />
        )}
        {listMode === VisualizationSelectPaneTab.Presets && (
          <VisualizationSuggestions
            onChange={onPresetSelected}
            trackSearch={trackSearch}
            searchQuery={searchQuery}
            panel={panelModel}
            presets={true}
            data={data}
          />
        )}
      </ScrollContainer>
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  wrapper: css({
    display: 'flex',
    flexDirection: 'column',
    flexGrow: 1,
    height: '100%',
  }),
  closeButton: css({
    marginLeft: theme.spacing(1),
  }),
  customFieldMargin: css({
    marginBottom: theme.spacing(1),
  }),
  filter: css({
    minHeight: theme.spacing(4),
  }),
});
