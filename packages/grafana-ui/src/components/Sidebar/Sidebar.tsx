import { css, cx } from '@emotion/css';
import React, { ReactNode } from 'react';

import { GrafanaTheme2 } from '@grafana/data';

import { useStyles2 } from '../../themes/ThemeContext';

import { SidebarButton } from './SidebarButton';
import { SidebarOpenPane } from './SidebarOpenPane';

export interface Props {
  children?: ReactNode;
  isDocked?: boolean;
}

export function SidebarComp({ children, isDocked }: Props) {
  const styles = useStyles2(getStyles);

  return <div className={cx(styles.container, isDocked && styles.containerDocked)}>{children}</div>;
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
      borderTopLeftRadius: theme.shape.radius.default,
      borderBottomLeftRadius: theme.shape.radius.default,
      zIndex: theme.zIndex.navbarFixed,
      boxShadow: theme.shadows.z3,
      bottom: 0,
      top: 0,
      right: 0,
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
      flexGrow: 1,
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
  };
};

export interface UseSideBarOptions {
  isPaneOpen: boolean;
}

export function useSiderbar({ isPaneOpen }: UseSideBarOptions) {
  const [isDocked, setIsDocked] = React.useState(false);

  const onDockChange = () => setIsDocked(!isDocked);

  const containerProps = {
    style: {
      paddingRight: isDocked && isPaneOpen ? '308px' : '55px',
    },
  };

  return { isDocked, onDockChange, containerProps };
}
