import { css, cx } from '@emotion/css';
import React, { ReactNode } from 'react';

import { GrafanaTheme2 } from '@grafana/data';

import { useStyles2 } from '../../themes/ThemeContext';

import { SidebarButton } from './SidebarButton';
import { SidebarOpenPane } from './SidebarOpenPane';

export interface Props {
  children?: ReactNode;
  isDocked?: boolean;
  position?: 'left' | 'right';
}

export type SidebarPosition = 'left' | 'right';

export function SidebarComp({ children, isDocked, position = 'right' }: Props) {
  const styles = useStyles2(getStyles);

  return (
    <div
      className={cx(styles.container, isDocked && styles.containerDocked, position === 'left' && styles.containerLeft)}
    >
      {children}
    </div>
  );
}

export interface SiderbarToolbarProps {
  children?: ReactNode;
  isDocked?: boolean;
  isPaneOpen?: boolean;
  onDockChange?: () => void;
}

export function SiderbarToolbar({ children, isDocked, onDockChange, isPaneOpen }: SiderbarToolbarProps) {
  const styles = useStyles2(getStyles);

  return (
    <div className={styles.toolbar}>
      {children}
      <div className={styles.flexGrow} />
      {isPaneOpen && (
        <SidebarButton
          icon={'web-section-alt'}
          onClick={onDockChange}
          tooltip={isDocked ? 'Undock sidebar' : 'Dock sidebar'}
        />
      )}
    </div>
  );
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

export interface UseSideBarOptions {
  isPaneOpen?: boolean;
  position?: SidebarPosition;
  tabsMode?: boolean;
}

export function useSiderbar({ isPaneOpen, position = 'right', tabsMode }: UseSideBarOptions) {
  const [isDocked, setIsDocked] = React.useState(false);

  const styles = useStyles2(getStyles);

  const onDockChange = () => setIsDocked(!isDocked);

  const prop = position === 'right' ? 'paddingRight' : 'paddingLeft';

  const containerProps = {
    style: {
      [prop]: isDocked && isPaneOpen ? '308px' : '55px',
    },
  };

  const sidebarProps = {
    className: cx({
      [styles.container]: true,
      [styles.containerDocked]: isDocked,
      [styles.containerLeft]: position === 'left',
      [styles.containerTabsMode]: tabsMode,
    }),
  };

  const toolbarProps = {
    className: styles.toolbar,
  };

  const openPaneProps = {
    className: cx(styles.openPane, position === 'right' ? styles.openPaneRight : styles.openPaneLeft),
  };

  const dockButton = (
    <>
      <div className={styles.flexGrow} />
      {isPaneOpen && (
        <SidebarButton
          icon={'web-section-alt'}
          onClick={onDockChange}
          tooltip={isDocked ? 'Undock sidebar' : 'Dock sidebar'}
        />
      )}
    </>
  );

  return { isDocked, onDockChange, containerProps, sidebarProps, toolbarProps, dockButton, openPaneProps };
}

export const getStyles = (theme: GrafanaTheme2) => {
  return {
    container: css({
      display: 'flex',
      position: 'absolute',
      flexDirection: 'row',
      flex: '1 1 0',
      border: `1px solid ${theme.colors.border.weak}`,
      background: theme.colors.background.primary,
      borderRight: 'none',
      borderRadius: `${theme.shape.radius.default} 0 0 ${theme.shape.radius.default}`,
      zIndex: theme.zIndex.navbarFixed,
      boxShadow: theme.shadows.z3,
      bottom: 0,
      top: 0,
      right: 0,
    }),
    containerTabsMode: css({
      position: 'relative',
      boxShadow: 'none',
    }),
    containerLeft: css({
      borderRight: `1px solid ${theme.colors.border.weak}`,
      borderLeft: 'none',
      right: 'unset',
      flexDirection: 'row-reverse',
      left: 0,
      borderRadius: `0 ${theme.shape.radius.default} ${theme.shape.radius.default} 0`,
    }),
    containerDocked: css({
      boxShadow: 'none',
    }),
    toolbar: css({
      width: '48px',
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      padding: theme.spacing(1, 0),
      flexGrow: 0,
      gap: theme.spacing(1),
    }),
    divider: css({
      height: '1px',
      background: theme.colors.border.weak,
      width: '100%',
    }),
    flexGrow: css({
      flexGrow: 1,
    }),
    openPane: css({
      width: '260px',
      flexGrow: 1,
      paddingBottom: theme.spacing(2),
    }),
    openPaneRight: css({
      borderRight: `1px solid ${theme.colors.border.weak}`,
    }),
    openPaneLeft: css({
      borderLeft: `1px solid ${theme.colors.border.weak}`,
    }),
  };
};
