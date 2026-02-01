/**
 * Integration Test: User Public Profile Endpoint
 * Tests the GET /api/users/:username endpoint.
 */
const request = require('supertest');
const app = require('../../src/app');
const prisma = require('../../src/utils/prisma');
const { hashPassword } = require('../../src/utils/password');

describe('GET /api/users/:username', () => {
  let testUser, loginCookies;

  afterEach(async () => {
    await prisma.follow?.deleteMany?.();
    await prisma.refreshToken?.deleteMany?.();
    await prisma.notification?.deleteMany?.();
    await prisma.user?.deleteMany?.();
  });

  /**
   * Setup: Create users and login
   */
  beforeEach(async () => {
    const hashedPassword = await hashPassword('TestPassword123!');

    // Create a user with public profile data
    testUser = await prisma.user.create({
      data: {
        email: 'dev.talas@test.com',
        password: hashedPassword,
        username: 'dev_talas',
        name: 'Developer Talas',
        bio: 'This is a public bio visible to everyone',
        avatarUrl: 'https://example.com/avatar.jpg',
        jobTitle: 'Senior Developer'
      }
    });

    // Create a viewer user and login
    const viewerUser = await prisma.user.create({
      data: {
        email: 'viewer@test.com',
        password: hashedPassword,
        username: 'viewer_user',
        name: 'Viewer User'
      }
    });

    const loginResponse = await request(app)
      .post('/api/auth/login')
      .send({
        email: viewerUser.email,
        password: 'TestPassword123!'
      });

    loginCookies = loginResponse.headers['set-cookie'];
  });

  /**
   * Test Case: Successfully view public profile of another user
   */
  describe('Success Cases', () => {
    it('should return 200 with public profile data for valid username', async () => {
      const response = await request(app)
        .get('/api/users/dev_talas')
        .set('Cookie', loginCookies)
        .expect('Content-Type', /json/);

      // Assert status code
      expect(response.status).toBe(200);

      // Assert response structure
      expect(response.body).toHaveProperty('code', 200);
      expect(response.body).toHaveProperty('success', true);
      expect(response.body).toHaveProperty('data');
      expect(response.body.data).toHaveProperty('user');

      // Assert public data is visible
      const returnedUser = response.body.data.user;
      expect(returnedUser).toHaveProperty('id', testUser.id);
      expect(returnedUser).toHaveProperty('username', 'dev_talas');
      expect(returnedUser).toHaveProperty('name', 'Developer Talas');
      expect(returnedUser).toHaveProperty('bio', 'This is a public bio visible to everyone');
      expect(returnedUser).toHaveProperty('avatar_url');
      expect(returnedUser).toHaveProperty('followers_count');
      expect(returnedUser).toHaveProperty('following_count');
      expect(returnedUser).toHaveProperty('is_following');

      // CRITICAL: Private data should NOT be exposed
      expect(returnedUser).not.toHaveProperty('email');
      expect(returnedUser).not.toHaveProperty('password');
    });

    it('should return is_following status correctly', async () => {
      // First, follow the user
      await request(app)
        .post(`/api/users/${testUser.id}/follow`)
        .set('Cookie', loginCookies);

      // Then, get profile
      const response = await request(app)
        .get('/api/users/dev_talas')
        .set('Cookie', loginCookies);

      expect(response.status).toBe(200);
      expect(response.body.data.user.is_following).toBe(true);
    });
  });

  /**
   * Test Case: Username not found
   */
  describe('Failure Cases - User Not Found', () => {
    it('should return 404 when username does not exist', async () => {
      const response = await request(app)
        .get('/api/users/hantu_belau')
        .set('Cookie', loginCookies)
        .expect('Content-Type', /json/);

      // Assert status code
      expect(response.status).toBe(404);

      // Assert response structure
      expect(response.body).toHaveProperty('code', 404);
      expect(response.body).toHaveProperty('success', false);
      expect(response.body).toHaveProperty('message');

      // Message should indicate user not found
      expect(response.body.message).toMatch(/tidak ditemukan|not found/i);
    });
  });

  /**
   * Test Case: Unauthorized access
   */
  describe('Failure Cases - Unauthorized', () => {
    it('should return 401 when no token is provided', async () => {
      const response = await request(app)
        .get('/api/users/dev_talas')
        .expect('Content-Type', /json/);

      expect(response.status).toBe(401);
      expect(response.body.success).toBe(false);
    });
  });
});
