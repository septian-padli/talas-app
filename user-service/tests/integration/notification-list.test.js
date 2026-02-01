/**
 * Integration Test: User Profile Me Endpoint
 * Tests the GET /api/users/me endpoint.
 */
const request = require('supertest');
const app = require('../../src/app');
const prisma = require('../../src/utils/prisma');
const { hashPassword } = require('../../src/utils/password');

describe('GET /api/notifications', () => {
  const testUser = {
    email: 'notif.test@example.com',
    password: 'NotifPassword123!',
    username: 'notifuser',
    name: 'Notif Test User',
    bio: 'Test bio for notifications endpoint'
  };

  let createdUser, loginCookies;

  beforeEach(async () => {
    // Bersihkan data
    await prisma.notification.deleteMany();
    await prisma.user.deleteMany();

    // Buat user
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

    // Login untuk dapatkan cookie
    const loginResponse = await request(app)
      .post('/api/auth/login')
      .send({
        email: testUser.email,
        password: testUser.password
      });
    loginCookies = loginResponse.headers['set-cookie'];

    // Seed 20 notifikasi dengan timestamp berbeda
    const now = new Date();
    for (let i = 0; i < 20; i++) {
      await prisma.notification.create({
        data: {
          userId: createdUser.id,
          type: i % 2 === 0 ? 'LIKE' : 'COMMENT',
          isRead: i % 3 === 0,
          createdAt: new Date(now.getTime() - i * 60000),
          data: {
            actor: { id: `actor-${i}`, username: `actor${i}` },
            entity: { type: 'SHOWCASE', id: `showcase-${i}`, title: `Item ${i + 1}`, slug: `item-${i + 1}` },
          },
        },
      });
    }
  });

  afterEach(async () => {
    await prisma.refreshToken?.deleteMany?.();
    await prisma.notification?.deleteMany?.();
    await prisma.user?.deleteMany?.();
  });

  it('should return first page with correct pagination and structure', async () => {
    const res = await request(app)
      .get('/api/notifications?limit=10')
      .set('Cookie', loginCookies)
      .expect(200);

    expect(res.body.code).toBe(200);
    expect(res.body.success).toBe(true);
    expect(Array.isArray(res.body.data.notifications)).toBe(true);
    expect(res.body.data.notifications.length).toBe(10);
    // Newest first
    expect(res.body.data.notifications[0].data.entity.title).toBe('Item 1');
    expect(res.body.data.notifications[9].data.entity.title).toBe('Item 10');
    expect(res.body.data.pagination.has_next).toBe(true);
    expect(typeof res.body.data.pagination.next_cursor).toBe('string');
    // Structure check
    expect(typeof res.body.data.notifications[0].data).toBe('object');
  });
});
