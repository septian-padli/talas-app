/**
 * Integration Test: Auth Login Endpoint
 * Tests the POST /api/auth/login endpoint.
 */
const request = require('supertest');
const app = require('../../src/app');
const prisma = require('../../src/utils/prisma');
const { hashPassword } = require('../../src/utils/password');

describe('POST /api/auth/login', () => {
  // Test user credentials
  const testUser = {
    email: 'login.test@example.com',
    password: 'SecurePassword123!',
    username: 'logintest',
    name: 'Login Test User'
  };

  /**
   * Setup: Create a user with hashed password before running login tests
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
   * Test Case: Successful login with valid credentials
   */
  describe('Success Cases', () => {
    it('should login successfully and return 200 with user data and cookies', async () => {
      const response = await request(app)
        .post('/api/auth/login')
        .send({
          email: testUser.email,
          password: testUser.password
        })
        .expect('Content-Type', /json/);

      // Assert status code
      expect(response.status).toBe(200);

      // Assert response structure
      expect(response.body).toHaveProperty('code', 200);
      expect(response.body).toHaveProperty('success', true);
      expect(response.body).toHaveProperty('data');
      expect(response.body.data).toHaveProperty('user');

      // Assert user data returned
      const returnedUser = response.body.data.user;
      expect(returnedUser).toHaveProperty('id');
      expect(returnedUser).toHaveProperty('email', testUser.email);
      expect(returnedUser).toHaveProperty('username', testUser.username);
      expect(returnedUser).toHaveProperty('name', testUser.name);

      // CRITICAL: Password should NOT be returned
      expect(returnedUser).not.toHaveProperty('password');

      // Verify Set-Cookie headers contain tokens
      const cookies = response.headers['set-cookie'];
      expect(cookies).toBeDefined();
      expect(Array.isArray(cookies)).toBe(true);

      // Check for accessToken cookie
      const accessTokenCookie = cookies.find(cookie => cookie.startsWith('accessToken='));
      expect(accessTokenCookie).toBeDefined();
      expect(accessTokenCookie).toMatch(/HttpOnly/i);

      // Check for refreshToken cookie
      const refreshTokenCookie = cookies.find(cookie => cookie.startsWith('refreshToken='));
      expect(refreshTokenCookie).toBeDefined();
      expect(refreshTokenCookie).toMatch(/HttpOnly/i);

      // Verify refresh token is saved in database
      const savedRefreshToken = await prisma.refreshToken.findFirst({
        where: {
          user: { email: testUser.email }
        }
      });
      expect(savedRefreshToken).not.toBeNull();
    });
  });

  /**
   * Test Case: Failed login with wrong password
   */
  describe('Failure Cases - Invalid Credentials', () => {
    it('should return 401 when password is incorrect', async () => {
      const response = await request(app)
        .post('/api/auth/login')
        .send({
          email: testUser.email,
          password: 'WrongPassword123!'
        })
        .expect('Content-Type', /json/);

      // Assert status code
      expect(response.status).toBe(401);

      // Assert response structure
      expect(response.body).toHaveProperty('code', 401);
      expect(response.body).toHaveProperty('success', false);
      expect(response.body).toHaveProperty('message');

      // Message should indicate invalid credentials
      expect(response.body.message).toMatch(/password|salah|invalid/i);

      // No cookies should be set
      const cookies = response.headers['set-cookie'];
      expect(cookies).toBeUndefined();
    });

    it('should return 401 when email does not exist', async () => {
      const response = await request(app)
        .post('/api/auth/login')
        .send({
          email: 'nonexistent@example.com',
          password: testUser.password
        })
        .expect('Content-Type', /json/);

      // Assert status code
      expect(response.status).toBe(401);

      // Assert response structure
      expect(response.body).toHaveProperty('code', 401);
      expect(response.body).toHaveProperty('success', false);
    });
  });

  /**
   * Test Case: Validation errors
   */
  describe('Failure Cases - Validation Errors', () => {
    it('should return 400 when email is missing', async () => {
      const response = await request(app)
        .post('/api/auth/login')
        .send({
          password: testUser.password
        });

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
    });

    it('should return 400 when password is missing', async () => {
      const response = await request(app)
        .post('/api/auth/login')
        .send({
          email: testUser.email
        });

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
    });
  });
});
