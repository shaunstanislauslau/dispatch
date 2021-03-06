module.exports = {
  presets: [
    [
      '@babel/preset-env',
      {
        modules: false,
        loose: true
      }
    ],
    '@babel/preset-react',
    [
      '@babel/preset-stage-0',
      {
        decoratorsLegacy: true
      }
    ]
  ],
  env: {
    development: {
      plugins: ['react-hot-loader/babel']
    },
    test: {
      plugins: ['@babel/plugin-transform-modules-commonjs']
    },
    production: {
      plugins: [
        '@babel/plugin-transform-react-inline-elements',
        '@babel/plugin-transform-react-constant-elements'
      ]
    }
  }
};
