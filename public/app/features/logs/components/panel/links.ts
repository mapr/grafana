import { LinkModel } from "@grafana/data";

import { FieldDef } from "../logParser";

export function getTraceFromLinks(fields: FieldDef[]) {
  for (const field of fields) {
    if (!field.links) {
      continue;
    }
    for (const link of field.links) {
      const trace = getTraceFromLink(link);
      if (trace) {
        return trace;
      }
    }
  }
  return null;
}

function getTraceFromLink(link: LinkModel) {
  const queryData = getDataSourceAndQueryFromLink(link);
  if (!queryData || queryData.queryType !== 'traceql') {
    return null;
  }
  return queryData;
}

export type EmbeddedInternalLink = {
  dsUID: string;
  query: string;
  queryType: string;
};

function getDataSourceAndQueryFromLink(link: LinkModel): EmbeddedInternalLink | null {
  if (!link.href) {
    return null;
  }
  const paramsStrings = link.href.split('?')[1];
  if (!paramsStrings) {
    return null;
  }
  const params = Object.values(Object.fromEntries(new URLSearchParams(paramsStrings)));
  try {
    const parsed = JSON.parse(params[0]);
    const dsUID: string = 'datasource' in parsed && typeof parsed.datasource === 'string' ? parsed.datasource : '';
    const query: string =
      'queries' in parsed &&
      Array.isArray(parsed.queries) &&
      'query' in parsed.queries[0] &&
      typeof parsed.queries[0].query === 'string'
        ? parsed.queries[0].query
        : '';
    const queryType = 'queryType' in parsed.queries[0] &&
      typeof parsed.queries[0].queryType === 'string'
        ? parsed.queries[0].queryType
        : '';
    return dsUID && query && queryType
      ? {
          dsUID,
          query,
          queryType,
        }
      : null;
  } catch (e) {}
  return null;
}
