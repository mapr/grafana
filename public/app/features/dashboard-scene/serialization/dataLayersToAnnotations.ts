import { SceneDataLayerProvider, dataLayers } from '@grafana/scenes';
import { AnnotationQuery } from '@grafana/schema';

export function dataLayersToAnnotations(layers: SceneDataLayerProvider[]) {
  const annotations: AnnotationQuery[] = [];
  for (const layer of layers) {
    if (!(layer instanceof dataLayers.AnnotationsDataLayer)) {
      continue;
    }

    const result = {
      ...layer.state.query,
      enable: Boolean(layer.state.isEnabled),
      hide: Boolean(layer.state.isHidden),
      placement: layer.state.placement,
    };

    // Don't include datasource if it's empty or has no meaningful values
    if (result.datasource && Object.keys(result.datasource).length === 0) {
      delete result.datasource;
    } else if (result.datasource && !result.datasource.uid && !result.datasource.type) {
      delete result.datasource;
    }

    // Ensure builtIn is always a number (1) in v1beta1, not a boolean
    // builtIn is already in layer.state.query from transformV2ToV1AnnotationQuery
    // which sets it to 1 if truthy, but we need to handle the case where
    // the backend output has builtIn: true (boolean) instead of builtIn: 1 (number)
    if (result.builtIn !== undefined) {
      if (typeof result.builtIn === 'boolean') {
        if (result.builtIn === true) {
          result.builtIn = 1;
        } else {
          delete result.builtIn;
        }
      } else if (typeof result.builtIn === 'number') {
        if (result.builtIn === 0) {
          delete result.builtIn;
        } else if (result.builtIn !== 1) {
          result.builtIn = 1;
        }
      }
    }

    // Always preserve type: "dashboard" for built-in annotations
    // This ensures type is preserved even if it was removed during transformation
    if (result.builtIn === 1 && !result.type) {
      result.type = 'dashboard';
    }

    annotations.push(result);
  }

  return annotations;
}
