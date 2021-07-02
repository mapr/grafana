import React, { useEffect, useState, useMemo } from 'react';
import { hot } from 'react-hot-loader';
import { connect } from 'react-redux';
import { Themeable2, withTheme2, ToolbarButton, PageToolbar, CascaderOption } from '@grafana/ui';
import Page from 'app/core/components/Page/Page';
import { StoreState } from 'app/types';
import { loadCatalog } from '../state/actions';
import { Catalog } from 'app/types/catalog';
import { ServiceRow, ServiceHeader } from '../components/ServiceRow';
import { Table } from '../components/CatalogTable';
import { ServiceSearchBar } from '../components/ServiceSearchBar';

interface Props extends Themeable2 {
  name: string;
  catalog: Catalog;
  loadCatalog: typeof loadCatalog;
  view: 'table' | 'graph';
  setView: (view: string) => void;
}

export const UnthemedCatalogPage = (props: Props) => {
  const { catalog, view, setView, loadCatalog } = props;
  useEffect(() => {
    loadCatalog();
  }, [loadCatalog]);

  const [label, setLabel] = useState<string>('');
  const [labels, setLabels] = useState<{ [key: string]: string[] }>({});

  useEffect(() => {
    const l: { [key: string]: string[] } = {};
    catalog.map((cs) =>
      cs.components.map((cmp) => {
        return Object.keys(cmp.labels || {}).map((key) => {
          if (!l[key]) {
            l[key] = [];
          }
          l[key].push(...cmp.labels![key].split(','));
        });
      })
    );

    setLabels(l);
  }, [catalog]);

  const labelKeys: CascaderOption[] = Object.keys(labels).map((key) => {
    return {
      label: key,
      value: key,
    };
  });

  const labelValues = useMemo(() => {
    const values =
      labels[label]?.map((v) => {
        return {
          value: v,
          label: v,
        };
      }) || [];

    const seen: { [key: string]: boolean } = {};

    return values.filter((val) => {
      if (seen[val.value]) {
        console.log('seen', val, 'skipping');
        return false;
      }
      console.log('new value', val);

      seen[val.value] = true;
      return true;
    });
  }, [labels, label]);

  return (
    <Page>
      <PageToolbar pageIcon={'apps'} title={'catalog'}>
        {view === 'graph' ? (
          <ToolbarButton tooltip="Table view" icon="table" onClick={() => setView('table')} />
        ) : (
          <ToolbarButton tooltip="Graph view" icon="gf-interpolation-linear" onClick={() => setView('graph')} />
        )}
      </PageToolbar>
      <Page.Contents>
        <ServiceSearchBar labelValues={labelValues} options={labelKeys} onSelect={(val) => setLabel(val)} />
        <Table hover>
          <ServiceHeader />
          {props.catalog.map((svc) => {
            return (
              <>
                {svc.components.map((cmp, i) => {
                  return (
                    <ServiceRow
                      collapsible={cmp.pods && cmp.pods.length > 0}
                      service={svc.name}
                      component={cmp}
                      key={i}
                    />
                  );
                })}
              </>
            );
          })}
        </Table>
      </Page.Contents>
    </Page>
  );
};

// Global state stuff, redux
const mapDispatchToProps = { loadCatalog };

export const mapStateToProps = (state: StoreState) => ({
  catalog: state.catalog.catalog,
});

// Theming
export const CatalogPage = withTheme2(UnthemedCatalogPage);

// Hot module reloading ?
export default hot(module)(connect(mapStateToProps, mapDispatchToProps)(CatalogPage));
