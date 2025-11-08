import { css, cx } from '@emotion/css';
import { Resizable } from 're-resizable';
import { useLocalStorage } from 'react-use';

import { GrafanaTheme2 } from '@grafana/data';
import { selectors } from '@grafana/e2e-selectors';
import { Trans, t } from '@grafana/i18n';
import { FlexItem } from '@grafana/plugin-ui';
import { useSceneObjectState } from '@grafana/scenes';
import {
  useStyles2,
  useSplitter,
  ToolbarButton,
  ScrollContainer,
  Text,
  Icon,
  clearButtonStyles,
  Sidebar,
} from '@grafana/ui';

import { DashboardScene } from '../scene/DashboardScene';
import { DashboardInteractions } from '../utils/interactions';

import { DashboardEditPane } from './DashboardEditPane';
import { DashboardOutline } from './DashboardOutline';
import { ElementEditPane } from './ElementEditPane';
import { useEditableElement } from './useEditableElement';

export interface Props {
  editPane: DashboardEditPane;
  dashboard: DashboardScene;
  toolbarProps: any;
  openPaneProps: any;
  isDocked: boolean;
  onDockChange: (docked: boolean) => void;
}

/**
 * Making the EditPane rendering completely standalone (not using editPane.Component) in order to pass custom react props
 */
export function DashboardEditPaneRenderer({
  editPane,
  dashboard,
  toolbarProps,
  openPaneProps,
  isDocked,
  onDockChange,
}: Props) {
  const { selection, openView } = useSceneObjectState(editPane, { shouldActivateOrKeepAlive: true });
  const { isEditing } = dashboard.useState();
  const styles = useStyles2(getStyles);
  const editableElement = useEditableElement(selection, editPane);
  const selectedObject = selection?.getFirstObject();
  const isNewElement = selection?.isNewElement() ?? false;
  const compact = false;

  return (
    <>
      {editableElement && (
        <div {...openPaneProps}>
          <ElementEditPane
            key={selectedObject?.state.key}
            editPane={editPane}
            element={editableElement}
            isNewElement={isNewElement}
          />
        </div>
      )}
      {openView === 'outline' && (
        <div {...openPaneProps}>
          <DashboardOutline editPane={editPane} />
        </div>
      )}
      <div {...toolbarProps}>
        {/* <ToolbarButton
            icon="eye"
            tooltip="View mode"
            onClick={() => dashboard.exitEditMode({ skipConfirm: false })}
          ></ToolbarButton> */}
        <Sidebar.Button
          active={isEditing}
          icon="pen"
          title="Edit"
          compact={compact}
          onClick={() => (isEditing ? dashboard.exitEditMode({ skipConfirm: false }) : dashboard.onEnterEditMode())}
        ></Sidebar.Button>
        <Sidebar.Button icon="download-alt" title="Export" compact={compact}></Sidebar.Button>

        {isEditing && (
          <>
            <Sidebar.Divider />
            {/* <Sidebar.Button icon="corner-up-left" variant="primary" title="Save" /> */}
            <Sidebar.Button icon="corner-up-left" title={'Undo'} compact={compact} />
            <Sidebar.Button icon="corner-up-right" title={'Redo'} compact={compact} />
          </>
        )}
        <Sidebar.Divider />
        <Sidebar.Button
          icon="list-ui-alt"
          onClick={() => editPane.onOpenView('outline')}
          title="Outline"
          tooltip="Content outline"
          active={openView === 'outline'}
          compact={compact}
        ></Sidebar.Button>
        {isEditing && (
          <Sidebar.Button
            icon="cog"
            onClick={() => editPane.selectObject(dashboard, dashboard.state.key!)}
            title="Options"
            compact={compact}
            active={selectedObject === dashboard ? true : false}
          ></Sidebar.Button>
        )}
        <FlexItem grow={1} />
        {(selectedObject || openView) && (
          <>
            <Sidebar.Divider />
            <Sidebar.Button
              icon="web-section-alt"
              onClick={onDockChange}
              title="Dock"
              compact={compact}
              active={isDocked ? true : false}
            ></Sidebar.Button>
          </>
        )}
      </div>
    </>
  );
}

function getStyles(theme: GrafanaTheme2) {
  return {
    wrapper: css({
      display: 'flex',
      flexDirection: 'column',
      flex: '1 1 0',
      marginTop: theme.spacing(2),
      borderLeft: `1px solid ${theme.colors.border.weak}`,
      borderTop: `1px solid ${theme.colors.border.weak}`,
      background: theme.colors.background.primary,
      borderTopLeftRadius: theme.shape.radius.default,
    }),
    overlayWrapper: css({
      right: 0,
      bottom: 0,
      top: theme.spacing(2),
      position: 'absolute !important' as 'absolute',
      background: theme.colors.background.primary,
      borderLeft: `1px solid ${theme.colors.border.weak}`,
      borderTop: `1px solid ${theme.colors.border.weak}`,
      boxShadow: theme.shadows.z3,
      zIndex: theme.zIndex.navbarFixed,
      flexGrow: 1,
    }),
    paneContent: css({
      overflow: 'hidden',
      display: 'flex',
      flexDirection: 'column',
    }),
    rotate180: css({
      rotate: '180deg',
    }),
    tabsbar: css({
      padding: theme.spacing(0, 1),
      margin: theme.spacing(0.5, 0),
    }),
    expandOptionsWrapper: css({
      display: 'flex',
      flexDirection: 'column',
      padding: theme.spacing(2, 1, 2, 0),
    }),
    splitter: css({
      '&::after': {
        background: 'transparent',
        transform: 'unset',
        width: '100%',
        height: '1px',
        top: '100%',
        left: '0',
      },
    }),
    outlineCollapseButton: css({
      display: 'flex',
      padding: theme.spacing(0.5, 2),
      gap: theme.spacing(1),
      justifyContent: 'space-between',
      alignItems: 'center',
      background: theme.colors.background.secondary,

      '&:hover': {
        background: theme.colors.action.hover,
      },
    }),
    outlineContainer: css({
      display: 'flex',
      flexDirection: 'column',
      flexGrow: 1,
      overflow: 'hidden',
    }),
  };
}
