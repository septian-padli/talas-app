/**
 * Mark all notifications as read for a user
 * @param {string} userId
 * @returns {Promise<number>} count of updated notifications
 */
async function markAllNotificationsAsRead(userId) {
  const result = await prisma.notification.updateMany({
    where: {
      userId,
      isRead: false,
    },
    data: { isRead: true },
  });
  return result.count;
}
const prisma = require('../utils/prisma');

/**
 * Get notifications for a user with cursor-based pagination.
 * @param {string} userId - User ID
 * @param {Object} params - Query params: { limit, cursor, status }
 * @returns {Promise<{notifications: Array, nextCursor: string|null, hasNext: boolean}>}
 */
async function getUserNotifications(userId, params = {}) {
  const limit = Math.min(parseInt(params.limit, 10) || 15, 50);
  let cursorDate = null;
  if (params.cursor) {
    try {
      const decoded = Buffer.from(params.cursor, 'base64').toString('utf8');
      cursorDate = new Date(decoded);
      if (isNaN(cursorDate.getTime())) cursorDate = null;
    } catch {
      cursorDate = null;
    }
  }

  const where = {
    userId,
    ...(params.status === 'unread' ? { isRead: false } : {}),
    ...(cursorDate ? { createdAt: { lt: cursorDate } } : {})
  };

  const notifications = await prisma.notification.findMany({
    where,
    orderBy: { createdAt: 'desc' },
    take: limit + 1,
  });

  let hasNext = false;
  let nextCursor = null;
  let items = notifications;
  if (notifications.length > limit) {
    hasNext = true;
    items = notifications.slice(0, limit);
    nextCursor = Buffer.from(items[items.length - 1].createdAt.toISOString()).toString('base64');
  }

  return {
    notifications: items,
    nextCursor,
    hasNext
  };
}

/**
 * Get unread notification count and breakdown by type for a user
 * @param {string} userId
 * @returns {Promise<{ total: number, breakdown: object }>} 
 */
async function getUnreadNotificationCount(userId) {
  // Aggregate unread notifications by type
  const result = await prisma.notification.groupBy({
    by: ['type'],
    where: {
      userId,
      isRead: false,
    },
    _count: { type: true },
  });
  // Calculate total and breakdown
  let total = 0;
  const breakdown = {};
  for (const row of result) {
    breakdown[row.type] = row._count.type;
    total += row._count.type;
  }
  return { total, breakdown };
}

/**
 * Batch mark notifications as read for a user
 * @param {string} userId
 * @param {string[]} notificationIds
 * @returns {Promise<number>} count of updated notifications
 */
async function markNotificationsAsRead(userId, notificationIds) {
  const result = await prisma.notification.updateMany({
    where: {
      userId,
      id: { in: notificationIds },
      isRead: false,
    },
    data: { isRead: true },
  });
  return result.count;
}

module.exports = {
  getUserNotifications,
  getUnreadNotificationCount,
  markNotificationsAsRead,
  markAllNotificationsAsRead,
};
