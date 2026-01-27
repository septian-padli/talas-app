const express = require('express');
const router = express.Router();
const internalController = require('../controllers/internalController');
const { verifyInternalKey } = require('../middlewares/internalMiddleware');

// PROTECT ALL ROUTES WITH SERVICE SECRET
router.use(verifyInternalKey);

// Define Routes
router.post('/users/bulk', internalController.getBulkUsers);
router.get('/users/:id/followers', internalController.getUserFollowers);

module.exports = router;
