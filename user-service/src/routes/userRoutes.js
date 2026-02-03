const express = require('express');
const router = express.Router();
const { protect } = require('../middlewares/authMiddleware');
const userController = require('../controllers/userController');
const upload = require("../middlewares/upload");

// All routes here start with /api/users
router.get('/me', protect, userController.getMyProfile);
router.patch('/me', protect, userController.updateUserProfile); // UPDATE PROFILE
router.get('/:username', protect, userController.getUserByUsername);

// PATCH /me/avatar (Edit Profile Picture)
router.patch(
	"/me/avatar",
	protect,
	upload.single("avatar"),
	userController.updateAvatar
);

router.post('/:id/follow', protect, userController.toggleFollow);
router.get('/:id/followers', protect, userController.getUserFollowers);
router.get('/:id/following', protect, userController.getUserFollowing);

module.exports = router;
