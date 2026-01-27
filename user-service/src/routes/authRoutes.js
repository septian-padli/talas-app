const express = require('express');
const { register, login, refresh, logout, getMe } = require('../controllers/authController');
const { protect } = require('../middlewares/authMiddleware');

const router = express.Router();

// Public Routes
router.post('/register', register);
router.post('/login', login);
router.post('/refresh', refresh);

// Protected Routes
router.post('/logout', protect, logout);
router.get('/me', protect, getMe);

module.exports = router;
