import { css } from '@emotion/css';
import { ReactNode } from 'react';

import { GrafanaTheme2 } from '@grafana/data';

import { useStyles2 } from '../../themes/ThemeContext';

import { SidebarButton } from './SidebarButton';

export interface Props {
  children?: ReactNode;
}

export function SidebarComp({ children }: Props) {
  const styles = useStyles2(getStyles);

  return <div className={styles.container}>{children}</div>;
}

export interface SiderbarToolbarProps {
  children?: ReactNode;
}

export function SiderbarToolbar({ children }: SiderbarToolbarProps) {
  const styles = useStyles2(getStyles);

  return <div className={styles.toolbar}>{children}</div>;
}

export interface SidebarOpenPaneProps {
  children?: ReactNode;
}

export function SidebarOpenPane({ children }: SidebarOpenPaneProps) {
  const styles = useStyles2(getStyles);

  return <div className={styles.openPane}>{children}</div>;
}

export function SidebarDivider() {
  const styles = useStyles2(getStyles);

  return <div className={styles.divider} />;
}

export const Sidebar = Object.assign(SidebarComp, {
  Toolbar: SiderbarToolbar,
  Button: SidebarButton,
  OpenPane: SidebarOpenPane,
  Divider: SidebarDivider,
});

const getStyles = (theme: GrafanaTheme2) => {
  return {
    container: css({
      display: 'flex',
      position: 'absolute',
      flexDirection: 'row',
      flex: '1 1 0',
      border: `1px solid ${theme.colors.border.weak}`,
      background: theme.colors.background.primary,
      borderRight: 'none',
      borderTopLeftRadius: theme.shape.radius.default,
      borderBottomLeftRadius: theme.shape.radius.default,
      zIndex: theme.zIndex.navbarFixed,
      boxShadow: theme.shadows.z3,
      bottom: 0,
      top: 0,
      right: 0,
    }),
    openPane: css({
      width: '260px',
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
    divider: css({
      height: '1px',
      background: theme.colors.border.weak,
      width: '100%',
    }),
  };
};
