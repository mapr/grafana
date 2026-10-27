import { useCallback, useMemo } from 'react';

import { useURLSearchParams } from '../../hooks/useURLSearchParams';

import { type FilterHelp, type RuleSearchSchema } from './filterDefinitions';
import { type FilterIssue, parse } from './parser';
import { type SharedRuleFilter } from './sharedRuleFilter';

export interface UseRuleListSearchStateResult<Filter extends SharedRuleFilter> {
  /** The applied query, as the user typed it. */
  query: string;
  filter: Filter;
  parseIssues: FilterIssue[];
  filters: FilterHelp[];
  applyQuery: (query: string) => void;
}

/**
 * The applied query lives in the `search` URL parameter as typed, so reloads, links and Back/Forward restore it.
 * `schema` should be a module-level constant: a new one on every render parses the query again.
 */
export function useRuleListSearchState<Filter extends SharedRuleFilter>(
  schema: RuleSearchSchema<Filter>
): UseRuleListSearchStateResult<Filter> {
  const [queryParams, updateQueryParams] = useURLSearchParams();

  const query = queryParams.get('search') ?? '';

  const { filter, issues: parseIssues } = useMemo(() => parse(query, schema), [query, schema]);

  const applyQuery = useCallback(
    (next: string) => {
      // Writing the same query again would add a history entry for nothing.
      if (next !== query) {
        updateQueryParams({ search: next });
      }
    },
    [updateQueryParams, query]
  );

  return { query, filter, parseIssues, filters: schema.filters, applyQuery };
}
