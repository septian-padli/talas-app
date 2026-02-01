/**
 * Integration Test: Auth Refresh Token Endpoint
 * Tests the POST /api/auth/refresh endpoint.
 */
const request = require('supertest');
const app = require('../../src/app');
const prisma = require('../../src/utils/prisma');
const { hashPassword } = require('../../src/utils/password');

describe('POST /api/auth/refresh', () => {
  // Test user credentials
  const testUser = {
    email: 'refresh.test@example.com',
    password: 'SecurePassword123!',
    username: 'refreshtest',
    name: 'Refresh Test User'
  };

  /**
   * Setup: Create a user before running refresh token tests
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

  afterEach(async () => {
    await prisma.refreshToken?.deleteMany?.();
    await prisma.notification?.deleteMany?.();
    await prisma.user?.deleteMany?.();
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
   * Helper: Extract specific cookie value from Set-Cookie array
   */
  function extractCookie(cookies, name) {
    if (!cookies) return null;
    const cookie = cookies.find(c => c.startsWith(`${name}=`));
    if (!cookie) return null;
    // Extract just the cookie name=value part
    return cookie.split(';')[0];
  }

  /**
   * Test Case: Successful token refresh with valid refresh token
   */
  describe('Success Cases', () => {
    it('should refresh token and return 200 with new accessToken cookie', async () => {
      // Step 1: Login to get valid refresh token
      const cookies = await loginAndGetCookies();
      expect(cookies).toBeDefined();

      // Step 2: Extract refresh token cookie
      const refreshTokenCookie = extractCookie(cookies, 'refreshToken');
      expect(refreshTokenCookie).toBeDefined();

      // Step 3: Call refresh endpoint with cookie
      const response = await request(app)
        .post('/api/auth/refresh')
        .set('Cookie', cookies)
        .expect('Content-Type', /json/);

      // Assert status code
      expect(response.status).toBe(200);

      // Assert response structure
      expect(response.body).toHaveProperty('code', 200);
      expect(response.body).toHaveProperty('success', true);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toMatch(/refresh|token/i);

      // Verify new accessToken cookie is set
      const newCookies = response.headers['set-cookie'];
      expect(newCookies).toBeDefined();
      
      const newAccessTokenCookie = newCookies.find(c => c.startsWith('accessToken='));
      expect(newAccessTokenCookie).toBeDefined();
      expect(newAccessTokenCookie).toMatch(/HttpOnly/i);
    });
  });

  /**
   * Test Case: Failed refresh with invalid/no token
   */
  describe('Failure Cases - Invalid Token', () => {
    it('should return 401 when no refresh token cookie is provided', async () => {
      const response = await request(app)
        .post('/api/auth/refresh')
        .expect('Content-Type', /json/);

      // Assert status code (401 = no token)
      expect(response.status).toBe(401);

      // Assert response structure
      expect(response.body).toHaveProperty('code', 401);
      expect(response.body).toHaveProperty('success', false);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toMatch(/unauthorized|no.*token/i);
    });

    it('should return 403 when refresh token is invalid/tampered', async () => {
      // Send invalid/tampered token
      const response = await request(app)
        .post('/api/auth/refresh')
        .set('Cookie', 'refreshToken=this.is.an.invalid.token.that.should.be.rejected')
        .expect('Content-Type', /json/);

      // Assert status code (403 = invalid token)
      expect(response.status).toBe(403);

      // Assert response structure
      expect(response.body).toHaveProperty('code', 403);
      expect(response.body).toHaveProperty('success', false);
      expect(response.body).toHaveProperty('message');
      expect(response.body.message).toMatch(/forbidden|invalid/i);
    });

    it('should return 403 when refresh token is not in database (revoked)', async () => {
      // Step 1: Login to get valid refresh token
      const cookies = await loginAndGetCookies();

      // Step 2: Delete all refresh tokens from database (simulate revocation)
      await prisma.refreshToken.deleteMany({});

      // Step 3: Try to refresh with the now-revoked token
      const response = await request(app)
        .post('/api/auth/refresh')
        .set('Cookie', cookies)
        .expect('Content-Type', /json/);

      // Assert status code (403 = token revoked)
      expect(response.status).toBe(403);
      expect(response.body.success).toBe(false);
    });
  });
});
