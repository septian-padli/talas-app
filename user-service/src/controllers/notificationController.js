const { getUserNotifications, getUnreadNotificationCount, markNotificationsAsRead, markAllNotificationsAsRead } = require('../services/notification.service');

/**
 * List notifications for the authenticated user (GET /notifications)
 */
async function listNotifications(req, res) {
  try {
    const userId = req.user.id;
    const { limit, cursor, status } = req.query;
    const result = await getUserNotifications(userId, { limit, cursor, status });

    const notifications = result.notifications.map(n => ({
      id: n.id,
      type: n.type,
      is_read: n.isRead,
      created_at: n.createdAt,
      data: n.data
    }));

    return res.json({
      code: 200,
      success: true,
      data: {
        notifications,
        pagination: {
          next_cursor: result.nextCursor,
          has_next: result.hasNext
        }
      }
    });
  } catch (err) {
    return res.status(500).json({
      code: 500,
      success: false,
      message: 'Failed to fetch notifications',
      errors: err.message || err
    });
  }
}


/**
 * Get unread notification count and breakdown (GET /notifications/count)
 */
async function getNotificationCount(req, res) {
  try {
    const userId = req.user.id;
    const result = await getUnreadNotificationCount(userId);
    return res.json({
      code: 200,
      success: true,
      data: {
        total: result.total,
        breakdown: result.breakdown
      }
    });
  } catch (err) {
    return res.status(500).json({
      code: 500,
      success: false,
      message: 'Failed to fetch notification count',
      errors: err.message || err
    });
  }
}


/**
 * Batch mark notifications as read (PATCH /notifications/read)
 */
async function markNotificationsRead(req, res) {
  try {
    const userId = req.user.id;
    const { notification_ids } = req.body;
    if (!Array.isArray(notification_ids) || notification_ids.length === 0) {
      return res.status(400).json({
        code: 400,
        success: false,
        data: null,
        errors: [
          {
            field: 'notification_ids',
            message: 'notification_ids wajib diisi dan harus berupa array'
          }
        ]
      });
    }
    const updatedCount = await markNotificationsAsRead(userId, notification_ids);
    return res.json({
      code: 200,
      success: true,
      data: {
        message: `${updatedCount} notifikasi berhasil ditandai sebagai sudah dibaca`,
        updated_ids: notification_ids
      },
      errors: null
    });
  } catch (err) {
    return res.status(500).json({
      code: 500,
      success: false,
      message: 'Failed to mark notifications as read',
      errors: err.message || err
    });
  }
}


/**
 * Mark all notifications as read (PATCH /notifications/read-all)
 */
async function markAllNotificationsRead(req, res) {
  try {
    const userId = req.user.id;
    const updatedCount = await markAllNotificationsAsRead(userId);
    return res.json({
      code: 200,
      success: true,
      data: {
        updated_count: updatedCount
      },
      errors: null
    });
  } catch (err) {
    return res.status(500).json({
      code: 500,
      success: false,
      message: 'Failed to mark all notifications as read',
      errors: err.message || err
    });
  }
}

module.exports = {
  listNotifications,
  getNotificationCount,
  markNotificationsRead,
  markAllNotificationsRead,
};
