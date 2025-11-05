import { css } from '@emotion/css';

import { GrafanaTheme2, IconName } from '@grafana/data';

import { useStyles2 } from '../../themes/ThemeContext';
import { ToolbarButton } from '../ToolbarButton/ToolbarButton';

export interface Props {
  icon: IconName;
  active?: boolean;
  onClick?: () => void;
  tooltip?: string;
}

export function SidebarButton({ icon, active, onClick, tooltip }: Props) {
  const styles = useStyles2(getStyles);

  return (
    <div className={styles.container}>
      <ToolbarButton icon={icon} variant={active ? 'active' : 'default'} onClick={onClick} tooltip={tooltip} />
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => {
  return {
    container: css({
      display: 'flex',
      flexDirection: 'column',
    }),
  };
};
