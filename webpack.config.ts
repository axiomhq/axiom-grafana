import type { Configuration } from 'webpack';
import { merge } from 'webpack-merge';
import CopyWebpackPlugin from 'copy-webpack-plugin';
import grafanaConfig from './.config/webpack/webpack.config';
import path from 'path';

const config = async (env): Promise<Configuration> => {
  const baseConfig = await grafanaConfig(env);

  if (baseConfig.cache && typeof baseConfig.cache === 'object' && baseConfig.cache.type === 'filesystem') {
    baseConfig.cache.version = `${baseConfig.cache.version ?? ''}|mpl-codemirror-extensionless-esm`;
    baseConfig.cache.buildDependencies = {
      ...baseConfig.cache.buildDependencies,
      config: [
        ...(baseConfig.cache.buildDependencies?.config ?? []),
        path.resolve(process.cwd(), 'webpack.config.ts'),
      ],
    };
  }

  const customConfig = {
    module: {
      rules: [
        {
          include: path.resolve(process.cwd(), 'node_modules/@axiomhq/mpl-codemirror/dist'),
          test: /\.js$/,
          resolve: {
            fullySpecified: false,
          },
        },
      ],
    },
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
