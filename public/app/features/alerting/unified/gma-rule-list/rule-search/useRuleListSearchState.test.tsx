import { act, getWrapper, renderHook } from 'test/test-utils';

import { locationService } from '@grafana/runtime';

import { type RuleSearchSchema } from './filterDefinitions';
import { sharedFilters } from './sharedFilters';
import { type SharedRuleFilter, createEmptyFilter } from './sharedRuleFilter';
import { useRuleListSearchState } from './useRuleListSearchState';

// A page with only some of the shared filters, so the hook is tested without either page's own.
const schema: RuleSearchSchema<SharedRuleFilter> = {
  filters: [sharedFilters.folder, sharedFilters.label],
  createEmptyFilter,
};

function renderAt(search: string) {
  return renderHook(() => useRuleListSearchState(schema), {
    wrapper: getWrapper({ renderWithRouter: true, historyOptions: { initialEntries: [`/rules${search}`] } }),
  });
}

describe('useRuleListSearchState', () => {
  it("reads the search parameter with the page's filters, and returns their help", () => {
    const { result } = renderAt(`?${new URLSearchParams({ search: 'plugins:hide label:  folder:f1' })}`);

    expect(result.current.query).toBe('plugins:hide label:  folder:f1');
    expect(result.current.filter).toEqual({ ...createEmptyFilter(), folders: ['f1'], text: ['plugins:hide'] });
    expect(result.current.parseIssues).toEqual([{ key: 'label', reason: 'invalid' }]);
    expect(result.current.filters).toBe(schema.filters);
  });

  it('writes an applied query as typed and keeps the other parameters', () => {
    const { result } = renderAt('?view=folders');

    act(() => result.current.applyQuery('"high load"   folder:f1'));

    expect(locationService.getSearchObject()).toEqual({ view: 'folders', search: '"high load"   folder:f1' });
    expect(result.current.query).toBe('"high load"   folder:f1');
  });

  it('adds no history entry when the applied query is applied again', () => {
    const { result } = renderAt('?search=cpu');
    const entries = locationService.getHistory().length;

    act(() => result.current.applyQuery('cpu'));

    expect(locationService.getHistory().length).toBe(entries);
  });

  it('writes an empty search parameter for an empty query, rather than removing it', () => {
    const { result } = renderAt('?search=cpu');

    act(() => result.current.applyQuery(''));

    expect(locationService.getLocation().search).toBe('?search=');
  });
});
