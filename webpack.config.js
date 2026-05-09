const path = require('path')
const outputPath = path.join(__dirname, 'dist')
const CopyWebpackPlugin = require('copy-webpack-plugin')

module.exports = [
  {
    target: 'electron-preload',
    mode: 'production',
    entry: {
      preload: './bridge/preload.js'
    },
    output: {
      path: outputPath,
      filename: '[name].js'
    },
    externals: {
      electron: 'commonjs electron',
      koffi: 'commonjs koffi'
    },
    plugins: [
      new CopyWebpackPlugin({
        patterns: [
          { from: 'node_modules/koffi/index.js', to: 'node_modules/koffi/index.js' },
          { from: 'node_modules/koffi/package.json', to: 'node_modules/koffi/package.json' },
          { from: 'node_modules/koffi/build/koffi/win32_x64/koffi.node', to: 'node_modules/koffi/build/koffi/win32_x64/koffi.node' },
        ]
      })
    ],
    module: {
      rules: [
        {
          test: /\.js$/,
          exclude: /node_modules/,
          use: {
            loader: 'babel-loader',
            options: {
              cacheDirectory: true,
              presets: [['@babel/preset-env', { targets: { electron: '22' } }]]
            }
          }
        }
      ]
    }
  },
  {
    target: 'web',
    mode: 'production',
    entry: {
      index: './src/index.js'
    },
    output: {
      path: outputPath,
      filename: '[name].js'
    },
    plugins: [
      new CopyWebpackPlugin({ patterns: [{ from: 'public', to: '.' }] })
    ],
    performance: {
      hints: false
    },
    module: {
      rules: [
        {
          test: /\.(js|jsx)$/,
          exclude: /node_modules/,
          use: {
            loader: 'babel-loader',
            options: {
              cacheDirectory: true,
              presets: [
                ['@babel/preset-env', { targets: { chrome: '108' }, modules: false }],
                ['@babel/preset-react', { runtime: 'automatic' }]]
            }
          }
        },
        {
          test: /\.(less|css)$/,
          use: [
            {
              loader: 'style-loader'
            },
            {
              loader: 'css-loader',
              options: { url: false }
            },
            {
              loader: 'less-loader'
            }
          ]
        },
        {
          test: /\.wasm$/,
          type: 'asset/resource'
        }
      ]
    }
  }
]
