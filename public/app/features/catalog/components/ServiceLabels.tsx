import React from 'react';
import { CatalogLabels } from 'app/types/catalog';
import { Tag } from '@grafana/ui';

interface ServiceLabelProps {
  name: string;
  value: string;
}

export const ServiceLabel = (props: ServiceLabelProps) => {
  const { name, value } = props;
  return <Tag name={`${name}: ${value}`} />;
};

interface Props {
  labels?: CatalogLabels;
}

export const ServiceLabels = (props: Props) => {
  const { labels } = props;
  return (
    <div className="service-labels">
      {Object.keys(labels || {}).map((key, i) => {
        return <ServiceLabel name={key} value={labels![key]} key={i} />;
      })}
    </div>
  );
};
