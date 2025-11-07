import { css, cx } from '@emotion/css';

import { GrafanaTheme2 } from '@grafana/data';
import { selectors } from '@grafana/e2e-selectors';
import { t } from '@grafana/i18n';
import { SceneComponentProps, VizPanel } from '@grafana/scenes';
import { Button, Spinner, useStyles2 } from '@grafana/ui';

import { NavToolbarActions } from '../scene/NavToolbarActions';
import { UnlinkModal } from '../scene/UnlinkModal';
import { getDashboardSceneFor, getLibraryPanelBehavior } from '../utils/utils';

import { PanelDataPaneRenderer } from './PanelDataPane/PanelDataPane';
import { PanelEditor } from './PanelEditor';
import { SaveLibraryVizPanelModal } from './SaveLibraryVizPanelModal';
import { useSnappingSplitter } from './splitter/useSnappingSplitter';
import { scrollReflowMediaCondition, useScrollReflowLimit } from './useScrollReflowLimit';

export function PanelEditorRenderer({ model }: SceneComponentProps<PanelEditor>) {
  const dashboard = getDashboardSceneFor(model);
  const { controls } = dashboard.useState();
  const { optionsPane } = model.useState();
  const styles = useStyles2(getStyles);

  return (
    <>
      {/* <NavToolbarActions dashboard={dashboard} /> */}
      <div className={cx(styles.pageContainer)}>
        {controls && (
          <div className={styles.controlsWrapper}>
            <controls.Component model={controls} />
          </div>
        )}
        <div className={styles.body}>
          <VizAndDataPane model={model} />
        </div>

        {/* <div className={styles.optionsPane}>
            {optionsPane && <optionsPane.Component model={optionsPane} />}
            {!optionsPane && <Spinner />}
          </div> */}
      </div>
    </>
  );
}

function VizAndDataPane({ model }: SceneComponentProps<PanelEditor>) {
  const { dataPane, showLibraryPanelSaveModal, showLibraryPanelUnlinkModal, tableView, optionsPane } = model.useState();
  const panel = model.getPanel();
  const libraryPanel = getLibraryPanelBehavior(panel);

  const styles = useStyles2(getStyles);

  const isScrollingLayout = useScrollReflowLimit();

  const { containerProps, primaryProps, secondaryProps, splitterProps, splitterState, onToggleCollapse } =
    useSnappingSplitter({
      direction: 'column',
      dragPosition: 'start',
      initialSize: 0.5,
      collapseBelowPixels: 150,
      disabled: isScrollingLayout,
    });

  containerProps.className = cx(containerProps.className, styles.container);

  if (!dataPane && !isScrollingLayout) {
    primaryProps.style.flexGrow = 1;
  }

  return (
    <div className={styles.dataPane}>
      <div {...containerProps}>
        <div {...primaryProps} className={cx(primaryProps.className, isScrollingLayout && styles.fixedSizeViz)}>
          <div className={styles.content}>
            <VizWrapper panel={panel} tableView={tableView} />
            <div className={styles.optionsPane}>
              {optionsPane && <optionsPane.Component model={optionsPane} />}
              {!optionsPane && <Spinner />}
            </div>
          </div>
        </div>
        {showLibraryPanelSaveModal && libraryPanel && (
          <SaveLibraryVizPanelModal
            libraryPanel={libraryPanel}
            onDismiss={model.onDismissLibraryPanelSaveModal}
            onConfirm={model.onConfirmSaveLibraryPanel}
            onDiscard={model.onDiscard}
          ></SaveLibraryVizPanelModal>
        )}
        {showLibraryPanelUnlinkModal && libraryPanel && (
          <UnlinkModal
            onDismiss={model.onDismissUnlinkLibraryPanelModal}
            onConfirm={model.onConfirmUnlinkLibraryPanel}
            isOpen
          />
        )}
        {dataPane && (
          <>
            <div {...splitterProps} />
            <div
              {...secondaryProps}
              className={cx(secondaryProps.className, isScrollingLayout && styles.fullSizeEditor)}
            >
              {splitterState.collapsed && (
                <div className={styles.expandDataPane}>
                  <Button
                    tooltip={t('dashboard-scene.viz-and-data-pane.tooltip-open-query-pane', 'Open query pane')}
                    icon={'arrow-to-right'}
                    onClick={onToggleCollapse}
                    variant="secondary"
                    fill="text"
                    fullWidth={true}
                    size="sm"
                    className={styles.openDataPaneButton}
                    aria-label={t('dashboard-scene.viz-and-data-pane.aria-label-open-query-pane', 'Open query pane')}
                  />
                </div>
              )}
              {!splitterState.collapsed && (
                <PanelDataPaneRenderer
                  model={dataPane}
                  onToggleCollapse={onToggleCollapse}
                  collapsed={splitterState.collapsed}
                />
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}

interface VizWrapperProps {
  panel: VizPanel;
  tableView?: VizPanel;
}

function VizWrapper({ panel, tableView }: VizWrapperProps) {
  const styles = useStyles2(getStyles);
  const panelToShow = tableView ?? panel;

  return (
    <div className={styles.vizWrapper}>
      <panelToShow.Component model={panelToShow} />
    </div>
  );
}

function getStyles(theme: GrafanaTheme2) {
  const scrollReflowMediaQuery = '@media ' + scrollReflowMediaCondition;
  return {
    pageContainer: css({
      display: 'flex',
      flexDirection: 'column',
      flex: '1 1 0',
      position: 'absolute',
      width: '100%',
      height: '100%',
      overflow: 'hidden',
    }),
    dataPane: css({
      display: 'flex',
      flexDirection: 'row',
      flexGrow: 1,
    }),
    container: css({
      gridArea: 'panels',
      height: '100%',
    }),
    canvasContent: css({
      label: 'canvas-content',
      display: 'flex',
      flexDirection: 'column',
      flexBasis: '100%',
      flexGrow: 1,
      minHeight: 0,
      width: '100%',
    }),
    content: css({
      display: 'flex',
      width: '100%',
      flexGrow: 1,
      overflow: 'hidden',
      minHeight: 0,
      gap: theme.spacing(2),
      [scrollReflowMediaQuery]: {
        height: 'auto',
        display: 'grid',
        gridTemplateColumns: 'minmax(470px, 1fr) 330px',
        gridTemplateRows: '1fr',
        gap: theme.spacing(1),
        position: 'static',
        width: '100%',
      },
    }),
    body: css({
      label: 'body',
      flexGrow: 1,
      display: 'flex',
      flexDirection: 'column',
      minHeight: 0,
    }),
    optionsPane: css({
      display: 'flex',
      flexDirection: 'column',
      background: theme.colors.background.primary,
    }),
    expandOptionsWrapper: css({
      display: 'flex',
      flexDirection: 'column',
      padding: theme.spacing(2, 1),
    }),
    expandDataPane: css({
      display: 'flex',
      flexDirection: 'row',
      padding: theme.spacing(1),
      borderTop: `1px solid ${theme.colors.border.weak}`,
      borderRight: `1px solid ${theme.colors.border.weak}`,
      background: theme.colors.background.primary,
      flexGrow: 1,
      justifyContent: 'space-around',
    }),
    rotate180: css({
      rotate: '180deg',
    }),
    controlsWrapper: css({
      display: 'flex',
      flexDirection: 'column',
      flexGrow: 0,
      gridArea: 'controls',
      paddingRight: theme.spacing(2),
    }),
    openDataPaneButton: css({
      width: theme.spacing(8),
      justifyContent: 'center',
      svg: {
        rotate: '-90deg',
      },
    }),
    vizWrapper: css({
      height: '100%',
      width: '100%',
      paddingLeft: theme.spacing(2),
    }),
    fixedSizeViz: css({
      height: '100vh',
    }),
    fullSizeEditor: css({
      height: 'max-content',
    }),
  };
}
