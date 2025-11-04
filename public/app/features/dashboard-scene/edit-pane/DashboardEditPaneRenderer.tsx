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
  Stack,
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
}

/**
 * Making the EditPane rendering completely standalone (not using editPane.Component) in order to pass custom react props
 */
export function DashboardEditPaneRenderer({ editPane, dashboard }: Props) {
  const { selection, openView, isDocked } = useSceneObjectState(editPane, { shouldActivateOrKeepAlive: true });
  const { isEditing } = dashboard.useState();
  const styles = useStyles2(getStyles);
  const editableElement = useEditableElement(selection, editPane);
  const selectedObject = selection?.getFirstObject();
  const isNewElement = selection?.isNewElement() ?? false;

  const [outlineCollapsed, setOutlineCollapsed] = useLocalStorage(
    'grafana.dashboard.edit-pane.outline.collapsed',
    true
  );

  return (
    <div className={cx(styles.wrapper, isDocked && styles.wrapperDocked)}>
      {editableElement && (
        <div className={cx(styles.sidebarView)} style={{ width: '260px' }}>
          <ElementEditPane
            key={selectedObject?.state.key}
            editPane={editPane}
            element={editableElement}
            isNewElement={isNewElement}
          />
        </div>
      )}
      {openView && (
        <div className={styles.sidebarView}>
          <DashboardOutline editPane={editPane} />
        </div>
      )}
      <div className={styles.toolbar}>
        {/* <ToolbarButton
            icon="eye"
            tooltip="View mode"
            onClick={() => dashboard.exitEditMode({ skipConfirm: false })}
          ></ToolbarButton> */}
        <ToolbarButton
          variant={isEditing ? 'active' : 'default'}
          icon="pen"
          onClick={() => (isEditing ? dashboard.exitEditMode({ skipConfirm: false }) : dashboard.onEnterEditMode())}
        ></ToolbarButton>
        <ToolbarButton variant={isEditing ? 'default' : 'primary'} icon="share-alt" tooltip="Share"></ToolbarButton>
        <ToolbarButton icon="download-alt" tooltip="Export"></ToolbarButton>

        {isEditing && (
          <>
            <div className={styles.separator} />
            <ToolbarButton icon="corner-up-left" variant="primary" icon="save" tooltip="Save" />
            <ToolbarButton icon="corner-up-left" disabled={true} onClick={() => {}} tooltip={'Undo'} />
            <ToolbarButton icon="corner-up-right" disabled={true} onClick={() => {}} tooltip={'Redo'} />
          </>
        )}
        <div className={styles.separator} />
        <ToolbarButton
          icon="list-ui-alt"
          onClick={editPane.onToggleOutline}
          tooltip="Content outline"
          variant={openView ? 'active' : 'default'}
        ></ToolbarButton>
        {isEditing && (
          <ToolbarButton
            icon="cog"
            onClick={() => editPane.selectObject(dashboard, dashboard.state.key!)}
            tooltip="Dashboard settings"
            variant={selectedObject === dashboard ? 'active' : 'default'}
          ></ToolbarButton>
        )}
        <FlexItem grow={1} />
        <div className={styles.separator} />
        <ToolbarButton
          icon="web-section-alt"
          onClick={editPane.onToggleDock}
          tooltip="Dock pane"
          variant={isDocked ? 'active' : 'default'}
        ></ToolbarButton>
      </div>
    </div>
  );
}

function getStyles(theme: GrafanaTheme2) {
  return {
    wrapper: css({
      display: 'flex',
      position: 'absolute',
      flexDirection: 'row',
      flex: '1 1 0',
      borderLeft: `1px solid ${theme.colors.border.weak}`,
      background: theme.colors.background.primary,
      borderTop: `1px solid ${theme.colors.border.weak}`,
      borderBottom: `1px solid ${theme.colors.border.weak}`,
      borderTopLeftRadius: theme.shape.radius.default,
      borderBottomLeftRadius: theme.shape.radius.default,
      marginBottom: theme.spacing(2),
      zIndex: theme.zIndex.navbarFixed,
      boxShadow: theme.shadows.z3,
      bottom: 0,
      top: 0,
      right: 0,
    }),
    wrapperDocked: css({
      boxShadow: 'none',
    }),
    sidebarView: css({
      width: '240px',
      flexGrow: 1,
      borderRight: `1px solid ${theme.colors.border.weak}`,
      paddingBottom: theme.spacing(2),
    }),
    toolbar: css({
      width: '48px',
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      padding: theme.spacing(1, 0),
      flexGrow: 1,
      gap: theme.spacing(1),
    }),
    separator: css({
      height: '1px',
      background: theme.colors.border.weak,
      width: '100%',
    }),
  };
}
