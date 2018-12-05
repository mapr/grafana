import React, { PureComponent, ReactElement } from 'react';
import _ from 'lodash';

interface ToggleButtonGroupProps {
  onChange: (value) => void;
  value?: any;
  label?: string;
}

export default class ToggleButtonGroup extends PureComponent<ToggleButtonGroupProps> {
  getValues() {
    const { children } = this.props;
    return React.Children.toArray(children).map(c => c['props'].value);
  }

  handleToggle(toggleValue) {
    const { value, onChange } = this.props;
    if (value !== toggleValue) {
      onChange(toggleValue);
    }
  }

  render() {
    const { children, value, label, ...props } = this.props;
    const values = this.getValues();
    const selectedValue = value || values[0];
    const formId = `${_.uniqueId()}-gf-form-toggle-button-group`;
    delete props.onChange;

    const childrenClones = React.Children.map(children, child => {
      const { value: buttonValue } = child['props'];

      return React.cloneElement(child as ReactElement<any>, {
        selected: buttonValue === selectedValue,
        onChange: this.handleToggle.bind(this),
      });
    });

    return (
      <div className="gf-form">
        <div id={formId} className="toggle-button-group">
          {label && <label className="gf-form-label">{label}</label>}
          {childrenClones}
        </div>
      </div>
    );
  }
}
