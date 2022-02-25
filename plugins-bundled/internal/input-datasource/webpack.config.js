const { merge } = require('lodash');
const PnpWebpackPlugin = require('pnp-webpack-plugin');
const SpeedMeasurePlugin = require('speed-measure-webpack5-plugin');
const config = environment.toWebpackConfig();

const smp = new SpeedMeasurePlugin();

module.exports = smp.wrap({
  getWebpackConfig: (baseConfig) => {
    return merge(baseConfig, {
      resolve: {
        plugins: [PnpWebpackPlugin],
      },
      resolveLoader: {
        plugins: [PnpWebpackPlugin.moduleLoader(module)],
      },
    });
  },
});
