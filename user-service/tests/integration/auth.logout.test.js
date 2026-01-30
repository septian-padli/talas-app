/**
 * Integration Test: Auth Logout Endpoint
 * Tests the POST /api/auth/logout endpoint.
 */
const request = require('supertest');
const app = require('../../src/app');
const prisma = require('../../src/utils/prisma');
const { hashPassword } = require('../../src/utils/password');

describe('POST /api/auth/logout', () => {
  // Test user credentials
  const testUser = {
    email: 'logout.test@example.com',
    password: 'SecurePassword123!',
    username: 'logouttest',
    name: 'Logout Test User'
  };

  /**
   * Setup: Create a user before running tests
   */
  beforeEach(async () => {
    const hashedPassword = await hashPassword(testUser.password);
    await prisma.user.create({
      data: {
        email: testUser.email,
        password: hashedPassword,
        username: testUser.username,
        name: testUser.name
      }
    });
  });

  /**
   * Helper: Login and get cookies
   */
  async function loginAndGetCookies() {
    const loginResponse = await request(app)
      .post('/api/auth/login')
      .send({
        email: testUser.email,
        password: testUser.password
      });

    return loginResponse.headers['set-cookie'];
  }

  /**
   * Test Case: Successful logout with active session
   */
  describe('Success Cases', () => {
    it('should logout successfully and clear cookies', async () => {
      // Step 1: Login to get session
      const loginCookies = await loginAndGetCookies();
      expect(loginCookies).toBeDefined();

      // Verify refresh token exists in database before logout
      const tokensBefore = await prisma.refreshToken.count();
      expect(tokensBefore).toBeGreaterThan(0);

      // Step 2: Logout
      const response = await request(app)
        .post('/api/auth/logout')
        .set('Cookie', loginCookies)
        .expect('Content-Type', /json/);

      // Assert status code
      expect(response.status).toBe(200);

      // Assert response structure
      expect(response.body).toHaveProperty('code', 200);
      expect(response.body).toHaveProperty('success', true);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toMatch(/logout|berhasil/i);

      // Verify cookies are cleared (Set-Cookie header should have expired tokens)
      const responseCookies = response.headers['set-cookie'];
      expect(responseCookies).toBeDefined();

      // Check that accessToken and refreshToken cookies are cleared
      // Cleared cookies typically have Max-Age=0 or Expires in the past
      const accessTokenClear = responseCookies.find(c => c.startsWith('accessToken='));
      const refreshTokenClear = responseCookies.find(c => c.startsWith('refreshToken='));
      
      if (accessTokenClear) {
        // Cookie should be empty or have past expiry
        expect(accessTokenClear).toMatch(/accessToken=;|Max-Age=0|Expires=Thu, 01 Jan 1970/i);
      }
      if (refreshTokenClear) {
        expect(refreshTokenClear).toMatch(/refreshToken=;|Max-Age=0|Expires=Thu, 01 Jan 1970/i);
      }

      // Verify refresh token is removed from database
      const tokensAfter = await prisma.refreshToken.count();
      expect(tokensAfter).toBe(0);
    });

    it('should invalidate refresh token after logout', async () => {
      // Step 1: Login
      const loginCookies = await loginAndGetCookies();

      // Step 2: Logout
      await request(app)
        .post('/api/auth/logout')
        .set('Cookie', loginCookies);

      // Step 3: Try to refresh with old cookies (refresh token should be deleted from DB)
      const refreshResponse = await request(app)
        .post('/api/auth/refresh')
        .set('Cookie', loginCookies);

      // Refresh should fail because token was deleted from database
      expect(refreshResponse.status).toBe(403);
      expect(refreshResponse.body.success).toBe(false);
    });
  });

  /**
   * Test Case: Failed logout without session/token
   */
  describe('Failure Cases - No Session', () => {
    it('should return 401 when no token is provided', async () => {
      const response = await request(app)
        .post('/api/auth/logout')
        .expect('Content-Type', /json/);

      // Assert status code
      expect(response.status).toBe(401);

      // Assert response structure
      expect(response.body).toHaveProperty('success', false);
      expect(response.body).toHaveProperty('message');
    });

    it('should return 401/403 when invalid token is provided', async () => {
      const response = await request(app)
        .post('/api/auth/logout')
        .set('Cookie', 'accessToken=invalid.token.here; refreshToken=also.invalid')
        .expect('Content-Type', /json/);

      // Assert status code (401 or 403)
      expect([401, 403]).toContain(response.status);
      expect(response.body.success).toBe(false);
    });
  });
});
