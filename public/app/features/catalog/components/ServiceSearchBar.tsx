import React from 'react';
import { Cascader, Segment } from '@grafana/ui';
import { SelectableValue } from '@grafana/data';
import { CascaderProps } from '@grafana/ui/src/components/Cascader/Cascader';

interface Props extends CascaderProps {
  labelValues: Array<SelectableValue<string>>;
}

export const ServiceSearchBar = (props: Props) => {
  const { labelValues } = props;
  return (
    <div className="page-action-bar">
      <div className="gf-form">
        <div className="gf-form-inline gf-form-inline--nowrap">
          <Cascader {...props} width={32} />
          <Segment options={labelValues} placeholder="Select a value..." onChange={() => {}} />
        </div>
      </div>
    </div>
  );
};
