/**
 * Integration Test: Auth Register Endpoint
 * Tests the POST /api/auth/register endpoint.
 */
const request = require('supertest');
const app = require('../../src/app');
const prisma = require('../../src/utils/prisma');

describe('POST /api/auth/register', () => {
  /**
   * Test Case: Successful registration with valid data
   */
  describe('Success Cases', () => {
    it('should register a new user and return 201 with user data (without password)', async () => {
      const newUser = {
        email: 'user.baru@test.com',
        password: 'SecurePassword123!',
        username: 'userbaru',
        name: 'User Baru Test'
      };

      const response = await request(app)
        .post('/api/auth/register')
        .send(newUser)
        .expect('Content-Type', /json/);

      // Assert status code
      expect(response.status).toBe(201);

      // Assert response structure
      expect(response.body).toHaveProperty('code', 201);
      expect(response.body).toHaveProperty('success', true);
      expect(response.body).toHaveProperty('data');
      expect(response.body.data).toHaveProperty('user');

      // Assert user data returned
      const returnedUser = response.body.data.user;
      expect(returnedUser).toHaveProperty('id');
      expect(returnedUser).toHaveProperty('email', newUser.email);
      expect(returnedUser).toHaveProperty('username', newUser.username);
      expect(returnedUser).toHaveProperty('name', newUser.name);

      // CRITICAL: Password should NOT be returned
      expect(returnedUser).not.toHaveProperty('password');

      // Verify user exists in database
      const dbUser = await prisma.user.findUnique({
        where: { email: newUser.email }
      });
      expect(dbUser).not.toBeNull();
      expect(dbUser.email).toBe(newUser.email);
      expect(dbUser.username).toBe(newUser.username);
      // Password in DB should be hashed (not plain text)
      expect(dbUser.password).not.toBe(newUser.password);
    });
  });

  /**
   * Test Case: Failed registration with duplicate email
   */
  describe('Failure Cases - Duplicate Email', () => {
    const existingUser = {
      email: 'user.lama@test.com',
      password: 'ExistingPassword123!',
      username: 'userlama',
      name: 'User Lama'
    };

    beforeEach(async () => {
      // Create existing user via API (simulates real registration)
      await request(app)
        .post('/api/auth/register')
        .send(existingUser);
    });

    it('should return 400 when registering with an already used email', async () => {
      const duplicateEmailUser = {
        email: existingUser.email, // Same email
        password: 'DifferentPassword123!',
        username: 'differentusername',
        name: 'Different Name'
      };

      const response = await request(app)
        .post('/api/auth/register')
        .send(duplicateEmailUser)
        .expect('Content-Type', /json/);

      // Assert status code (based on controller: 400)
      expect(response.status).toBe(400);

      // Assert response structure
      expect(response.body).toHaveProperty('code', 400);
      expect(response.body).toHaveProperty('success', false);
      expect(response.body).toHaveProperty('errors');

      // Assert error message mentions email is already used
      const errors = response.body.errors;
      expect(Array.isArray(errors)).toBe(true);
      expect(errors.length).toBeGreaterThan(0);
      
      const emailError = errors.find(err => err.field === 'email');
      expect(emailError).toBeDefined();
      expect(emailError.message).toMatch(/email/i);
    });

    it('should return 400 when registering with an already used username', async () => {
      const duplicateUsernameUser = {
        email: 'new.email@test.com',
        password: 'DifferentPassword123!',
        username: existingUser.username, // Same username
        name: 'Different Name'
      };

      const response = await request(app)
        .post('/api/auth/register')
        .send(duplicateUsernameUser)
        .expect('Content-Type', /json/);

      // Assert status code
      expect(response.status).toBe(400);

      // Assert error mentions username
      const errors = response.body.errors;
      expect(Array.isArray(errors)).toBe(true);
      
      const usernameError = errors.find(err => err.field === 'username');
      expect(usernameError).toBeDefined();
      expect(usernameError.message).toMatch(/username/i);
    });
  });

  /**
   * Test Case: Validation errors
   */
  describe('Failure Cases - Validation Errors', () => {
    it('should return 400 when email format is invalid', async () => {
      const invalidUser = {
        email: 'invalid-email-format',
        password: 'SecurePassword123!',
        username: 'validuser',
        name: 'Valid Name'
      };

      const response = await request(app)
        .post('/api/auth/register')
        .send(invalidUser);

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
      expect(response.body.errors).toBeDefined();
    });

    it('should return 400 when required fields are missing', async () => {
      const incompleteUser = {
        email: 'incomplete@test.com'
        // Missing: password, username, name
      };

      const response = await request(app)
        .post('/api/auth/register')
        .send(incompleteUser);

      expect(response.status).toBe(400);
      expect(response.body.success).toBe(false);
      expect(response.body.errors).toBeDefined();
      expect(response.body.errors.length).toBeGreaterThan(0);
    });
  });
});
