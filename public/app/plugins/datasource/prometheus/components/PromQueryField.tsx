import React, { ReactNode } from 'react';

import { css, cx } from '@emotion/css';
import { Plugin } from 'slate';
import {
  SlatePrism,
  TypeaheadInput,
  TypeaheadOutput,
  QueryField,
  BracesPlugin,
  DOMUtil,
  SuggestionsState,
  Icon,
  Modal,
  ButtonGroup,
  ToolbarButton,
  LoadingPlaceholder,
  Tooltip,
  JSONFormatter,
} from '@grafana/ui';

import { LanguageMap, languages as prismLanguages } from 'prismjs';

// dom also includes Element polyfills
import { PromQuery, PromOptions } from '../types';
import { roundMsToMin } from '../language_utils';
import { CancelablePromise, makePromiseCancelable } from 'app/core/utils/CancelablePromise';
import {
  ExploreQueryFieldProps,
  QueryHint,
  isDataFrame,
  toLegacyResponseData,
  HistoryItem,
  TimeRange,
} from '@grafana/data';
import { PrometheusDatasource } from '../datasource';
import { PrometheusMetricsBrowser } from './PrometheusMetricsBrowser';

export const RECORDING_RULES_GROUP = '__recording_rules__';

function getChooserText(metricsLookupDisabled: boolean, hasSyntax: boolean, hasMetrics: boolean) {
  if (metricsLookupDisabled) {
    return '(Disabled)';
  }

  if (!hasSyntax) {
    return 'Loading metrics...';
  }

  if (!hasMetrics) {
    return '(No metrics found)';
  }

  return 'Metrics browser';
}

export function willApplySuggestion(suggestion: string, { typeaheadContext, typeaheadText }: SuggestionsState): string {
  // Modify suggestion based on context
  switch (typeaheadContext) {
    case 'context-labels': {
      const nextChar = DOMUtil.getNextCharacter();
      if (!nextChar || nextChar === '}' || nextChar === ',') {
        suggestion += '=';
      }
      break;
    }

    case 'context-label-values': {
      // Always add quotes and remove existing ones instead
      if (!typeaheadText.match(/^(!?=~?"|")/)) {
        suggestion = `"${suggestion}`;
      }
      if (DOMUtil.getNextCharacter() !== '"') {
        suggestion = `${suggestion}"`;
      }
      break;
    }

    default:
  }
  return suggestion;
}

interface PromQueryFieldProps extends ExploreQueryFieldProps<PrometheusDatasource, PromQuery, PromOptions> {
  history: Array<HistoryItem<PromQuery>>;
  ExtraFieldElement?: ReactNode;
  placeholder?: string;
  'data-testid'?: string;
}

export interface TranslationResponseNLQ {
  logs: string;
  translation: string;
}

interface PromQueryFieldState {
  labelBrowserVisible: boolean;
  syntaxLoaded: boolean;
  hint: QueryHint | null;
  showModal: boolean;
  showNLQ: boolean;
  nlqQuery: string;
  translationNLQ: TranslationResponseNLQ;
  isNLQLoading: boolean;
}

class PromQueryField extends React.PureComponent<PromQueryFieldProps, PromQueryFieldState> {
  plugins: Plugin[];
  languageProviderInitializationPromise: CancelablePromise<any>;

  constructor(props: PromQueryFieldProps, context: React.Context<any>) {
    super(props, context);

    this.plugins = [
      BracesPlugin(),
      SlatePrism(
        {
          onlyIn: (node: any) => node.type === 'code_block',
          getSyntax: (node: any) => 'promql',
        },
        { ...(prismLanguages as LanguageMap), promql: this.props.datasource.languageProvider.syntax }
      ),
    ];

    this.state = {
      labelBrowserVisible: false,
      syntaxLoaded: false,
      hint: null,
      showModal: false,
      showNLQ: false,
      nlqQuery: '',
      translationNLQ: { translation: '', logs: '' },
      isNLQLoading: false,
    };
  }

  componentDidMount() {
    if (this.props.datasource.languageProvider) {
      this.refreshMetrics();
    }
    this.refreshHint();
  }

  componentWillUnmount() {
    if (this.languageProviderInitializationPromise) {
      this.languageProviderInitializationPromise.cancel();
    }
  }

  componentDidUpdate(prevProps: PromQueryFieldProps) {
    const {
      data,
      datasource: { languageProvider },
      range,
    } = this.props;

    if (languageProvider !== prevProps.datasource.languageProvider) {
      // We reset this only on DS change so we do not flesh loading state on every rangeChange which happens on every
      // query run if using relative range.
      this.setState({
        syntaxLoaded: false,
      });
    }

    const changedRangeToRefresh = this.rangeChangedToRefresh(range, prevProps.range);
    // We want to refresh metrics when language provider changes and/or when range changes (we round up intervals to a minute)
    if (languageProvider !== prevProps.datasource.languageProvider || changedRangeToRefresh) {
      this.refreshMetrics();
    }

    if (data && prevProps.data && prevProps.data.series !== data.series) {
      this.refreshHint();
    }
  }

  refreshHint = () => {
    const { datasource, query, data } = this.props;
    const initHints = datasource.getInitHints();
    const initHint = initHints.length > 0 ? initHints[0] : null;

    if (!data || data.series.length === 0) {
      this.setState({
        hint: initHint,
      });
      return;
    }

    const result = isDataFrame(data.series[0]) ? data.series.map(toLegacyResponseData) : data.series;
    const queryHints = datasource.getQueryHints(query, result);
    let queryHint = queryHints.length > 0 ? queryHints[0] : null;

    this.setState({ hint: queryHint ?? initHint });
  };

  refreshMetrics = async () => {
    const {
      datasource: { languageProvider },
    } = this.props;

    this.languageProviderInitializationPromise = makePromiseCancelable(languageProvider.start());

    try {
      const remainingTasks = await this.languageProviderInitializationPromise.promise;
      await Promise.all(remainingTasks);
      this.onUpdateLanguage();
    } catch (err) {
      if (!err.isCanceled) {
        throw err;
      }
    }
  };

  rangeChangedToRefresh(range?: TimeRange, prevRange?: TimeRange): boolean {
    if (range && prevRange) {
      const sameMinuteFrom = roundMsToMin(range.from.valueOf()) === roundMsToMin(prevRange.from.valueOf());
      const sameMinuteTo = roundMsToMin(range.to.valueOf()) === roundMsToMin(prevRange.to.valueOf());
      // If both are same, don't need to refresh.
      return !(sameMinuteFrom && sameMinuteTo);
    }
    return false;
  }

  /**
   * TODO #33976: Remove this, add histogram group (query = `histogram_quantile(0.95, sum(rate(${metric}[5m])) by (le))`;)
   */
  onChangeLabelBrowser = (selector: string) => {
    this.onChangeQuery(selector, true);
    this.setState({ labelBrowserVisible: false });
  };
  onTranslateNLQ = async (query: PromQuery, value: string) => {
    if (!value) {
      this.onClearQuery();
      this.onClearNLQQuery();
      return;
    }

    // call api
    try {
      this.setState({ isNLQLoading: true });
      const result = await this.props.datasource.getQueryFromNLQ(value);
      const { translation } = result;
      if (!result?.translation.includes('Could not translate')) {
        const nextQuery: PromQuery = { ...query, expr: translation };
        this.props.onChange(nextQuery);
      } else {
        this.onClearQuery();
      }
      this.setState({ isNLQLoading: false, translationNLQ: { translation: result.translation, logs: result.logs } });
    } catch (err) {
      this.setState({ isNLQLoading: false });
    }
  };
  onChangeQuery = async (value: string, override?: boolean) => {
    // Send text change to parent
    const { query, onChange, onRunQuery } = this.props;
    if (onChange) {
      const nextQuery: PromQuery = { ...query, expr: value };
      // call the endpoint
      if (this.state.showNLQ) {
        await this.onTranslateNLQ(query, value);
      } else {
        onChange(nextQuery);
      }

      if (override && onRunQuery) {
        onRunQuery();
      }
    }
  };

  onClickChooserButton = () => {
    this.setState((state) => ({ labelBrowserVisible: !state.labelBrowserVisible }));
  };

  onClickHintFix = () => {
    const { datasource, query, onChange, onRunQuery } = this.props;
    const { hint } = this.state;

    onChange(datasource.modifyQuery(query, hint!.fix!.action));
    onRunQuery();
  };

  onUpdateLanguage = () => {
    const {
      datasource: { languageProvider },
    } = this.props;
    const { metrics } = languageProvider;

    if (!metrics) {
      return;
    }

    this.setState({ syntaxLoaded: true });
  };

  onShowNaturalLanguage = () => {
    this.setState({ showModal: true });
  };
  onHideNaturalLanguage = () => {
    this.setState({ showNLQ: false, showModal: false });
  };

  onSwitchToNLQ = () => {
    // clear current prometheus query
    this.onClearQuery();
    this.setState({ showNLQ: true, showModal: false });
  };

  onClearQuery = () => {
    const nextQuery: PromQuery = { ...this.props.query, expr: '' };
    this.props.onChange(nextQuery);
  };

  onClearNLQQuery = () => {
    this.setState({ translationNLQ: { translation: '', logs: '' } });
  };
  onSwitchToPromQL = () => {
    this.setState({ showNLQ: false });
    this.onClearNLQQuery();
  };

  onTypeahead = async (typeahead: TypeaheadInput): Promise<TypeaheadOutput> => {
    const {
      datasource: { languageProvider },
    } = this.props;

    if (!languageProvider) {
      return { suggestions: [] };
    }

    const { history } = this.props;
    const { prefix, text, value, wrapperClasses, labelKey } = typeahead;

    const result = await languageProvider.provideCompletionItems(
      { text, value, prefix, wrapperClasses, labelKey },
      { history }
    );

    return result;
  };

  render() {
    const {
      datasource,
      datasource: { languageProvider },
      query,
      ExtraFieldElement,
      placeholder = 'Enter a PromQL query (run with Shift+Enter)',
    } = this.props;

    const { nlqQuery, isNLQLoading, translationNLQ } = this.state;

    const { labelBrowserVisible, syntaxLoaded, hint } = this.state;
    const cleanText = languageProvider ? languageProvider.cleanText : undefined;
    const hasMetrics = languageProvider.metrics.length > 0;
    const chooserText = getChooserText(datasource.lookupsDisabled, syntaxLoaded, hasMetrics);
    const buttonDisabled = !(syntaxLoaded && hasMetrics);
    const styles = getNLQStyles();
    return (
      <>
        <div
          className="gf-form-inline gf-form-inline--xs-view-flex-column flex-grow-1"
          data-testid={this.props['data-testid']}
        >
          {!this.state.showNLQ && (
            <button
              className="gf-form-label query-keyword pointer"
              onClick={this.onClickChooserButton}
              disabled={buttonDisabled}
            >
              {chooserText}
              <Icon name={labelBrowserVisible ? 'angle-down' : 'angle-right'} />
            </button>
          )}
          <div className="gf-form gf-form--grow flex-shrink-1 min-width-15">
            {!this.state.showNLQ && (
              <QueryField
                additionalPlugins={this.plugins}
                cleanText={cleanText}
                query={query.expr}
                onTypeahead={this.onTypeahead}
                onWillApplySuggestion={willApplySuggestion}
                onBlur={this.props.onBlur}
                onChange={this.onChangeQuery}
                onRunQuery={this.props.onRunQuery}
                placeholder={placeholder}
                portalOrigin="prometheus"
                syntaxLoaded={syntaxLoaded}
              />
            )}
            {this.state.showNLQ && (
              <div className="gf-form gf-form--grow gf-form--alt flex-shrink-1 min-width-15">
                <QueryField
                  cleanText={cleanText}
                  query={nlqQuery}
                  onBlur={this.props.onBlur}
                  onChange={this.onChangeQuery}
                  onRunQuery={this.props.onRunQuery}
                  placeholder="Ask me anything, i.e how many users are there for Grafana? (run Shift + Enter)"
                  portalOrigin="prometheus"
                  syntaxLoaded={syntaxLoaded}
                />

                <div className={styles.translatedQueryContainer}>
                  PromQL result:
                  <span className={styles.translatedQuery}>
                    {isNLQLoading ? (
                      <LoadingPlaceholder className={styles.loadingPlaceholder} text="We are doing magic..." />
                    ) : !translationNLQ?.translation ? (
                      <span className={styles.noResults}> Your PromQL query will appear here </span>
                    ) : (
                      translationNLQ.translation
                    )}
                  </span>
                  {translationNLQ?.logs && (
                    <Tooltip theme="info-alt" content={<JSONFormatter json={translationNLQ.logs} />}>
                      <ToolbarButton>&#128526; Logs for nerds </ToolbarButton>
                    </Tooltip>
                  )}
                </div>
              </div>
            )}
          </div>
          {!this.state.showNLQ && (
            <ToolbarButton
              className={cx(styles.switchButton, 'gf-form-label query-keyword')}
              onClick={this.onShowNaturalLanguage}
              disabled={buttonDisabled}
              variant="active"
            >
              &#x1F914; New to PromQl?
            </ToolbarButton>
          )}
          {this.state.showNLQ && (
            <ToolbarButton
              className={cx(styles.switchButton, 'gf-form-label query-keyword')}
              narrow
              onClick={this.onSwitchToPromQL}
            >
              Switch back to PromQL
            </ToolbarButton>
          )}
        </div>
        {labelBrowserVisible && (
          <div className="gf-form">
            <PrometheusMetricsBrowser languageProvider={languageProvider} onChange={this.onChangeLabelBrowser} />
          </div>
        )}

        {ExtraFieldElement}
        {hint ? (
          <div className="query-row-break">
            <div className="prom-query-field-info text-warning">
              {hint.label}{' '}
              {hint.fix ? (
                <a className="text-link muted" onClick={this.onClickHintFix}>
                  {hint.fix.label}
                </a>
              ) : null}
            </div>
          </div>
        ) : null}
        <Modal
          isOpen={this.state.showModal}
          closeOnEscape
          icon="trash-alt"
          title="New to PromQL?"
          onDismiss={this.onHideNaturalLanguage}
          contentClassName={styles.modalContent}
        >
          <div>
            <div>
              <p>If you are new to PromQL, would you like to try using a </p>
              <p> natural language query (NLQ)?</p>
            </div>
            <ButtonGroup key="nlq-buttons">
              <ToolbarButton variant="primary" tooltip="try new NLQ" onClick={this.onSwitchToNLQ}>
                Yes, switch to NLQ
              </ToolbarButton>
              <ToolbarButton onClick={this.onHideNaturalLanguage}>Maybe later</ToolbarButton>
            </ButtonGroup>
          </div>
        </Modal>
      </>
    );
  }
}

const getNLQStyles = () => ({
  translatedQueryContainer: css`
    display: flex;
    align-items: center;
    margin-top: 10px;
    margin-bottom: 10px;
  `,
  translatedQuery: css`
    border: 2px dashed silver;
    padding: 5px;
    margin-left: 5px;
    margin-right: 25px;
    font-weight: 400;
  `,
  noResults: css`
    color: gray;
  `,
  loadingPlaceholder: css`
    margin-bottom: 0;
    color: #ff780a;
  `,
  modalContent: css`
    padding: calc(32px);
    overflow: auto;
    width: 100%;
    max-height: calc(90vh - 32px);
  `,
  switchButton: css`
    margin-left: 5px;
  `,
});

export default PromQueryField;
