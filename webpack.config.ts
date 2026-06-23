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
    optimization: {
      splitChunks: {
        cacheGroups: {
          kustoMonaco: {
            test: /[\\/]node_modules[\\/](?:@kusto[\\/]monaco-kusto|@kusto[\\/]language-service|@axiomhq[\\/]language-service-next)[\\/]/,
            name: 'kusto-monaco-vendor',
            chunks: 'all',
            enforce: true,
            priority: 100,
          },
        },
      },
    },
    resolve: {
      alias: {
        'vs/language/kusto/kustoMode': '@kusto/monaco-kusto/release/esm/kustoMode',
        bridge: '@axiomhq/language-service-next/bridge',
        'kusto.javascript.client': '@kusto/language-service/Kusto.JavaScript.Client',
        'Kusto.Language.Bridge': '@axiomhq/language-service-next/Kusto.Language.Bridge',
        Kusto: '@axiomhq/language-service-next/Kusto.Language.Bridge',
      },
      fallback: {
        fs: false,
      },
    },
    module: {
      rules: [
        { test: /bridge\.js/, parser: { system: false } },
        { test: /kusto\.javascript\.client\.js/, parser: { system: false } },
        { test: /Kusto\.JavaScript\.Client\.js/, parser: { system: false } },
        { test: /Kusto\.Language\.Bridge\.js/, parser: { system: false } },
        {
          test: /kustoLanguageService/,
          parser: { system: false },
          use: { loader: path.resolve(process.cwd(), 'node_modules/@axiomhq/axiom-frontend-workers/kustoLanguageServiceLoader.js') },
        },
        {
          test: /Kusto\.Language\.Bridge/,
          use: [
            {
              loader: 'imports-loader',
              options: {
                type: 'commonjs',
                imports: ['single bridge Bridge'],
              },
            },
          ],
        },
        {
          test: /kustoMonarchLanguageDefinition/,
          loader: 'imports-loader',
          options: {
            imports: ['side-effects Kusto', 'side-effects kusto.javascript.client'],
          },
        },
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
