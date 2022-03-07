import { css } from '@emotion/css';
import {
  DataSourceJsonData,
  DataSourcePluginOptionsEditorProps,
  GrafanaTheme,
  updateDatasourcePluginJsonDataOption,
} from '@grafana/data';
import { DataSourcePicker } from '@grafana/runtime';
import { Button, InlineField, InlineFieldRow, useStyles } from '@grafana/ui';
import React from 'react';

export interface TraceToMetricsOptions {
  datasourceUid?: string;
}

export interface TraceToMetricsData extends DataSourceJsonData {
  tracesToMetrics?: TraceToMetricsOptions;
}

interface Props extends DataSourcePluginOptionsEditorProps<TraceToMetricsData> {}

export function TraceToMetricsSettings({ options, onOptionsChange }: Props) {
  const styles = useStyles(getStyles);

  return (
    <div className={css({ width: '100%' })}>
      <h3 className="page-heading">Trace to metrics</h3>

      <div className={styles.infoText}>Trace to metrics lets you link trace spans to Prometheus queries.</div>

      <InlineFieldRow className={styles.row}>
        <InlineField
          tooltip="The Prometheus data source the trace is going to query"
          label="Data source"
          labelWidth={26}
        >
          <DataSourcePicker
            inputId="trace-to-metrics-data-source-picker"
            pluginId="prometheus"
            current={options.jsonData.tracesToMetrics?.datasourceUid}
            noDefault={true}
            width={40}
            onChange={(ds) =>
              updateDatasourcePluginJsonDataOption({ onOptionsChange, options }, 'tracesToMetrics', {
                ...options.jsonData.tracesToMetrics,
                datasourceUid: ds.uid,
              })
            }
          />
        </InlineField>
        {options.jsonData.tracesToMetrics?.datasourceUid ? (
          <Button
            type={'button'}
            variant={'secondary'}
            size={'sm'}
            fill={'text'}
            onClick={() => {
              updateDatasourcePluginJsonDataOption({ onOptionsChange, options }, 'tracesToMetrics', {
                ...options.jsonData.tracesToMetrics,
                datasourceUid: undefined,
              });
            }}
          >
            Clear
          </Button>
        ) : null}
      </InlineFieldRow>
    </div>
  );
}

const getStyles = (theme: GrafanaTheme) => ({
  infoText: css`
    padding-bottom: ${theme.spacing.md};
    color: ${theme.colors.textSemiWeak};
  `,
  row: css`
    label: row;
    align-items: baseline;
  `,
});
