import { GrafanaRuleGroupIdentifier } from 'app/types/unified-alerting';
import { GrafanaPromRuleDTO, PromRuleType } from 'app/types/unified-alerting-dto';

import { alertRuleApi } from '../api/alertRuleApi';
import { GrafanaRulesSource } from '../utils/datasource';

import { AlertRuleListItem, RecordingRuleListItem, UnknownRuleListItem } from './components/AlertRuleListItem';
import { AlertRuleListItemLoader } from './components/AlertRuleListItemLoader';
import { RuleActionsButtons } from './components/RuleActionsButtons.V2';

const { useGetGrafanaRulerGroupQuery } = alertRuleApi;

interface GrafanaRuleLoaderProps {
  rule: GrafanaPromRuleDTO;
  groupIdentifier: GrafanaRuleGroupIdentifier;
  // TODO: How to improve this?
  namespaceName: string;
}

export function GrafanaRuleLoader({ rule, groupIdentifier, namespaceName }: GrafanaRuleLoaderProps) {
  const { data: rulerRuleGroup } = useGetGrafanaRulerGroupQuery(groupIdentifier);

  const rulerRule = rulerRuleGroup?.rules.find((rulerRule) => rulerRule.grafana_alert.uid === rule.uid);

  if (!rulerRule) {
    return <AlertRuleListItemLoader />;
  }

  const {
    grafana_alert: { title, provenance },
    annotations = {},
    labels = {},
  } = rulerRule;

  const isProvisioned = Boolean(provenance);

  switch (rule.type) {
    case PromRuleType.Alerting:
      return (
        <AlertRuleListItem
          name={title}
          rulesSource={GrafanaRulesSource}
          application="grafana"
          group={groupIdentifier.groupName}
          namespace={namespaceName}
          href={''}
          summary={annotations.summary}
          state={rule.state}
          health={rule.health}
          error={rule.lastError}
          labels={labels}
          isProvisioned={isProvisioned}
          instancesCount={rule.alerts?.length}
          actions={<RuleActionsButtons rule={rulerRule} promRule={rule} groupIdentifier={groupIdentifier} compact />}
        />
      );
    case PromRuleType.Recording:
      return (
        <RecordingRuleListItem
          name={rule.name}
          rulesSource={GrafanaRulesSource}
          application="grafana"
          group={groupIdentifier.groupName}
          namespace={namespaceName}
          href={''}
          health={rule.health}
          error={rule.lastError}
          labels={rule.labels}
          isProvisioned={isProvisioned}
          actions={<RuleActionsButtons rule={rulerRule} promRule={rule} groupIdentifier={groupIdentifier} compact />}
        />
      );
    default:
      return <UnknownRuleListItem rule={rule} groupIdentifier={groupIdentifier} />;
  }
}
