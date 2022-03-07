import React from 'react';

import { TraceSpan } from './trace';

export type SpanLinkDef = {
  href: string;
  onClick?: (event: any) => void;
  content: React.ReactNode;
};

export type SpanLinks = {
  logLinks?: SpanLinkDef[];
  traceLinks?: SpanLinkDef[];
  metricLinks?: SpanLinkDef[];
  count: number;
};

export type SpanLinkFunc = (span: TraceSpan) => SpanLinks | undefined;
