import { css } from '@emotion/css';

import { GrafanaTheme2 } from '@grafana/data';
import { Stack, ToolbarButton, useStyles2 } from '@grafana/ui';

import { DashboardScene } from '../scene/DashboardScene';

export function DashboardSidebar({ dashboard }: { dashboard: DashboardScene }) {
  const styles = useStyles2(getStyles);
  const { isEditing } = dashboard.useState();

  return (
    <div className={styles.wrapper}>
      <Stack direction="column" gap={2} alignItems="center">
        <ToolbarButton
          icon="eye"
          tooltip="View mode"
          onClick={() => dashboard.exitEditMode({ skipConfirm: false })}
        ></ToolbarButton>
        <ToolbarButton icon="pen" onClick={() => dashboard.onEnterEditMode()}></ToolbarButton>
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
        <ToolbarButton icon="list-ui-alt"></ToolbarButton>
      </Stack>
    </div>
  );
}

function getStyles(theme: GrafanaTheme2) {
  return {
    wrapper: css({
      display: 'flex',
      flexDirection: 'column',
      flex: '1 1 0',
      borderLeft: `1px solid ${theme.colors.border.weak}`,
      background: theme.colors.background.primary,
      paddingTop: theme.spacing(2),
      borderTop: `1px solid ${theme.colors.border.weak}`,
      borderBottom: `1px solid ${theme.colors.border.weak}`,
      borderTopLeftRadius: theme.shape.radius.default,
      borderBottomLeftRadius: theme.shape.radius.default,
      marginBottom: theme.spacing(2),
    }),
    separator: css({
      height: '1px',
      background: theme.colors.border.weak,
      width: '100%',
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
