/**
 * Integration Test: User Following Endpoint
 * Tests the GET /api/users/:id/following endpoint.
 */
const request = require('supertest');
const app = require('../../src/app');
const prisma = require('../../src/utils/prisma');
const { hashPassword } = require('../../src/utils/password');

describe('GET /api/users/:id/following', () => {
  let userA, userB, loginCookies;

  /**
   * Setup: Create users and login
   */
  beforeEach(async () => {
    const hashedPassword = await hashPassword('TestPassword123!');

    // Create User A (will be following others)
    userA = await prisma.user.create({
      data: {
        email: 'usera.following@test.com',
        password: hashedPassword,
        username: 'usera_following',
        name: 'User A Following Test'
      }
    });

    // Create User B (will be followed by User A)
    userB = await prisma.user.create({
      data: {
        email: 'userb.following@test.com',
        password: hashedPassword,
        username: 'userb_following',
        name: 'User B Following Test'
      }
    });

    // Login as User A to get cookies
    const loginResponse = await request(app)
      .post('/api/auth/login')
      .send({
        email: userA.email,
        password: 'TestPassword123!'
      });

    loginCookies = loginResponse.headers['set-cookie'];
  });

  /**
   * Test Case: Successfully get following list of a valid user
   */
  describe('Success Cases', () => {
    it('should return 200 with following list when user is following others', async () => {
      // Create follow relationship: User A follows User B
      await prisma.follow.create({
        data: {
          followerId: userA.id,
          followingId: userB.id
        }
      });

      // Update counts
      await prisma.user.update({
        where: { id: userA.id },
        data: { followingCount: 1 }
      });

      // Get following list of User A (should include User B)
      const response = await request(app)
        .get(`/api/users/${userA.id}/following`)
        .set('Cookie', loginCookies)
        .expect('Content-Type', /json/);

      // Assert status code
      expect(response.status).toBe(200);

      // Assert response structure
      expect(response.body).toHaveProperty('code', 200);
      expect(response.body).toHaveProperty('success', true);
      expect(response.body).toHaveProperty('data');
      expect(response.body.data).toHaveProperty('users');
      expect(response.body.data).toHaveProperty('meta');

      // Assert User B is in following list
      const following = response.body.data.users;
      expect(Array.isArray(following)).toBe(true);
      expect(following.length).toBe(1);
      expect(following[0]).toHaveProperty('id', userB.id);
      expect(following[0]).toHaveProperty('username', userB.username);
      expect(following[0]).toHaveProperty('name', userB.name);
      expect(following[0]).toHaveProperty('is_following');
      expect(following[0]).toHaveProperty('followed_at');

      // Assert pagination meta
      const meta = response.body.data.meta;
      expect(meta).toHaveProperty('has_next', false);
      expect(meta).toHaveProperty('limit');
    });

    it('should return empty list when user is not following anyone', async () => {
      const response = await request(app)
        .get(`/api/users/${userA.id}/following`)
        .set('Cookie', loginCookies)
        .expect('Content-Type', /json/);

      expect(response.status).toBe(200);
      expect(response.body.data.users).toEqual([]);
    });
  });

  /**
   * Test Case: Failed - Invalid ID format
   */
  describe('Failure Cases - Invalid ID Format', () => {
    it('should return error when ID format is invalid (non-UUID)', async () => {
      // Send invalid ID format
      const response = await request(app)
        .get('/api/users/invalid-id-format/following')
        .set('Cookie', loginCookies)
        .expect('Content-Type', /json/);

      // Prisma throws error for invalid UUID format
      // This results in 400 (validation) or 500 (internal)
      expect([400, 500]).toContain(response.status);
      expect(response.body.success).toBe(false);
    });
  });

  /**
   * Test Case: Failed - No authentication
   */
  describe('Failure Cases - Unauthorized', () => {
    it('should return 401 when no token is provided', async () => {
      const response = await request(app)
        .get(`/api/users/${userA.id}/following`)
        .expect('Content-Type', /json/);

      expect(response.status).toBe(401);
      expect(response.body.success).toBe(false);
    });
  });
});
