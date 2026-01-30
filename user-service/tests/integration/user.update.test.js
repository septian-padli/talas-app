/**
 * Integration Test: User Profile Update Endpoint
 * Tests the PATCH /api/users/me endpoint.
 */
const request = require('supertest');
const app = require('../../src/app');
const prisma = require('../../src/utils/prisma');
const { hashPassword } = require('../../src/utils/password');

describe('PATCH /api/users/me', () => {
  // Test user credentials
  const testUser = {
    email: 'updateprofile.test@example.com',
    password: 'SecurePassword123!',
    username: 'updateprofiletest',
    name: 'Original Name',
    bio: 'Original bio text'
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
   * Test Case: Successfully update profile with valid data
   */
  describe('Success Cases', () => {
    it('should update profile and return 200 with updated data', async () => {
      const updateData = {
        name: 'Updated Name',
        bio: 'This is my updated bio text for testing'
      };

      const response = await request(app)
        .patch('/api/users/me')
        .set('Cookie', loginCookies)
        .send(updateData)
        .expect('Content-Type', /json/);

      // Assert status code
      expect(response.status).toBe(200);

      // Assert response structure
      expect(response.body).toHaveProperty('code', 200);
      expect(response.body).toHaveProperty('success', true);
      expect(response.body).toHaveProperty('data');
      expect(response.body.data).toHaveProperty('user');

      // Assert updated data in response
      const returnedUser = response.body.data.user;
      expect(returnedUser).toHaveProperty('name', updateData.name);
      expect(returnedUser).toHaveProperty('bio', updateData.bio);
      expect(returnedUser).toHaveProperty('id', createdUser.id);

      // Verify data is persisted in database
      const dbUser = await prisma.user.findUnique({
        where: { id: createdUser.id }
      });
      expect(dbUser.name).toBe(updateData.name);
      expect(dbUser.bio).toBe(updateData.bio);
    });

    it('should update jobTitle and avatarUrl successfully', async () => {
      const updateData = {
        jobTitle: 'Senior Developer',
        avatarUrl: 'https://example.com/avatar.jpg'
      };

      const response = await request(app)
        .patch('/api/users/me')
        .set('Cookie', loginCookies)
        .send(updateData);

      expect(response.status).toBe(200);
      expect(response.body.data.user).toHaveProperty('job_title', updateData.jobTitle);
      expect(response.body.data.user).toHaveProperty('avatar_url', updateData.avatarUrl);

      // Verify in database
      const dbUser = await prisma.user.findUnique({
        where: { id: createdUser.id }
      });
      expect(dbUser.jobTitle).toBe(updateData.jobTitle);
      expect(dbUser.avatarUrl).toBe(updateData.avatarUrl);
    });

    it('should update socialLinks successfully', async () => {
      const updateData = {
        socialLinks: [
          {
            social: 'GITHUB',
            link: 'https://github.com/testuser',
            username: 'testuser'
          },
          {
            social: 'LINKEDIN',
            link: 'https://linkedin.com/in/testuser'
          }
        ]
      };

      const response = await request(app)
        .patch('/api/users/me')
        .set('Cookie', loginCookies)
        .send(updateData);

      expect(response.status).toBe(200);
      expect(response.body.data.user).toHaveProperty('social_links');
      expect(response.body.data.user.social_links.length).toBe(2);

      // Verify GitHub link in response
      const githubLink = response.body.data.user.social_links.find(
        sl => sl.social === 'GITHUB'
      );
      expect(githubLink).toBeDefined();
      expect(githubLink.link).toBe('https://github.com/testuser');

      // Verify in database
      const dbSocialLinks = await prisma.socialLink.findMany({
        where: { userId: createdUser.id }
      });
      expect(dbSocialLinks.length).toBe(2);
    });
  });

  /**
   * Test Case: Failed update with invalid data format
   */
  describe('Failure Cases - Validation Errors', () => {
    it('should return 400 when name is too short', async () => {
      const invalidData = {
        name: 'A' // Min 2 characters required
      };

      const response = await request(app)
        .patch('/api/users/me')
        .set('Cookie', loginCookies)
        .send(invalidData)
        .expect('Content-Type', /json/);

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
      expect(response.body).toHaveProperty('errors');
      expect(Array.isArray(response.body.errors)).toBe(true);

      // Should indicate name field error
      const nameError = response.body.errors.find(err => err.field === 'name');
      expect(nameError).toBeDefined();
    });

    it('should return 400 when avatarUrl is not a valid URL', async () => {
      const invalidData = {
        avatarUrl: 'not-a-valid-url'
      };

      const response = await request(app)
        .patch('/api/users/me')
        .set('Cookie', loginCookies)
        .send(invalidData);

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);

      // Should indicate avatarUrl field error
      const urlError = response.body.errors.find(err => err.field === 'avatarUrl');
      expect(urlError).toBeDefined();
      expect(urlError.message).toMatch(/url|valid/i);
    });

    it('should return 400 when socialLinks has invalid platform', async () => {
      const invalidData = {
        socialLinks: [
          {
            social: 'INVALID_PLATFORM',
            link: 'https://example.com'
          }
        ]
      };

      const response = await request(app)
        .patch('/api/users/me')
        .set('Cookie', loginCookies)
        .send(invalidData);

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
      expect(response.body.errors.length).toBeGreaterThan(0);
    });

    it('should return 400 when socialLinks has invalid URL', async () => {
      const invalidData = {
        socialLinks: [
          {
            social: 'GITHUB',
            link: 'not-a-valid-url'
          }
        ]
      };

      const response = await request(app)
        .patch('/api/users/me')
        .set('Cookie', loginCookies)
        .send(invalidData);

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
    });
  });

  /**
   * Test Case: Failed - No authentication
   */
  describe('Failure Cases - Unauthorized', () => {
    it('should return 401 when no token is provided', async () => {
      const updateData = {
        name: 'New Name'
      };

      const response = await request(app)
        .patch('/api/users/me')
        .send(updateData)
        .expect('Content-Type', /json/);

      expect(response.status).toBe(401);
      expect(response.body.success).toBe(false);
    });
  });
});
