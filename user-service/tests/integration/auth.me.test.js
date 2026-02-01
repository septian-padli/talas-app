/**
 * Integration Test: Auth GetMe Endpoint
 * Tests the GET /api/auth/me endpoint.
 */
const request = require('supertest');
const app = require('../../src/app');
const prisma = require('../../src/utils/prisma');
const { hashPassword } = require('../../src/utils/password');

describe('GET /api/auth/me', () => {
  // Test user credentials
  const testUser = {
    email: 'getme.test@example.com',
    password: 'SecurePassword123!',
    username: 'getmetest',
    name: 'GetMe Test User'
  };

  let loginCookies;
  let createdUser;

  afterEach(async () => {
    await prisma.refreshToken?.deleteMany?.();
    await prisma.notification?.deleteMany?.();
    await prisma.user?.deleteMany?.();
  });

  /**
   * Setup: Create a user and login before running tests
   */
  beforeEach(async () => {
    const hashedPassword = await hashPassword(testUser.password);
    createdUser = await prisma.user.create({
      data: {
        email: testUser.email,
        password: hashedPassword,
        username: testUser.username,
        name: testUser.name
      }
    });

    // Login to get cookies
    const loginResponse = await request(app)
      .post('/api/auth/login')
      .send({
        email: testUser.email,
        password: testUser.password
      });

    loginCookies = loginResponse.headers['set-cookie'];
  });

  /**
   * Test Case: Successfully get user profile with valid token
   */
  describe('Success Cases', () => {
    it('should return 200 with user data when valid token is provided', async () => {
      const response = await request(app)
        .get('/api/auth/me')
        .set('Cookie', loginCookies)
        .expect('Content-Type', /json/);

      // Assert status code
      expect(response.status).toBe(200);

      // Assert response structure
      expect(response.body).toHaveProperty('code', 200);
      expect(response.body).toHaveProperty('success', true);
      expect(response.body).toHaveProperty('data');
      expect(response.body.data).toHaveProperty('user');

      // Assert user data matches logged-in user
      const returnedUser = response.body.data.user;
      expect(returnedUser).toHaveProperty('id', createdUser.id);
      expect(returnedUser).toHaveProperty('email', testUser.email);
      expect(returnedUser).toHaveProperty('username', testUser.username);
      expect(returnedUser).toHaveProperty('name', testUser.name);

      // CRITICAL: Password should NOT be returned
      expect(returnedUser).not.toHaveProperty('password');
    });
  });

  /**
   * Test Case: Failed access without token
   */
  describe('Failure Cases - Unauthorized', () => {
    it('should return 401 when no token is provided', async () => {
      const response = await request(app)
        .get('/api/auth/me')
        .expect('Content-Type', /json/);

      expect(response.status).toBe(401);
      expect(response.body.success).toBe(false);
      expect(response.body).toHaveProperty('errors');
      expect(Array.isArray(response.body.errors)).toBe(true);
      expect(response.body.errors[0].message).toMatch(/unauthorized|token|login/i);
    });

    it('should return 401 when invalid token is provided', async () => {
      const response = await request(app)
        .get('/api/auth/me')
        .set('Cookie', 'accessToken=invalid.token.here')
        .expect('Content-Type', /json/);

      // Assert status code (401 or 403 depending on implementation)
      expect([401, 403]).toContain(response.status);
      expect(response.body.success).toBe(false);
    });
  });
});
