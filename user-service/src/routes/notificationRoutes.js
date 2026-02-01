const express = require('express');
const router = express.Router();
const { protect } = require('../middlewares/authMiddleware');
const notificationController = require('../controllers/notificationController');

// All routes here start with /api/notifications

router.get('/', protect, notificationController.listNotifications);
router.get('/count', protect, notificationController.getNotificationCount);
router.patch('/read', protect, notificationController.markNotificationsRead);
router.patch('/read-all', protect, notificationController.markAllNotificationsRead);

module.exports = router;
