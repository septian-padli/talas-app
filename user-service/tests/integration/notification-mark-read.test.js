/**
 * Integration Test: PATCH /api/notifications/read
 * Skenario: User A memiliki 3 notifikasi unread, ingin menandai 2 sebagai read.
 */
const request = require('supertest');
const app = require('../../src/app');
const prisma = require('../../src/utils/prisma');
const { hashPassword } = require('../../src/utils/password');

describe('PATCH /api/notifications/read', () => {
  const testUser = {
    email: 'notif.read@example.com',
    password: 'NotifRead123!',
    username: 'notifreaduser',
    name: 'Notif Read User',
    bio: 'Test bio for notifications read endpoint'
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

    // Seed 3 unread notifications
    notifIds = [];
    for (let i = 0; i < 3; i++) {
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
  });

  afterEach(async () => {
    await prisma.refreshToken?.deleteMany?.();
    await prisma.notification?.deleteMany?.();
    await prisma.user?.deleteMany?.();
  });

  it('should mark 2 notifications as read and leave the 3rd unread', async () => {
    // Pick 2 notification IDs to mark as read
    const toMark = notifIds.slice(0, 2);
    const toRemainUnread = notifIds[2];

    const res = await request(app)
      .patch('/api/notifications/read')
      .set('Cookie', loginCookies)
      .send({ notification_ids: toMark })
      .expect(200);

    // Response contract
    expect(res.body.code).toBe(200);
    expect(res.body.success).toBe(true);
    expect(res.body.data).toHaveProperty('message');
    expect(Array.isArray(res.body.data.updated_ids)).toBe(true);
    expect(res.body.data.updated_ids).toEqual(expect.arrayContaining(toMark));
    expect(res.body.errors).toBeNull();

    // DB check: 2 marked as read, 1 remains unread
    const notif1 = await prisma.notification.findUnique({ where: { id: toMark[0] } });
    const notif2 = await prisma.notification.findUnique({ where: { id: toMark[1] } });
    const notif3 = await prisma.notification.findUnique({ where: { id: toRemainUnread } });
    expect(notif1.isRead).toBe(true);
    expect(notif2.isRead).toBe(true);
    expect(notif3.isRead).toBe(false);
  });

  it('should return 400 if notification_ids is missing', async () => {
    const res = await request(app)
      .patch('/api/notifications/read')
      .set('Cookie', loginCookies)
      .send({})
      .expect(400);
    expect(res.body.code).toBe(400);
    expect(res.body.success).toBe(false);
    expect(res.body.data).toBeNull();
    expect(Array.isArray(res.body.errors)).toBe(true);
    expect(res.body.errors[0]).toHaveProperty('field', 'notification_ids');
    expect(res.body.errors[0].message).toMatch(/wajib diisi/i);
  });

  it('should return 400 if notification_ids is empty array', async () => {
    const res = await request(app)
      .patch('/api/notifications/read')
      .set('Cookie', loginCookies)
      .send({ notification_ids: [] })
      .expect(400);
    expect(res.body.code).toBe(400);
    expect(res.body.success).toBe(false);
    expect(res.body.data).toBeNull();
    expect(Array.isArray(res.body.errors)).toBe(true);
    expect(res.body.errors[0]).toHaveProperty('field', 'notification_ids');
    expect(res.body.errors[0].message).toMatch(/wajib diisi/i);
  });

  it('should return 400 if notification_ids is not an array', async () => {
    const res = await request(app)
      .patch('/api/notifications/read')
      .set('Cookie', loginCookies)
      .send({ notification_ids: 'not-an-array' })
      .expect(400);
    expect(res.body.code).toBe(400);
    expect(res.body.success).toBe(false);
    expect(res.body.data).toBeNull();
    expect(Array.isArray(res.body.errors)).toBe(true);
    expect(res.body.errors[0]).toHaveProperty('field', 'notification_ids');
    expect(res.body.errors[0].message).toMatch(/wajib diisi/i);
  });

  it('should return 400 if notification_ids is null', async () => {
    const res = await request(app)
      .patch('/api/notifications/read')
      .set('Cookie', loginCookies)
      .send({ notification_ids: null })
      .expect(400);
    expect(res.body.code).toBe(400);
    expect(res.body.success).toBe(false);
    expect(res.body.data).toBeNull();
    expect(Array.isArray(res.body.errors)).toBe(true);
    expect(res.body.errors[0]).toHaveProperty('field', 'notification_ids');
    expect(res.body.errors[0].message).toMatch(/wajib diisi/i);
  });
});
