/**
 * Jest Setup File
 * Runs before each test suite to prepare the database environment.
 */
const { execSync } = require('child_process');
const prisma = require('../src/utils/prisma');

// List of all tables to truncate (order matters due to foreign keys)
const TABLES_TO_TRUNCATE = [
  'social_links',
  'notifications',
  'refresh_tokens',
  'follows',
  'users',
];

/**
 * Truncates all tables in the test database.
 * Uses raw SQL for performance.
 */
async function cleanupDatabase() {
  // Disable FK checks, truncate, re-enable
  await prisma.$executeRawUnsafe(`
    TRUNCATE TABLE ${TABLES_TO_TRUNCATE.join(', ')} RESTART IDENTITY CASCADE;
  `);
}

/**
 * Global setup before all tests run.
 * Syncs the test database schema with Prisma migrations.
 */
beforeAll(async () => {
  console.log('🔧 Setting up test database...');
  
  try {
    // Push Prisma schema to test database (creates tables if not exist)
    execSync('npx prisma db push --skip-generate', {
      env: { ...process.env },
      stdio: 'inherit',
    });
    console.log('✅ Test database schema synced.');
  } catch (error) {
    console.error('❌ Failed to sync test database:', error.message);
    throw error;
  }
});

/**
 * Cleanup after each test case to ensure isolation.
 */
afterEach(async () => {
  await cleanupDatabase();
});

/**
 * Disconnect Prisma after all tests complete.
 */
afterAll(async () => {
  await prisma.$disconnect();
  console.log('🔌 Prisma disconnected.');
});
