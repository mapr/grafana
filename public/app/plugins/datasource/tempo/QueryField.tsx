import { css } from '@emotion/css';
import { DataSourceApi, ExploreQueryFieldProps, SelectableValue } from '@grafana/data';
import { selectors } from '@grafana/e2e-selectors';
import { config, getDataSourceSrv } from '@grafana/runtime';
import {
  FileDropzone,
  InlineField,
  InlineFieldRow,
  InlineLabel,
  LegacyForms,
  RadioButtonGroup,
  Themeable2,
  withTheme2,
} from '@grafana/ui';
import { TraceToLogsOptions } from 'app/core/components/TraceToLogsSettings';
import React from 'react';
import { LokiQueryField } from '../loki/components/LokiQueryField';
import { LokiQuery } from '../loki/types';
import { TempoDatasource, TempoQuery, TempoQueryType } from './datasource';
import NativeSearch from './NativeSearch';

interface Props extends ExploreQueryFieldProps<TempoDatasource, TempoQuery>, Themeable2 {}

const DEFAULT_QUERY_TYPE: TempoQueryType = 'traceId';
interface State {
  linkedDatasource?: DataSourceApi;
}

class TempoQueryFieldComponent extends React.PureComponent<Props, State> {
  state = {
    linkedDatasource: undefined,
  };

  constructor(props: Props) {
    super(props);
  }

  async componentDidMount() {
    const { datasource } = this.props;
    // Find query field from linked datasource
    const tracesToLogsOptions: TraceToLogsOptions = datasource.tracesToLogs || {};
    const linkedDatasourceUid = tracesToLogsOptions.datasourceUid;
    if (linkedDatasourceUid) {
      const dsSrv = getDataSourceSrv();
      const linkedDatasource = await dsSrv.get(linkedDatasourceUid);
      this.setState({
        linkedDatasource,
      });
    }
  }

  onChangeLinkedQuery = (value: LokiQuery) => {
    const { query, onChange } = this.props;
    onChange({
      ...query,
      linkedQuery: { ...value, refId: 'linked' },
    });
  };

  onRunLinkedQuery = () => {
    this.props.onRunQuery();
  };

  render() {
    const { query, onChange } = this.props;
    const { linkedDatasource } = this.state;

    const queryTypeOptions: Array<SelectableValue<TempoQueryType>> = [
      { value: 'traceId', label: 'TraceID' },
      { value: 'upload', label: 'JSON file' },
    ];

    if (config.featureToggles.tempoSearch) {
      queryTypeOptions.unshift({ value: 'nativeSearch', label: 'Search' });
    }

    if (linkedDatasource) {
      queryTypeOptions.push({ value: 'search', label: 'Loki Search' });
    }

    return (
      <>
        <InlineFieldRow>
          <InlineField label="Query type">
            <RadioButtonGroup<TempoQueryType>
              options={queryTypeOptions}
              value={query.queryType || DEFAULT_QUERY_TYPE}
              onChange={(v) =>
                onChange({
                  ...query,
                  queryType: v,
                })
              }
              size="md"
            />
          </InlineField>
        </InlineFieldRow>
        {query.queryType === 'nativeSearch' && (
          <NativeSearch
            languageProvider={this.props.datasource.languageProvider}
            query={query}
            onChange={onChange}
            onBlur={this.props.onBlur}
            onRunQuery={this.props.onRunQuery}
          />
        )}
        {query.queryType === 'search' && (
          <>
            <InlineLabel>
              Tempo uses {((linkedDatasource as unknown) as DataSourceApi).name} to find traces.
            </InlineLabel>

            <LokiQueryField
              datasource={linkedDatasource!}
              onChange={this.onChangeLinkedQuery}
              onRunQuery={this.onRunLinkedQuery}
              query={this.props.query.linkedQuery ?? ({ refId: 'linked' } as any)}
              history={[]}
            />
          </>
        )}
        {query.queryType === 'upload' && (
          <div className={css({ padding: this.props.theme.spacing(2) })}>
            <FileDropzone
              options={{ multiple: false }}
              onLoad={(result) => {
                this.props.datasource.uploadedJson = result;
                this.props.onRunQuery();
              }}
            />
          </div>
        )}
        {(!query.queryType || query.queryType === 'traceId') && (
          <LegacyForms.FormField
            label="Trace ID"
            labelWidth={4}
            inputEl={
              <div className="slate-query-field__wrapper">
                <div className="slate-query-field" aria-label={selectors.components.QueryField.container}>
                  <input
                    style={{ width: '100%' }}
                    value={query.query || ''}
                    onChange={(e) =>
                      onChange({
                        ...query,
                        query: e.currentTarget.value,
                        queryType: 'traceId',
                        linkedQuery: undefined,
                      })
                    }
                  />
                </div>
              </div>
            }
          />
        )}
      </>
    );
  }
}

export const TempoQueryField = withTheme2(TempoQueryFieldComponent);
