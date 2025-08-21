import { css } from '@emotion/css';
import { useEffect, useMemo, useState } from 'react';
import { isObservable, lastValueFrom } from 'rxjs';

import { DataFrame, DataSourceApi, GrafanaTheme2, TimeRange } from '@grafana/data';
import { t } from '@grafana/i18n';
import { getDataSourceSrv } from '@grafana/runtime';
import { Icon, Spinner, Tooltip, useStyles2 } from '@grafana/ui';
import { TraceView } from 'app/features/explore/TraceView/TraceView';
import { transformDataFrames } from 'app/features/explore/TraceView/utils/transform';
import { SearchTableType } from 'app/plugins/datasource/tempo/dataquery.gen';

import { useLogListContext } from './LogListContext';
import { EmbeddedInternalLink } from './links';

interface Props {
  traceRef: EmbeddedInternalLink;
  timeRange: TimeRange;
  timeZone: string;
}

export const LogLineDetailsTrace = ({ timeRange, timeZone, traceRef }: Props) => {
  const [dataSource, setDataSource] = useState<DataSourceApi | null>(null);
  const [dataFrames, setDataFrames] = useState<DataFrame[] | null | undefined>(undefined);
  const { app } = useLogListContext();
  const styles = useStyles2(getStyles);

  useEffect(() => {
    getDataSourceSrv()
      .get(traceRef.dsUID)
      .then((dataSource) => {
        if (dataSource) {
          setDataSource(dataSource);
        } else {
          setDataFrames(null);
        }
      });
  }, [traceRef.dsUID]);

  useEffect(() => {
    if (!dataSource) {
      return;
    }
    const query = dataSource.query({
      app,
      requestId: 'test',
      targets: [
        {
          // @ts-expect-error
          query: traceRef.query,
          queryType: 'traceql',
          refId: 'log-line-details-trace',
          tableType: SearchTableType.Traces,
          filters: [],
        },
      ],
      interval: '',
      intervalMs: 0,
      range: timeRange,
      scopedVars: {},
      timezone: timeZone,
      startTime: Date.now(),
    });
    if (isObservable(query)) {
      lastValueFrom(query)
        .then((response) => {
          setDataFrames(response.data?.length ? response.data : null);
        })
        .catch(() => {
          setDataFrames(null);
        });
    }
  }, [app, dataSource, timeRange, timeZone, traceRef.query]);

  const traceProp = useMemo(() => (dataFrames?.length ? transformDataFrames(dataFrames[0]) : undefined), [dataFrames]);

  return (
    <div>
      {dataSource && Array.isArray(dataFrames) && traceProp && (
        <TraceView dataFrames={dataFrames} traceProp={traceProp} datasource={dataSource} timeRange={timeRange} />
      )}
      {dataFrames === null && (
        <div className={styles.message}>
          <Tooltip
            content={t(
              'logs.log-line-details.trace.error-tooltip',
              'The trace could have been sampled or be temporarily unavailable.'
            )}
          >
            <Icon name="info-circle" />
          </Tooltip>
          {t('logs.log-line-details.trace.error-message', 'Could not retrieve trace.')}
        </div>
      )}
      {dataFrames === undefined && (
        <div className={styles.message}>
          <Spinner />
          {t('logs.log-line-details.trace.loading-message', 'Loading trace...')}
        </div>
      )}
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
  message: css({
    display: 'flex',
    gap: theme.spacing(1),
    alignItems: 'center',
  }),
});
