/**
 * Integration Test: User Profile Me Endpoint
 * Tests the GET /api/users/me endpoint.
 */
const request = require('supertest');
const app = require('../../src/app');
const prisma = require('../../src/utils/prisma');
const { hashPassword } = require('../../src/utils/password');

describe('GET /api/users/me', () => {
  // Test user credentials
  const testUser = {
    email: 'usersme.test@example.com',
    password: 'SecurePassword123!',
    username: 'usersmetest',
    name: 'Users Me Test User',
    bio: 'Test bio for users/me endpoint'
  };

  let createdUser, loginCookies;

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
        name: testUser.name,
        bio: testUser.bio
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
   * Test Case: Successfully get own profile with valid token
   */
  describe('Success Cases', () => {
    it('should return 200 with full user profile when valid token is provided', async () => {
      const response = await request(app)
        .get('/api/users/me')
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
      expect(returnedUser).toHaveProperty('email', testUser.email); // Private field visible
      expect(returnedUser).toHaveProperty('username', testUser.username);
      expect(returnedUser).toHaveProperty('name', testUser.name);
      expect(returnedUser).toHaveProperty('bio', testUser.bio);

      // Additional profile fields (snake_case per API contract)
      expect(returnedUser).toHaveProperty('avatar_url');
      expect(returnedUser).toHaveProperty('followers_count');
      expect(returnedUser).toHaveProperty('following_count');
      expect(returnedUser).toHaveProperty('created_at');

      // CRITICAL: Password should NOT be returned
      expect(returnedUser).not.toHaveProperty('password');
    });

    it('should return consistent user ID with login session', async () => {
      // Get user from /users/me using existing login cookies
      const meResponse = await request(app)
        .get('/api/users/me')
        .set('Cookie', loginCookies);

      // User ID should match the created user
      expect(meResponse.body.data.user.id).toBe(createdUser.id);
    });
  });

  /**
   * Test Case: Failed access without token
   */
  describe('Failure Cases - Unauthorized', () => {
    it('should return 401 when no token is provided', async () => {
      const response = await request(app)
        .get('/api/users/me')
        .expect('Content-Type', /json/);

      // Assert status code
      expect(response.status).toBe(401);

      // Assert response structure
      expect(response.body).toHaveProperty('success', false);
      expect(response.body).toHaveProperty('message');

      // Message should indicate authentication required
      expect(response.body.message).toMatch(/unauthorized|token|login/i);
    });

    it('should return 401/403 when invalid token is provided', async () => {
      const response = await request(app)
        .get('/api/users/me')
        .set('Cookie', 'accessToken=invalid.token.here')
        .expect('Content-Type', /json/);

      // Assert status code (401 or 403)
      expect([401, 403]).toContain(response.status);
      expect(response.body.success).toBe(false);
    });
  });
});
