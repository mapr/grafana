import { css, cx } from '@emotion/css';
import React, { CSSProperties, useEffect } from 'react';

import { GrafanaTheme2 } from '@grafana/data';
import { selectors } from '@grafana/e2e-selectors';
import { config, useChromeHeaderHeight } from '@grafana/runtime';
import { useSceneObjectState } from '@grafana/scenes';
import { ElementSelectionContext, useSiderbar, useStyles2 } from '@grafana/ui';
import NativeScrollbar, { DivScrollElement } from 'app/core/components/NativeScrollbar';

import { useSnappingSplitter } from '../panel-edit/splitter/useSnappingSplitter';
import { DashboardScene } from '../scene/DashboardScene';
import { NavToolbarActions } from '../scene/NavToolbarActions';

import { DashboardEditPaneRenderer } from './DashboardEditPaneRenderer';
import { useEditPaneCollapsed } from './shared';

interface Props {
  dashboard: DashboardScene;
  isEditing?: boolean;
  body?: React.ReactNode;
  controls?: React.ReactNode;
}

export function DashboardEditPaneSplitter({ dashboard, isEditing, body, controls }: Props) {
  const headerHeight = useChromeHeaderHeight();
  const { editPane } = dashboard.state;
  const styles = useStyles2(getStyles, headerHeight ?? 0);
  const { openView, selection } = useSceneObjectState(editPane, { shouldActivateOrKeepAlive: true });

  if (!config.featureToggles.dashboardNewLayouts) {
    return (
      <NativeScrollbar onSetScrollRef={dashboard.onSetScrollRef}>
        <div className={styles.canvasWrappperOld}>
          <NavToolbarActions dashboard={dashboard} />
          <div className={styles.controlsWrapperSticky}>{controls}</div>
          <div className={styles.body}>{body}</div>
        </div>
      </NativeScrollbar>
    );
  }

  /**
   * Enable / disable selection based on dashboard isEditing state
   */
  useEffect(() => {
    if (isEditing) {
      editPane.enableSelection();
    } else {
      editPane.disableSelection();
    }
  }, [isEditing, editPane]);

  const { selectionContext } = useSceneObjectState(editPane, { shouldActivateOrKeepAlive: true });

  const onBodyRef = (ref: HTMLDivElement | null) => {
    if (ref) {
      dashboard.onSetScrollRef(new DivScrollElement(ref));
    }
  };

  const onClearSelection: React.PointerEventHandler<HTMLDivElement> = (evt) => {
    if (evt.shiftKey) {
      return;
    }

    editPane.clearSelection();
  };

  const { toolbarProps, sidebarProps, openPaneProps, containerProps, isDocked, onDockChange } = useSiderbar({
    position: 'right',
    compact: false,
    isPaneOpen: openView !== 'closed' || selection?.getSelection() !== undefined,
  });

  return (
    <div className={styles.container}>
      <ElementSelectionContext.Provider value={selectionContext}>
        <div className={cx(styles.controlsWrapperSticky)} onPointerDown={onClearSelection}>
          {controls}
        </div>
        <div className={styles.bodyWrapper}>
          <div
            className={styles.bodyWithToolbar}
            {...containerProps}
            data-testid={selectors.components.DashboardEditPaneSplitter.primaryBody}
            ref={onBodyRef}
            onPointerDown={onClearSelection}
          >
            {body}
          </div>
          <div {...sidebarProps}>
            <DashboardEditPaneRenderer
              editPane={editPane}
              dashboard={dashboard}
              toolbarProps={toolbarProps}
              openPaneProps={openPaneProps}
              isDocked={isDocked}
              onDockChange={onDockChange}
            />
          </div>
        </div>
      </ElementSelectionContext.Provider>
    </div>
  );
}

function getStyles(theme: GrafanaTheme2, headerHeight: number) {
  return {
    canvasWrappperOld: css({
      label: 'canvas-wrapper-old',
      display: 'flex',
      flexDirection: 'column',
      flexGrow: 1,
    }),
    canvasWithSplitter: css({
      overflow: 'unset',
      display: 'flex',
      flexDirection: 'column',
      flexGrow: 1,
    }),
    container: css({
      label: 'container',
      display: 'flex',
      flexDirection: 'column',
      flexGrow: 1,
      position: 'relative',
    }),
    primary: css({
      display: 'flex',
      flexDirection: 'column',
      flexGrow: 1,
    }),
    secondary: css({
      width: '48px',
      display: 'flex',
      height: '100%',
      flexShrink: 0,
    }),
    canvasWithSplitterEditing: css({
      overflow: 'unset',
    }),
    bodyWrapper: css({
      label: 'body-wrapper',
      display: 'flex',
      flexDirection: 'row',
      flexGrow: 1,
      position: 'relative',
    }),
    body: css({
      label: 'body',
      display: 'flex',
      flexGrow: 1,
      gap: theme.spacing(1),
      boxSizing: 'border-box',
      flexDirection: 'column',
      // without top padding the fixed controls headers is rendered over the selection outline.
      padding: theme.spacing(0.125, 2, 2, 2),
    }),
    bodyWithToolbar: css({
      position: 'absolute',
      display: 'flex',
      flexDirection: 'column',
      left: 0,
      top: 0,
      right: 0,
      bottom: 0,
      overflow: 'auto',
      scrollbarWidth: 'thin',
      scrollbarGutter: 'stable',
      // without top padding the fixed controls headers is rendered over the selection outline.
      padding: theme.spacing(0.125, 1, 2, 2),
    }),
    toolbar: css({
      display: 'flex',
      width: '48px',
      position: 'absolute',
      right: 0,
      bottom: 0,
      top: 0,
      height: '100%',
      flexDirection: 'column',
      // borderLeft: `1px solid ${theme.colors.border.weak}`,
      // background: theme.colors.background.primary,
    }),
    splitter: css({
      '&:after': {
        display: 'none',
      },
    }),
    controlsWrapperSticky: css({
      [theme.breakpoints.up('md')]: {
        position: 'sticky',
        zIndex: theme.zIndex.activePanel,
        background: theme.colors.background.canvas,
        top: headerHeight,
      },
    }),
  };
}
