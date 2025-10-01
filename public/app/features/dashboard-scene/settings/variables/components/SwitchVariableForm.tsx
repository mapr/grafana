import { ChangeEvent } from 'react';

import { Trans, t } from '@grafana/i18n';
import { Field, Switch } from '@grafana/ui';

import { VariableLegend } from './VariableLegend';

interface SwitchVariableFormProps {
  value: boolean;
  onChange: (event: ChangeEvent<HTMLInputElement>) => void;
}

export function SwitchVariableForm({ value, onChange }: SwitchVariableFormProps) {
  return (
    <>
      <VariableLegend>
        <Trans i18nKey="dashboard-scene.switch-variable-form.switch-options">Switch options</Trans>
      </VariableLegend>

      <Field
        noMargin
        label={t('dashboard-scene.switch-variable.name-default-value', 'Default value')}
        description={t(
          'dashboard-scene.switch-variable-form.description-default-state',
          'The default state of the switch'
        )}
      >
        <Switch value={value} onChange={onChange} data-testid="switch-variable-default-value" />
      </Field>
    </>
  );
}
