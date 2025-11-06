import { css } from '@emotion/css';
import { Meta, StoryFn } from '@storybook/react';
import { useState } from 'react';

import { Box } from '../Layout/Box/Box';

import { Sidebar, useSiderbar } from './Sidebar';
import mdx from './Sidebar.mdx';

const meta: Meta<typeof Sidebar> = {
  title: 'Overlays/Sidebar',
  component: Sidebar,
  parameters: {
    docs: {
      page: mdx,
    },
    controls: {},
  },
  args: {},
  argTypes: {},
};

export const Example: StoryFn<typeof Sidebar> = (args) => {
  const [openPane, setOpenPane] = useState('');

  const containerStyle = css({
    width: '100%',
    flexGrow: 1,
    height: '600px',
    display: 'flex',
    flexDirection: 'column',
    position: 'relative',
    overflow: 'hidden',
  });

  const gridStyle = css({
    display: 'grid',
    gridTemplateColumns: 'repeat(2, 1fr)',
    gridAutoRows: '300px',
    gap: '8px',
    padding: '0 8px',
    flexGrow: 1,
    overflow: 'auto',
  });

  const renderBox = (label: string) => {
    return (
      <Box
        backgroundColor={'primary'}
        borderColor={'weak'}
        borderStyle={'solid'}
        justifyContent={'center'}
        alignItems={'center'}
        display={'flex'}
      >
        {label}
      </Box>
    );
  };

  const togglePane = (pane: string) => {
    if (openPane === pane) {
      setOpenPane('');
    } else {
      setOpenPane(pane);
    }
  };

  const { isDocked, onDockChange, containerProps } = useSiderbar({ isPaneOpen: !!openPane });

  return (
    <Box paddingY={2} backgroundColor={'canvas'} maxWidth={100} borderStyle={'solid'} borderColor={'weak'}>
      <div className={containerStyle} {...containerProps}>
        <div className={gridStyle}>
          {renderBox('A')}
          {renderBox('B')}
          {renderBox('C')}
          {renderBox('D')}
          {renderBox('E')}
          {renderBox('F')}
          {renderBox('G')}
        </div>
        <Sidebar isDocked={isDocked}>
          {openPane === 'settings' && <Sidebar.OpenPane>Settings</Sidebar.OpenPane>}
          {openPane === 'outline' && <Sidebar.OpenPane>Outline</Sidebar.OpenPane>}
          <Sidebar.Toolbar isDocked={isDocked} onDockChange={onDockChange} isPaneOpen={!!openPane}>
            <Sidebar.Button icon="share-alt" tooltip="Share" />
            <Sidebar.Button icon="info-circle" tooltip="Insights" />
            <Sidebar.Divider />
            <Sidebar.Button icon="cog" active={openPane === 'settings'} onClick={() => togglePane('settings')} />
            <Sidebar.Button icon="list-ui-alt" active={openPane === 'outline'} onClick={() => togglePane('outline')} />
          </Sidebar.Toolbar>
        </Sidebar>
      </div>
    </Box>
  );
};

export default meta;
