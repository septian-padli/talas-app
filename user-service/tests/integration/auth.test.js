/**
 * Integration Test: Auth Endpoints
 * Tests the /api/auth/init-testing endpoint.
 */
const request = require('supertest');
const app = require('../../src/app');

afterEach(async () => {
  await prisma.refreshToken?.deleteMany?.();
  await prisma.notification?.deleteMany?.();
  await prisma.user?.deleteMany?.();
});

describe('GET /api/auth/init-testing', () => {
  it('should return 200 with correct JSON structure', async () => {
    const response = await request(app).get('/api/auth/init-testing');

    // Assert status code
    expect(response.status).toBe(200);

    // Assert JSON structure
    expect(response.body).toHaveProperty('status', 'success');
    expect(response.body).toHaveProperty('service', 'User Service');
    expect(response.body).toHaveProperty('timestamp');

    // Validate timestamp is a valid ISO date string
    const timestamp = new Date(response.body.timestamp);
    expect(timestamp.toISOString()).toBe(response.body.timestamp);
  });

  it('should include a message in the response', async () => {
    const response = await request(app).get('/api/auth/init-testing');

    expect(response.body).toHaveProperty('message');
    expect(typeof response.body.message).toBe('string');
    expect(response.body.message.length).toBeGreaterThan(0);
  });
});
