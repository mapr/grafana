import { t } from '@grafana/i18n';
import { SceneComponentProps, sceneGraph } from '@grafana/scenes';
import { ConditionalRenderingGroupKind } from '@grafana/schema/dist/esm/schema/dashboard/v2alpha1/types.spec.gen';
import { Stack } from '@grafana/ui';

import { ConditionalRenderingBase, ConditionalRenderingBaseState } from './ConditionalRenderingBase';
import { ConditionalRenderingData } from './ConditionalRenderingData';
import { ConditionalRenderingGroupAdd } from './ConditionalRenderingGroupAdd';
import { ConditionalRenderingGroupCondition } from './ConditionalRenderingGroupCondition';
import { ConditionalRenderingGroupVisibility } from './ConditionalRenderingGroupVisibility';
import { ConditionalRenderingTimeRangeSize } from './ConditionalRenderingTimeRangeSize';
import { ConditionalRenderingVariable } from './ConditionalRenderingVariable';
import { conditionalRenderingSerializerRegistry } from './serializers';
import {
  ConditionalRenderingKindTypes,
  ConditionalRenderingSerializerRegistryItem,
  GroupConditionCondition,
  GroupConditionItemType,
  GroupConditionVisibility,
  GroupConditionValue,
  ConditionEvaluationResult,
  ConditionalRenderingConditions,
} from './types';

export interface ConditionalRenderingGroupState extends ConditionalRenderingBaseState<GroupConditionValue> {
  visibility: GroupConditionVisibility;
  condition: GroupConditionCondition;
}

export class ConditionalRenderingGroup extends ConditionalRenderingBase<ConditionalRenderingGroupState> {
  public static Component = ConditionalRenderingGroupRenderer;

  public static serializer: ConditionalRenderingSerializerRegistryItem = {
    id: 'ConditionalRenderingGroup',
    name: 'Group',
    deserialize: this.deserialize,
  };

  public get title(): string {
    return t('dashboard.conditional-rendering.conditions.group.label', 'Group');
  }

  public get info(): undefined {
    return undefined;
  }

  private _shouldShow: boolean;
  private _shouldMatchAll: boolean;

  public constructor(state: ConditionalRenderingGroupState) {
    super(state);
    this._shouldShow = state.visibility === 'show';
    this._shouldMatchAll = state.condition === 'and';
  }

  public evaluate(): ConditionEvaluationResult {
    if (this.state.value.length === 0) {
      return undefined;
    }

    return this._shouldMatchAll
      ? this.state.value.every((item) => this._evaluateItem(item))
      : this.state.value.some((item) => this._evaluateItem(item));
  }

  public changeVisibility(visibility: GroupConditionVisibility) {
    if (visibility !== this.state.visibility) {
      this._shouldShow = visibility === 'show';
      this.setStateAndRecalculate({ visibility });
    }
  }

  public changeCondition(condition: GroupConditionCondition) {
    if (condition !== this.state.condition) {
      this._shouldMatchAll = condition === 'and';
      this.setStateAndRecalculate({ condition });
    }
  }

  public addItem(itemType: GroupConditionItemType) {
    const item =
      itemType === 'data'
        ? ConditionalRenderingData.createEmpty()
        : itemType === 'variable'
          ? ConditionalRenderingVariable.createEmpty(sceneGraph.getVariables(this).state.variables[0].state.name)
          : ConditionalRenderingTimeRangeSize.createEmpty();

    // We don't use `setStateAndNotify` here because
    // We need to set a parent and activate the new condition before notifying the root
    this.setState({ value: [...this.state.value, item] });

    if (this.isActive && !item.isActive) {
      item.activate();
    }

    this.recalculateResult();
  }

  public deleteItem(key: string) {
    this.setStateAndRecalculate({ value: this.state.value.filter((condition) => condition.state.key !== key) });
  }

  public serialize(): ConditionalRenderingGroupKind {
    if (this.state.value.some((item) => item instanceof ConditionalRenderingGroup)) {
      throw new Error('ConditionalRenderingGroup cannot contain nested ConditionalRenderingGroups');
    }

    return {
      kind: 'ConditionalRenderingGroup',
      spec: {
        visibility: this.state.visibility,
        condition: this.state.condition,
        items: this.state.value
          .map((condition) => condition.serialize())
          .filter((item) => item.kind !== 'ConditionalRenderingGroup'),
      },
    };
  }

  private _evaluateItem(item: ConditionalRenderingConditions): boolean {
    const { result } = item.state;

    // When the result is undefined, we consider it to be truthy
    if (result === undefined) {
      return true;
    }

    return result === this._shouldShow;
  }

  public static deserialize(model: ConditionalRenderingGroupKind): ConditionalRenderingGroup {
    return new ConditionalRenderingGroup({
      condition: model.spec.condition,
      visibility: model.spec.visibility,
      value: model.spec.items.map((item: ConditionalRenderingKindTypes) => {
        const serializerRegistryItem = conditionalRenderingSerializerRegistry.getIfExists(item.kind);

        if (!serializerRegistryItem) {
          throw new Error(`No serializer found for conditional rendering kind: ${item.kind}`);
        }

        return serializerRegistryItem.deserialize(item);
      }),
      result: undefined,
    });
  }

  public static createEmpty(): ConditionalRenderingGroup {
    return new ConditionalRenderingGroup({ condition: 'and', visibility: 'show', value: [], result: undefined });
  }
}

function ConditionalRenderingGroupRenderer({ model }: SceneComponentProps<ConditionalRenderingGroup>) {
  const { condition, visibility, value } = model.useState();
  const { variables } = sceneGraph.getVariables(model).useState();

  return (
    <Stack direction="column" gap={2}>
      <ConditionalRenderingGroupVisibility
        itemType={model.getItemType()}
        value={visibility}
        onChange={(value) => model.changeVisibility(value)}
      />
      {value.length > 1 && (
        <ConditionalRenderingGroupCondition value={condition} onChange={(value) => model.changeCondition(value)} />
      )}
      {value.map((entry) => entry.render())}
      <ConditionalRenderingGroupAdd
        itemType={model.getItemType()}
        hasVariables={variables.length > 0}
        onAdd={(itemType) => model.addItem(itemType)}
      />
    </Stack>
  );
}
