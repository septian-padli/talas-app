/**
 * Integration Test: User Follow Toggle Endpoint
 * Tests the POST /api/users/:id/follow endpoint.
 */
const request = require('supertest');
const app = require('../../src/app');
const prisma = require('../../src/utils/prisma');
const { hashPassword } = require('../../src/utils/password');

describe('POST /api/users/:id/follow', () => {
  let userA, userB, userACookies;

  /**
   * Setup: Create two users and login as User A
   */
  beforeEach(async () => {
    const hashedPassword = await hashPassword('TestPassword123!');

    // Create User A (the one who will follow)
    userA = await prisma.user.create({
      data: {
        email: 'usera.follow@test.com',
        password: hashedPassword,
        username: 'usera_follow',
        name: 'User A Follow Test'
      }
    });

    // Create User B (the target to be followed)
    userB = await prisma.user.create({
      data: {
        email: 'userb.follow@test.com',
        password: hashedPassword,
        username: 'userb_follow',
        name: 'User B Follow Test'
      }
    });

    // Login as User A
    const loginResponse = await request(app)
      .post('/api/auth/login')
      .send({
        email: userA.email,
        password: 'TestPassword123!'
      });

    userACookies = loginResponse.headers['set-cookie'];
  });

  /**
   * Test Case: Successfully follow another user
   */
  describe('Success Cases - Follow', () => {
    it('should follow another user and return 200 with followed status', async () => {
      const response = await request(app)
        .post(`/api/users/${userB.id}/follow`)
        .set('Cookie', userACookies)
        .expect('Content-Type', /json/);

      // Assert status code
      expect(response.status).toBe(200);

      // Assert response structure
      expect(response.body).toHaveProperty('code', 200);
      expect(response.body).toHaveProperty('success', true);
      expect(response.body).toHaveProperty('data');
      expect(response.body.data).toHaveProperty('is_following', true);

      // Verify follow relationship exists in database
      const followRecord = await prisma.follow.findUnique({
        where: {
          followerId_followingId: {
            followerId: userA.id,
            followingId: userB.id
          }
        }
      });
      expect(followRecord).not.toBeNull();

      // Verify follower/following counts updated
      const updatedUserB = await prisma.user.findUnique({
        where: { id: userB.id }
      });
      expect(updatedUserB.followersCount).toBe(1);

      const updatedUserA = await prisma.user.findUnique({
        where: { id: userA.id }
      });
      expect(updatedUserA.followingCount).toBe(1);
    });
  });

  /**
   * Test Case: Successfully unfollow (toggle)
   */
  describe('Success Cases - Unfollow (Toggle)', () => {
    it('should unfollow when already following (toggle behavior)', async () => {
      // First, follow
      await request(app)
        .post(`/api/users/${userB.id}/follow`)
        .set('Cookie', userACookies);

      // Then, call follow again to unfollow (toggle)
      const response = await request(app)
        .post(`/api/users/${userB.id}/follow`)
        .set('Cookie', userACookies);

      expect(response.status).toBe(200);
      expect(response.body.data.is_following).toBe(false);

      // Verify follow relationship is deleted from database
      const followRecord = await prisma.follow.findUnique({
        where: {
          followerId_followingId: {
            followerId: userA.id,
            followingId: userB.id
          }
        }
      });
      expect(followRecord).toBeNull();
    });
  });

  /**
   * Test Case: Failed - Follow self
   */
  describe('Failure Cases - Self Follow', () => {
    it('should return 400 when trying to follow self', async () => {
      const response = await request(app)
        .post(`/api/users/${userA.id}/follow`)
        .set('Cookie', userACookies)
        .expect('Content-Type', /json/);

      // Assert status code
      expect(response.status).toBe(400);

      // Assert response structure
      expect(response.body).toHaveProperty('code', 400);
      expect(response.body).toHaveProperty('success', false);
      expect(response.body).toHaveProperty('message');

      // Message should indicate self-follow is not allowed
      expect(response.body.message).toMatch(/diri sendiri|yourself|self/i);

      // Verify no follow relationship was created
      const followRecord = await prisma.follow.findFirst({
        where: {
          followerId: userA.id,
          followingId: userA.id
        }
      });
      expect(followRecord).toBeNull();
    });
  });

  /**
   * Test Case: Failed - Target user not found
   */
  describe('Failure Cases - User Not Found', () => {
    it('should return 404 when target user does not exist', async () => {
      const { randomUUID } = require('crypto');
      const nonExistentId = randomUUID();

      const response = await request(app)
        .post(`/api/users/${nonExistentId}/follow`)
        .set('Cookie', userACookies)
        .expect('Content-Type', /json/);

      expect(response.status).toBe(404);
      expect(response.body.success).toBe(false);
      expect(response.body.message).toMatch(/tidak ditemukan|not found/i);
    });
  });

  /**
   * Test Case: Failed - Unauthorized
   */
  describe('Failure Cases - Unauthorized', () => {
    it('should return 401 when no token is provided', async () => {
      const response = await request(app)
        .post(`/api/users/${userB.id}/follow`)
        .expect('Content-Type', /json/);

      expect(response.status).toBe(401);
      expect(response.body.success).toBe(false);
    });
  });
});
