/**
 * Integration Test: User Followers Endpoint
 * Tests the GET /api/users/:id/followers endpoint.
 */
const request = require('supertest');
const app = require('../../src/app');
const prisma = require('../../src/utils/prisma');
const { hashPassword } = require('../../src/utils/password');
const { randomUUID } = require('crypto');

describe('GET /api/users/:id/followers', () => {
  let userA, userB, loginCookies;

  /**
   * Setup: Create users and login
   */
  beforeEach(async () => {
    const hashedPassword = await hashPassword('TestPassword123!');

    // Create User A (the follower)
    userA = await prisma.user.create({
      data: {
        email: 'usera.followers@test.com',
        password: hashedPassword,
        username: 'usera_followers',
        name: 'User A Followers Test'
      }
    });

    // Create User B (the one being followed)
    userB = await prisma.user.create({
      data: {
        email: 'userb.followers@test.com',
        password: hashedPassword,
        username: 'userb_followers',
        name: 'User B Followers Test'
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
   * Test Case: Successfully get followers of a valid user
   */
  describe('Success Cases', () => {
    it('should return 200 with followers list when user has followers', async () => {
      // Create follow relationship: User A follows User B
      await prisma.follow.create({
        data: {
          followerId: userA.id,
          followingId: userB.id
        }
      });

      // Update counts
      await prisma.user.update({
        where: { id: userB.id },
        data: { followersCount: 1 }
      });

      // Get followers of User B (should include User A)
      const response = await request(app)
        .get(`/api/users/${userB.id}/followers`)
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

      // Assert User A is in followers list
      const followers = response.body.data.users;
      expect(Array.isArray(followers)).toBe(true);
      expect(followers.length).toBe(1);
      expect(followers[0]).toHaveProperty('id', userA.id);
      expect(followers[0]).toHaveProperty('username', userA.username);
      expect(followers[0]).toHaveProperty('name', userA.name);
      expect(followers[0]).toHaveProperty('is_following');

      // Assert pagination meta
      const meta = response.body.data.meta;
      expect(meta).toHaveProperty('has_next', false);
      expect(meta).toHaveProperty('limit');
    });

    it('should return empty list when user has no followers', async () => {
      const response = await request(app)
        .get(`/api/users/${userB.id}/followers`)
        .set('Cookie', loginCookies)
        .expect('Content-Type', /json/);

      expect(response.status).toBe(200);
      expect(response.body.data.users).toEqual([]);
    });
  });

  /**
   * Test Case: Failed - User not found
   */
  describe('Failure Cases - User Not Found', () => {
    it('should return 200 with empty list when user ID does not exist (soft behavior)', async () => {
      // Generate random UUID that doesn't exist
      const nonExistentId = randomUUID();

      const response = await request(app)
        .get(`/api/users/${nonExistentId}/followers`)
        .set('Cookie', loginCookies)
        .expect('Content-Type', /json/);

      // Current implementation returns 200 with empty list
      // (Not 404 because it just returns empty followers)
      expect(response.status).toBe(200);
      expect(response.body.data.users).toEqual([]);
    });
  });

  /**
   * Test Case: Failed - No authentication
   */
  describe('Failure Cases - Unauthorized', () => {
    it('should return 401 when no token is provided', async () => {
      const response = await request(app)
        .get(`/api/users/${userB.id}/followers`)
        .expect('Content-Type', /json/);

      expect(response.status).toBe(401);
      expect(response.body.success).toBe(false);
    });
  });
});
