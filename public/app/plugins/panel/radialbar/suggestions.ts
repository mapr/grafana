import { FieldColorModeId, VisualizationSuggestionsBuilder } from '@grafana/data';
import { SuggestionName } from 'app/types/suggestions';

import { Options } from './panelcfg.gen';

export class GaugeSuggestionsSupplier {
  getSuggestionsForData(builder: VisualizationSuggestionsBuilder) {
    const { dataSummary } = builder;

    if (!dataSummary.hasData || !dataSummary.hasNumberField) {
      return;
    }

    // for many fields / series this is probably not a good fit
    if (dataSummary.numberFieldCount >= 10) {
      return;
    }

    const list = builder.getListAppender<Options, {}>({
      name: SuggestionName.Gauge,
      pluginId: 'gauge',
      options: {},
      fieldConfig: {
        defaults: {},
        overrides: [],
      },
      cardOptions: {
        previewModifier: (s) => {
          if (s.options!.reduceOptions.values) {
            s.options!.reduceOptions.limit = 2;
          }
        },
      },
    });

    if (dataSummary.hasStringField && dataSummary.frameCount === 1 && dataSummary.rowCountTotal < 10) {
      list.append({
        name: SuggestionName.Gauge,
        options: {
          reduceOptions: {
            values: true,
            calcs: [],
          },
        },
      });
      list.append({
        name: SuggestionName.GaugeCircular,
        options: {
          shape: 'circle',
          showThresholdMarkers: false,
          reduceOptions: {
            values: true,
            calcs: [],
          },
        },
        fieldConfig: {
          defaults: {
            color: { mode: FieldColorModeId.PaletteClassic },
          },
          overrides: [],
        },
      });
    } else {
      list.append({
        name: SuggestionName.Gauge,
        options: {
          reduceOptions: {
            values: false,
            calcs: ['lastNotNull'],
          },
        },
        fieldConfig: {
          defaults: {
            min: 0,
            max: 100,
          },
          overrides: [],
        },
      });

      list.append({
        name: 'Segmented gauge',
        isPreset: true,
        options: {
          segmentCount: 35,
          segmentSpacing: 0.4,
          reduceOptions: {
            values: false,
            calcs: ['lastNotNull'],
          },
        },
        fieldConfig: {
          defaults: {
            min: 0,
            max: 100,
          },
          overrides: [],
        },
      });

      list.append({
        name: 'Color scale',
        isPreset: true,
        options: {
          segmentCount: 35,
          segmentSpacing: 0.4,
          showThresholdMarkers: false,
          reduceOptions: {
            values: false,
            calcs: ['lastNotNull'],
          },
        },
        fieldConfig: {
          defaults: {
            min: 0,
            max: 100,
            color: { mode: FieldColorModeId.ContinuousGrYlRd },
          },
          overrides: [],
        },
      });

      list.append({
        name: 'Circular gauge',
        isPreset: true,
        options: {
          shape: 'circle',
          showThresholdMarkers: false,
          barWidthFactor: 0.15,
          effects: {
            barGlow: true,
            centerGlow: true,
            rounded: true,
            spotlight: true,
          },
          reduceOptions: {
            values: false,
            calcs: ['lastNotNull'],
          },
        },
        fieldConfig: {
          defaults: {
            min: 0,
            max: 100,
            color: { mode: FieldColorModeId.PaletteClassic },
          },
          overrides: [],
        },
      });

      list.append({
        name: 'Segmented circular gauge',
        isPreset: true,
        options: {
          shape: 'circle',
          showThresholdMarkers: false,
          barWidthFactor: 0.7,
          segmentCount: 40,
          segmentSpacing: 0.7,
          effects: {
            barGlow: true,
            centerGlow: true,
            rounded: true,
          },
          reduceOptions: {
            values: false,
            calcs: ['lastNotNull'],
          },
        },
        fieldConfig: {
          defaults: {
            min: 0,
            max: 100,
            color: { mode: FieldColorModeId.PaletteClassic },
          },
          overrides: [],
        },
      });
      list.append({
        name: SuggestionName.GaugeCircular,
        options: {
          shape: 'circle',
          showThresholdMarkers: false,
          barWidthFactor: 0.3,
          effects: {
            rounded: true,
            barGlow: true,
            centerGlow: true,
            spotlight: true,
          },
          reduceOptions: {
            values: false,
            calcs: ['lastNotNull'],
          },
        },
        fieldConfig: {
          defaults: {
            color: { mode: FieldColorModeId.PaletteClassic },
          },
          overrides: [],
        },
      });
    }
  }
}
