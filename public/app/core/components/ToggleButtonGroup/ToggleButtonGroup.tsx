import React, { PureComponent, ReactElement } from 'react';

interface ToggleButtonGroupProps {
  onChange: (value) => void;
  value?: any;
  label?: string;
}

export default class ToggleButtonGroup extends PureComponent<ToggleButtonGroupProps> {
  getValues() {
    const { children } = this.props;
    return React.Children.toArray(children).map((c: ReactElement<any>) => c.props.value);
  }

  smallChildren() {
    const { children } = this.props;
    return React.Children.toArray(children).every((c: ReactElement<any>) => c.props.classNames.includes('small'));
  }

  handleToggle(toggleValue) {
    const { value, onChange } = this.props;
    if (value && value === toggleValue) {
      return;
    }
    onChange(toggleValue);
  }

  render() {
    const { children, value, label } = this.props;
    const values = this.getValues();
    const selectedValue = value || values[0];
    const labelClassName = `gf-form-label ${this.smallChildren() ? 'small' : ''}`;

    const childClones = React.Children.map(children, (child: ReactElement<any>) => {
      const { value: buttonValue } = child.props;

      return React.cloneElement(child, {
        selected: buttonValue === selectedValue,
        onChange: this.handleToggle.bind(this),
      });
    });

    return (
      <div className="gf-form">
        <div className="toggle-button-group">
          {label && <label className={labelClassName}>{label}</label>}
          {childClones}
        </div>
      </div>
    );
  }
}
