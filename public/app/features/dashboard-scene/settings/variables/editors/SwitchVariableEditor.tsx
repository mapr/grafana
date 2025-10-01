import { noop } from 'lodash';
import { ChangeEvent } from 'react';
import { lastValueFrom } from 'rxjs';

import { t } from '@grafana/i18n';
import { SceneVariable, SwitchVariable } from '@grafana/scenes';
import { OptionsPaneItemDescriptor } from 'app/features/dashboard/components/PanelEditor/OptionsPaneItemDescriptor';

import { SwitchVariableForm } from '../components/SwitchVariableForm';

interface SwitchVariableEditorProps {
  variable: SwitchVariable;
  onChange: (variable: SwitchVariable) => void;
}

export function SwitchVariableEditor({ variable }: SwitchVariableEditorProps) {
  const { value } = variable.useState();

  const onSwitchValueChange = async (e: ChangeEvent<HTMLInputElement>) => {
    variable.setState({ value: e.currentTarget.checked });
    if (variable.validateAndUpdate) {
      await lastValueFrom(variable.validateAndUpdate());
    }
  };

  return <SwitchVariableForm value={value} onChange={onSwitchValueChange} />;
}

export function getSwitchVariableOptions(variable: SceneVariable): OptionsPaneItemDescriptor[] {
  if (!(variable instanceof SwitchVariable)) {
    console.warn('getSwitchVariableOptions: variable is not a SwitchVariable');
    return [];
  }

  return [
    new OptionsPaneItemDescriptor({
      title: t('dashboard-scene.switch-variable-form.label-value', 'Default value'),
      id: `variable-${variable.state.name}-value`,
      render: () => <SwitchVariableEditor onChange={noop} variable={variable} />,
    }),
  ];
}
