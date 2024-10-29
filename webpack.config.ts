import type { Configuration } from 'webpack';
import { merge } from 'webpack-merge';
import CopyWebpackPlugin from 'copy-webpack-plugin';
import grafanaConfig from './.config/webpack/webpack.config';

const config = async (env): Promise<Configuration> => {
  const baseConfig = await grafanaConfig(env);
  const customConfig = {
    plugins: [
      new CopyWebpackPlugin({
        patterns: [
          {
            from: '../node_modules/@axiomhq/axiom-frontend-workers/dist',
            to: './workers',
          },
        ],
      }),
    ],
  };

  return merge(baseConfig, customConfig);
};

export default config;
