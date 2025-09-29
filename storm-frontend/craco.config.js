module.exports = {
  webpack: {
    configure: (webpackConfig) => {
      webpackConfig.resolve.fallback = {
        ...webpackConfig.resolve.fallback,
        'grpc-web': require.resolve('@improbable-eng/grpc-web'),
      };
      return webpackConfig;
    },
  },
};