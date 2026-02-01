/**
 * Integration Test: PATCH /api/notifications/read-all
 * Skenario: User A memiliki 5 notifikasi unread dan 2 notifikasi read.
 */
const request = require('supertest');
const app = require('../../src/app');
const prisma = require('../../src/utils/prisma');
const { hashPassword } = require('../../src/utils/password');

describe('PATCH /api/notifications/read-all', () => {
  const testUser = {
    email: 'notif.readall@example.com',
    password: 'NotifReadAll123!',
    username: 'notifreadalluser',
    name: 'Notif ReadAll User',
    bio: 'Test bio for notifications read-all endpoint'
  };

  let createdUser, loginCookies, notifIds;

  beforeEach(async () => {
    // Cleanup
    await prisma.notification.deleteMany();
    await prisma.user.deleteMany();

    // Create user
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

    // Login to get cookie
    const loginResponse = await request(app)
      .post('/api/auth/login')
      .send({
        email: testUser.email,
        password: testUser.password
      });
    loginCookies = loginResponse.headers['set-cookie'];

    // Seed 5 unread + 2 read notifications
    notifIds = [];
    for (let i = 0; i < 5; i++) {
      const notif = await prisma.notification.create({
        data: {
          userId: createdUser.id,
          type: 'LIKE',
          isRead: false,
          createdAt: new Date(Date.now() - i * 1000),
          data: { actor: { id: `actor-${i}` }, entity: { type: 'SHOWCASE', id: `showcase-${i}` } }
        }
      });
      notifIds.push(notif.id);
    }
    for (let i = 5; i < 7; i++) {
      await prisma.notification.create({
        data: {
          userId: createdUser.id,
          type: 'LIKE',
          isRead: true,
          createdAt: new Date(Date.now() - i * 1000),
          data: { actor: { id: `actor-${i}` }, entity: { type: 'SHOWCASE', id: `showcase-${i}` } }
        }
      });
    }
  });

  afterEach(async () => {
    await prisma.refreshToken?.deleteMany?.();
    await prisma.notification?.deleteMany?.();
    await prisma.user?.deleteMany?.();
  });

  it('should mark all unread notifications as read and return updated count', async () => {
    const res = await request(app)
      .patch('/api/notifications/read-all')
      .set('Cookie', loginCookies)
      .expect(200);

    // Response contract
    expect(res.body.code).toBe(200);
    expect(res.body.success).toBe(true);
    expect(res.body.data).toHaveProperty('updated_count', 5);
    expect(res.body.errors).toBeNull();

    // DB check: all notifications for user are read
    const unread = await prisma.notification.count({ where: { userId: createdUser.id, isRead: false } });
    expect(unread).toBe(0);
  });

  it('should return 401 if not authenticated and not update any notification', async () => {
    // Pastikan ada unread sebelum request
    const unreadBefore = await prisma.notification.count({ where: { userId: createdUser.id, isRead: false } });
    expect(unreadBefore).toBe(5);

    const res = await request(app)
      .patch('/api/notifications/read-all')
      // Tidak set Cookie
      .expect(401);

    expect(res.body.code).toBe(401);
    expect(res.body.success).toBe(false);
    // Data boleh null atau tidak ada
    expect(res.body.data === null || res.body.data === undefined).toBe(true);
    expect(Array.isArray(res.body.errors)).toBe(true);
    expect(res.body.errors[0]).toHaveProperty('field', 'auth');

    // DB check: unread tetap sama
    const unreadAfter = await prisma.notification.count({ where: { userId: createdUser.id, isRead: false } });
    expect(unreadAfter).toBe(5);
  });
});
