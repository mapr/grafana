import { css } from '@emotion/css';
import React from 'react';
import { SpanLinks } from 'src/types/links';

import { GrafanaTheme2 } from '@grafana/data';
import { useStyles2, WithContextMenu, MenuGroup, MenuItem, Icon } from '@grafana/ui';

interface SpanLinksProps {
  links: SpanLinks;
}

const renderMenuItems = (links: SpanLinks) => {
  return (
    <>
      {links.logLinks ? (
        <MenuGroup label="Logs">
          {links.logLinks.map((link, i) => (
            <MenuItem
              key={i}
              label="Logs for this span"
              onClick={link.onClick ? link.onClick : undefined}
              url={link.href}
            />
          ))}
        </MenuGroup>
      ) : null}
      {links.metricLinks ? (
        <MenuGroup label="Metrics">
          {links.metricLinks.map((link, i) => (
            <MenuItem
              key={i}
              label="Metrics for this span"
              onClick={link.onClick ? link.onClick : undefined}
              url={link.href}
            />
          ))}
        </MenuGroup>
      ) : null}
      {links.traceLinks ? (
        <MenuGroup label="Traces">
          {links.traceLinks.map((link, i) => (
            <MenuItem
              key={i}
              label={link.title ?? 'View linked span'}
              onClick={link.onClick ? link.onClick : undefined}
              url={link.href}
            />
          ))}
        </MenuGroup>
      ) : null}
    </>
  );
};

export const SpanLinksMenu = ({ links }: SpanLinksProps) => {
  const styles = useStyles2(getStyles);

  return (
    <WithContextMenu renderMenuItems={() => renderMenuItems(links)}>
      {({ openMenu }) => (
        <button onClick={openMenu} className={styles.button}>
          <Icon name="link" className={styles.button} />
        </button>
      )}
    </WithContextMenu>
  );
};

const getStyles = (theme: GrafanaTheme2) => {
  return {
    button: css`
      background: transparent;
      border: none;
      padding: 0;
      margin: 0;
      outline: none;
      margin-right: 3px;
    `,
  };
};
