/**
 * Jest Configuration for User Service Integration Testing
 */
module.exports = {
  // Root directory for tests
  rootDir: '.',

  // Test file patterns
  testMatch: ['**/tests/**/*.test.js'],

  // Setup file to run before each test suite
  setupFilesAfterEnv: ['<rootDir>/tests/setup.js'],

  // Test environment
  testEnvironment: 'node',

  // Verbose output
  verbose: true,

  // Force exit after all tests complete
  forceExit: true,

  // Clear mocks between tests
  clearMocks: true,

  // Timeout for each test (10 seconds)
  testTimeout: 10000,

  // Transform ESM packages (uuid v13+ uses ESM)
  transformIgnorePatterns: [
    'node_modules/(?!(uuid)/)',
  ],

  // Use babel-jest for transformation
  transform: {},
};
